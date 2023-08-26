package game

import (
	"fmt"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
	"math"
)

func NewGameInstanceWithMap(gameID string, mapFile string) *GameInstance {
	println(fmt.Sprintf("[GameInstance] '%s' created", gameID))
	g := &GameInstance{
		id:             gameID,
		mapFile:        mapFile,
		players:        make([]uint64, 0),
		playerFactions: make(map[uint64]*Faction),
		losMatrix:      make(map[uint64]map[uint64]bool),
		pressureMatrix: make(map[uint64]map[uint64]float64),
		playerUnits:    make(map[uint64][]uint64),
		units:          make(map[uint64]*UnitInstance),
		playersNeeded:  2,
		voxelMap:       voxel.NewMapFromFile(mapFile, nil, nil),
		mapMeta:        NewMapMetadataFromFile(mapFile + ".meta"),
		overwatch:      make(map[voxel.Int3][]*UnitInstance),
	}
	g.rules = NewDefaultRuleset(g)
	return g
}

type Ruleset struct {
	engine                    *GameInstance
	MaxPressureDistance       int32
	MaxOverwatchRange         uint
	OverwatchAccuracyModifier float64
	OverwatchDamageModifier   float64
	IsRangedAttackTurnEnding  bool
}

func NewDefaultRuleset(engine *GameInstance) *Ruleset {
	return &Ruleset{
		engine:                    engine,
		MaxPressureDistance:       4,
		MaxOverwatchRange:         20,
		OverwatchAccuracyModifier: 0.8, // 20% penalty for overwatch shots
		OverwatchDamageModifier:   1.1, // 10% bonus damage for overwatch shots
	}
}

type ShotAction interface {
	GetUnit() *UnitInstance
	GetAccuracyModifier() float64
}

func (r *Ruleset) GetShotAccuracy(action ShotAction) float64 {
	unit := action.GetUnit()
	unitAndWeaponAccuracy := unit.GetFreeAimAccuracy() // factors penalties for the unit and the weapon
	actionModifier := action.GetAccuracyModifier()     // currently only used by overwatch

	// pressure rule for the sniper rifle
	pressureModifier := 1.0
	if unit.GetWeapon().Definition.WeaponType == WeaponSniper {
		// add penalty for sniper shots under pressure
		pressureOnUnit := r.engine.GetTotalPressure(unit.UnitID())
		pressureOnUnit = util.Clamp(pressureOnUnit, 0.0, 1.0)
		pressureModifier = 1.0 - pressureOnUnit
	}

	return unitAndWeaponAccuracy * actionModifier * pressureModifier
}

// GameInstance is the core game state. This data structure is shared by server and client albeit with different states.
type GameInstance struct {
	// game instance metadata
	id      string
	owner   uint64
	mapFile string
	public  bool

	rules *Ruleset

	// game instance state
	currentPlayerIndex int
	units              map[uint64]*UnitInstance
	losMatrix          map[uint64]map[uint64]bool
	voxelMap           *voxel.Map
	mapMeta            *MapMetadata
	players            []uint64
	playerFactions     map[uint64]*Faction
	playerUnits        map[uint64][]uint64
	playersNeeded      int

	blockLibrary *BlockLibrary
	// debug
	environment string

	// mechanics
	overwatch      map[voxel.Int3][]*UnitInstance
	pressureMatrix map[uint64]map[uint64]float64
}

func (g *GameInstance) SetEnvironment(environment string) {
	g.environment = environment
}

func (g *GameInstance) GetPlayerFactions() map[uint64]string {
	result := make(map[uint64]string)
	for playerID, faction := range g.playerFactions {
		result[playerID] = faction.Name
	}
	return result
}

func (g *GameInstance) NextPlayer() uint64 {
	println(fmt.Sprintf("[GameInstance] Ending turn for %s", g.currentPlayerFaction().Name))
	g.currentPlayerIndex = (g.currentPlayerIndex + 1) % len(g.players)
	println(fmt.Sprintf("[GameInstance] Starting turn for %s", g.currentPlayerFaction().Name))

	for _, unit := range g.currentPlayerUnits() {
		if !unit.IsActive() {
			continue
		}
		unit.NextTurn()
	}
	return g.currentPlayerID()
}

func (g *GameInstance) currentPlayerUnits() []*UnitInstance {
	return g.GetPlayerUnits(g.currentPlayerID())
}

func (g *GameInstance) currentPlayerFaction() *Faction {
	return g.playerFactions[g.currentPlayerID()]
}

func (g *GameInstance) currentPlayerID() uint64 {
	return g.players[g.currentPlayerIndex]
}

func (g *GameInstance) AddPlayer(id uint64) {
	println(fmt.Sprintf("[GameInstance] Adding player %d to game %s", id, g.id))
	g.players = append(g.players, id)
}

func (g *GameInstance) IsReady() bool {
	return len(g.players) == g.playersNeeded && len(g.playerFactions) == g.playersNeeded && len(g.playerUnits) == g.playersNeeded
}

func (g *GameInstance) Start() uint64 {
	firstPlayer := g.players[0]
	return firstPlayer
}

func (g *GameInstance) SetFaction(userID uint64, faction *Faction) {
	g.playerFactions[userID] = faction
	println(fmt.Sprintf("[GameInstance] Player %d is now in faction %s", userID, faction.Name))
}

func (g *GameInstance) ServerSpawnUnit(userID uint64, unit *UnitInstance) uint64 {
	if _, unitsExist := g.playerUnits[userID]; !unitsExist {
		g.playerUnits[userID] = make([]uint64, 0)
	}
	unitInstanceID := uint64(len(g.units))
	unit.SetUnitID(unitInstanceID)
	g.playerUnits[userID] = append(g.playerUnits[userID], unitInstanceID)
	g.units[unitInstanceID] = unit

	// determine map placement
	teamIndex := g.IndexOfPlayer(userID) // 0..n
	spawnsForTeam := g.mapMeta.SpawnPositions[teamIndex]

	randomChoice := util.RandomChoice(spawnsForTeam)
	for canBePlaced, _ := g.voxelMap.IsUnitPlaceable(unit, randomChoice); !canBePlaced; {
		randomChoice = util.RandomChoice(spawnsForTeam)
		canBePlaced, _ = g.voxelMap.IsUnitPlaceable(unit, randomChoice)
	}

	unit.SetBlockPosition(randomChoice)
	unit.StartStanceAnimation()

	println(fmt.Sprintf("[ServerSpawnUnit] Adding unit %d -> %s of type %d for player %d", unitInstanceID, unit.Name, unit.Definition.ID, userID))

	return unitInstanceID
}
func (g *GameInstance) ClientAddUnit(userID uint64, unit *UnitInstance) uint64 {
	if _, unitsExist := g.playerUnits[userID]; !unitsExist {
		g.playerUnits[userID] = make([]uint64, 0)
	}
	unitInstanceID := unit.UnitID()
	println(fmt.Sprintf("[ClientAddUnit] Adding unit %d -> %s of type %d for player %d", unitInstanceID, unit.Name, unit.Definition.ID, userID))
	g.playerUnits[userID] = append(g.playerUnits[userID], unitInstanceID)
	g.units[unitInstanceID] = unit

	unit.SetVoxelMap(g.voxelMap)

	return unitInstanceID
}

func (g *GameInstance) ClientUpdateUnit(unit *UnitInstance) {
	g.units[unit.UnitID()] = unit
}

func (g *GameInstance) GetPlayerUnits(userID uint64) []*UnitInstance {
	result := make([]*UnitInstance, 0)
	for _, unitID := range g.playerUnits[userID] {
		result = append(result, g.units[unitID])
	}
	return result
}
func (g *GameInstance) IsPlayerTurn(id uint64) bool {
	return g.currentPlayerID() == id
}

func (g *GameInstance) SetLOS(observer uint64, target uint64, canSee bool) {
	if _, ok := g.losMatrix[observer]; !ok {
		g.losMatrix[observer] = make(map[uint64]bool)
	}
	g.losMatrix[observer][target] = canSee
}

func (g *GameInstance) IsGameOver() (bool, uint64) {
	playersWithActiveUnits := make(map[uint64]bool)
	for playerID, units := range g.playerUnits {
		for _, unitID := range units {
			unit := g.units[unitID]
			if unit.IsActive() {
				playersWithActiveUnits[playerID] = true
				break
			}
		}
	}
	if len(playersWithActiveUnits) == 1 {
		for playerID := range playersWithActiveUnits {
			return true, playerID
		}
	}
	return false, 0
}

func (g *GameInstance) Kill(killer, victim *UnitInstance) {
	if killer != nil {
		println(fmt.Sprintf("[%s] %s(%d) killed %s(%d)", g.environment, killer.Name, killer.UnitID(), victim.Name, victim.UnitID()))
	} else {
		println(fmt.Sprintf("[%s] %s(%d) died", g.environment, victim.Name, victim.UnitID()))
	}
	victim.Kill()
}

func (g *GameInstance) GetLOSState(playerID uint64) (map[uint64]map[uint64]bool, []*UnitInstance) {
	whoCanSeeWho := make(map[uint64]map[uint64]bool)
	for _, unitID := range g.playerUnits[playerID] {
		unit := g.units[unitID]
		if !unit.IsActive() {
			continue
		}
		whoCanSeeWho[unit.UnitID()] = g.losMatrix[unit.UnitID()]
	}
	return whoCanSeeWho, toList(g.GetAllVisibleEnemies(playerID))
}

func (g *GameInstance) GetPressureMatrix() map[uint64]map[uint64]float64 {
	return g.pressureMatrix
}

func (g *GameInstance) GetAllVisibleEnemies(playerID uint64) map[*UnitInstance]bool {
	result := make(map[*UnitInstance]bool)
	for observerID, unitsVisible := range g.losMatrix {
		observer := g.units[observerID]
		if observer.ControlledBy() != playerID {
			continue
		}
		for unitID, isVisible := range unitsVisible {
			observed := g.units[unitID]
			if observed.ControlledBy() == playerID {
				continue
			}
			if isVisible {
				result[g.units[unitID]] = true
			}
		}
	}
	return result
}

func (g *GameInstance) GetVoxelMap() *voxel.Map {
	return g.voxelMap
}

func (g *GameInstance) GetPlayerIDs() []uint64 {
	return g.players
}

func (g *GameInstance) GetID() string {
	return g.id
}

func (g *GameInstance) GetMapFile() string {
	return g.mapFile
}

func (g *GameInstance) GetUnit(unitID uint64) (*UnitInstance, bool) {
	unit, ok := g.units[unitID]
	return unit, ok
}

func (g *GameInstance) GetAllUnits() map[uint64]*UnitInstance {
	return g.units
}

func toList(result map[*UnitInstance]bool) []*UnitInstance {
	var list []*UnitInstance
	for unit := range result {
		list = append(list, unit)
	}
	return list
}
func (g *GameInstance) SetVoxelMap(loadedMap *voxel.Map) {
	g.voxelMap = loadedMap
}

func (g *GameInstance) SaveMapToDisk() {
	g.voxelMap.SaveToDisk()
	g.mapMeta.SaveToDisk(g.mapFile)
}
func (g *GameInstance) ApplyDamage(attacker, hitUnit *UnitInstance, damage int, bodyPart util.DamageZone) bool {
	lethal := hitUnit.ApplyDamage(damage, bodyPart)
	if lethal {
		g.Kill(attacker, hitUnit)
		return true
	}
	return false
}

func (g *GameInstance) CreateExplodeEffect(position voxel.Int3, radius int) {
	println(fmt.Sprintf("[%s] Explosion at %s with radius %d", g.environment, position.ToString(), radius))
	// TODO: implement visual effect
	xStart := int32(position.X) - int32(radius)
	xEnd := int32(position.X) + int32(radius)
	yStart := int32(position.Y) - int32(radius)
	yEnd := int32(position.Y) + int32(radius)
	zStart := int32(position.Z) - int32(radius)
	zEnd := int32(position.Z) + int32(radius)

	for x := xStart; x <= xEnd; x++ {
		for y := yStart; y <= yEnd; y++ {
			for z := zStart; z <= zEnd; z++ {
				dist := math.Sqrt(math.Pow(float64(x-position.X), 2) + math.Pow(float64(y-position.Y), 2) + math.Pow(float64(z-position.Z), 2))
				if dist <= float64(radius) {
					explodingBlock := g.voxelMap.GetGlobalBlock(x, y, z)
					if explodingBlock.IsOccupied() {
						affectedUnit := explodingBlock.GetOccupant().(*UnitInstance)
						g.ApplyDamage(nil, affectedUnit, 5, util.ZoneTorso) // TODO: can we do better with the damage zone?
					}
					g.voxelMap.SetBlock(x, y, z, voxel.NewAirBlock())
				}
			}
		}
	}
	g.voxelMap.GenerateAllMeshes()
}

func (g *GameInstance) SetBlockLibrary(bl *BlockLibrary) {
	g.blockLibrary = bl
}

func (g *GameInstance) GetBlockLibrary() *BlockLibrary {
	return g.blockLibrary
}
func (g *GameInstance) GetBlockDefAt(blockPos voxel.Int3) *BlockDefinition {
	block := g.voxelMap.GetGlobalBlock(blockPos.X, blockPos.Y, blockPos.Z)
	if block == nil {
		return VoidBlockDefinition
	}
	return g.blockLibrary.GetBlockDefinition(block.ID)
}

func (g *GameInstance) HandleUnitHitWithProjectile(attacker *UnitInstance, damageModifier float64, rayHitInfo FreeAimHit) (int, bool) {
	hitUnit := rayHitInfo.UnitHit.(*UnitInstance)
	//direction := rayHitInfo.HitInfo3D.CollisionWorldPosition.Sub(rayHitInfo.Origin).Normalize()
	distance := rayHitInfo.Distance
	projectileBaseDamage := attacker.GetWeapon().Definition.BaseDamagePerBullet
	// actual server side simulation
	projectileBaseDamage = attacker.GetWeapon().AdjustDamageForDistance(float32(distance), projectileBaseDamage)

	projectileBaseDamage = int(math.Ceil(float64(projectileBaseDamage) * damageModifier))
	// state changes here
	// 1. apply damage
	lethal := g.ApplyDamage(attacker, hitUnit, projectileBaseDamage, rayHitInfo.BodyPart)
	// 2. change unit orientation
	// hitUnit.Transform.SetForward2D(direction.Mul(-1.0)) // this is only visual fluff and should be done on the clients only
	return projectileBaseDamage, lethal
}

func (g *GameInstance) RegisterOverwatch(unit *UnitInstance, targets []voxel.Int3) {
	println(fmt.Sprintf("[%s] Registering overwatch for %s(%d) on %v", g.environment, unit.GetName(), unit.UnitID(), targets))
	for _, target := range targets {
		g.overwatch[target] = append(g.overwatch[target], unit)
	}
}

func (g *GameInstance) GetEnemiesWatchingPosition(playerID uint64, pos voxel.Int3) ([]*UnitInstance, bool) {
	instances, overwatch := g.overwatch[pos]
	if !overwatch {
		return nil, false
	}
	var enemies []*UnitInstance
	for _, instance := range instances {
		if instance.ControlledBy() != playerID {
			enemies = append(enemies, instance)
		}
	}
	return enemies, len(enemies) > 0
}

func (g *GameInstance) RemoveOverwatch(id uint64, pos voxel.Int3) {
	instances, overwatch := g.overwatch[pos]
	if !overwatch {
		return
	}
	for i := len(instances) - 1; i >= 0; i-- {
		instance := instances[i]
		if instance.UnitID() == id {
			g.overwatch[pos] = append(instances[:i], instances[i+1:]...)
			return
		}
	}
}

func (g *GameInstance) AreAllies(unit *UnitInstance, other *UnitInstance) bool {
	return unit.ControlledBy() == other.ControlledBy()
}

// UpdatePressureAfterMove updates the pressure matrix after a unit has moved.
// Should always be called after a unit has moved.
// Assumes the losMatrix is up to date.
func (g *GameInstance) UpdatePressureAfterMove(unit *UnitInstance) {
	for _, otherUnit := range g.units {
		if g.AreAllies(unit, otherUnit) {
			continue
		}
		canSeeA := g.losMatrix[unit.UnitID()][otherUnit.UnitID()]
		canSeeB := g.losMatrix[otherUnit.UnitID()][unit.UnitID()]
		if canSeeA && canSeeB {
			g.SetPressure(unit, otherUnit)
		} else {
			g.RemovePressure(unit.UnitID(), otherUnit.UnitID())
		}
	}
}

func (g *GameInstance) GetRules() *Ruleset {
	return g.rules
}

func (g *GameInstance) GetMapMetadata() *MapMetadata {
	return g.mapMeta
}

func (g *GameInstance) IndexOfPlayer(id uint64) int {
	for i, playerID := range g.players {
		if playerID == id {
			return i
		}
	}
	return -1
}
