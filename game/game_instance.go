package game

import (
	"fmt"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
	"math"
)

func NewGameInstanceWithMap(gameID string, mapFile string, details *MissionDetails) *GameInstance {
	println(fmt.Sprintf("[GameInstance] '%s' created", gameID))
	assetLoader := NewAssets()
	mapMetadata := assetLoader.LoadMapMetadata(mapFile)
	details.SyncFromMap(mapMetadata)

	g := &GameInstance{
        id:             gameID,
        mapFile:        mapFile,
		assets:            assetLoader,
        players:        make([]uint64, 0),
        playerFactions: make(map[uint64]*Faction),
        losMatrix:      make(map[uint64]map[uint64]bool),
        pressureMatrix: make(map[uint64]map[uint64]float64),
        playerUnits:    make(map[uint64][]uint64),
        units:          make(map[uint64]*UnitInstance),
        playersNeeded:  2,
		waitForDeployment: details.Placement == PlacementModeManual,
        voxelMap:       voxel.NewMapFromSource(assetLoader.LoadMap(mapFile), nil, nil),
        mapMeta:        &mapMetadata,
        overwatch:      make(map[voxel.Int3][]*UnitInstance),
        missionDetails: details,
		activeBlockEffects: make(map[voxel.Int3]BlockStatusEffectInstance),
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
	IsGroundLayerDestructible bool
	IsThrowTurnEnding         bool
}

func NewDefaultRuleset(engine *GameInstance) *Ruleset {
	return &Ruleset{
		engine:                    engine,
		MaxPressureDistance:       4,
		MaxOverwatchRange:         20,
		OverwatchAccuracyModifier: 0.8, // 20% penalty for overwatch shots
		OverwatchDamageModifier:   1.1, // 10% bonus damage for overwatch shots
		IsRangedAttackTurnEnding:  true,
		IsGroundLayerDestructible: false,
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
    if unit.HasWeaponOfType(WeaponSniper) {
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

	rules          *Ruleset
	assets         *Assets
	missionDetails *MissionDetails

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
	overwatch            map[voxel.Int3][]*UnitInstance
	pressureMatrix       map[uint64]map[uint64]float64
	waitForDeployment    bool
	onTargetedEffect     func(voxel.Int3, TargetedEffect, float64, int)
	onNotification       func(string)
	onBlockEffectAdded   func(voxel.Int3, BlockEffect)
	onBlockEffectRemoved func(voxel.Int3, BlockEffect)

	turnCounter        int
	activeBlockEffects map[voxel.Int3]BlockStatusEffectInstance
}

func (g *GameInstance) SetEnvironment(environment string) {
	g.environment = environment
}
func (g *GameInstance) SetOnTargetedEffect(onTargetedEffect func(voxel.Int3, TargetedEffect, float64, int)) {
	g.onTargetedEffect = onTargetedEffect
}
func (g *GameInstance) SetOnNotification(onNotification func(string)) {
	g.onNotification = onNotification
}
func (g *GameInstance) SetOnBlockEffectAdded(onBlockEffectAdded func(voxel.Int3, BlockEffect)) {
	g.onBlockEffectAdded = onBlockEffectAdded
}
func (g *GameInstance) SetOnBlockEffectRemoved(onBlockEffectRemoved func(voxel.Int3, BlockEffect)) {
	g.onBlockEffectRemoved = onBlockEffectRemoved
}
func (g *GameInstance) GetPlayerFactions() map[uint64]string {
	result := make(map[uint64]string)
	for playerID, faction := range g.playerFactions {
		result[playerID] = faction.Name
	}
	return result
}

func (g *GameInstance) NextPlayer() uint64 {
	//println(fmt.Sprintf("[GameInstance] Ending turn for %s", g.currentPlayerFaction().Name))
	g.turnCounter++
	g.currentPlayerIndex = (g.currentPlayerIndex + 1) % len(g.players)
	//println(fmt.Sprintf("[GameInstance] Starting turn for %s", g.currentPlayerFaction().Name))

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
	g.logGameInfo(fmt.Sprintf("[GameInstance] Adding player %d to game %s", id, g.id))
	g.players = append(g.players, id)
}

func (g *GameInstance) IsReady() bool {
	return len(g.players) == g.playersNeeded && len(g.playerFactions) == g.playersNeeded && len(g.playerUnits) == g.playersNeeded
}
func (g *GameInstance) AllUnitsDeployed() bool {
	if !g.waitForDeployment {
		return true
	}
	for _, unit := range g.units {
		if !g.GetVoxelMap().IsUnitOnMap(unit) {
			return false
		}
	}
	return true
}
func (g *GameInstance) Start() uint64 {
	firstPlayer := g.players[0]
	return firstPlayer
}

func (g *GameInstance) SetFaction(userID uint64, faction *Faction) {
	g.playerFactions[userID] = faction
	g.logGameInfo(fmt.Sprintf("[GameInstance] Player %d is now in faction %s", userID, faction.Name))
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
	unit.SetBlockPositionAndUpdateStance(randomChoice)
	unit.UpdateMapPosition()
	unit.StartStanceAnimation()

	g.logGameInfo(fmt.Sprintf("[ServerSpawnUnit] Adding unit %d -> %s of type %d for player %d", unitInstanceID, unit.Name, unit.Definition.ID, userID))

	return unitInstanceID
}
func (g *GameInstance) ClientAddUnit(userID uint64, unit *UnitInstance) uint64 {
	if _, unitsExist := g.playerUnits[userID]; !unitsExist {
		g.playerUnits[userID] = make([]uint64, 0)
	}
	unitInstanceID := unit.UnitID()
	g.logGameInfo(fmt.Sprintf("[ClientAddUnit] Adding unit %d -> %s of type %d for player %d", unitInstanceID, unit.Name, unit.Definition.ID, userID))
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

    if g.missionDetails.Scenario == MissionScenarioDefend { // player at Index 0 is the defender
        if g.missionDetails.AllObjectivesDestroyed() {
            return true, g.players[1]
		} else if g.turnCounter >= g.missionDetails.TurnLimit {
			return true, g.players[0]
        }
    }

	return false, 0
}

func (g *GameInstance) Kill(killer, victim *UnitInstance) {
	if killer != nil {
		g.logGameInfo(fmt.Sprintf("[%s] %s(%d) killed %s(%d)", g.environment, killer.Name, killer.UnitID(), victim.Name, victim.UnitID()))
	} else {
		g.logGameInfo(fmt.Sprintf("[%s] %s(%d) died", g.environment, victim.Name, victim.UnitID()))
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
			if !isVisible {
				continue
			}
			observed := g.units[unitID]
			if observed.ControlledBy() == playerID {
				continue
			}
			result[g.units[unitID]] = true
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
	mapFileName := g.assets.GetMapPath(g.mapFile)
	errMap := g.voxelMap.SaveToDisk(mapFileName)
	metaErr := g.mapMeta.SaveToDisk(mapFileName)
	if errMap != nil || metaErr != nil {
		g.logGameError(fmt.Sprintf("[GameInstance] ERR - SaveMapToDisk - %v %v", errMap, metaErr))
		g.onNotification("ERROR saving map")
	} else if g.onNotification != nil {
		g.onNotification("Saved successfully")
	}
}
func (g *GameInstance) ApplyDamage(attacker, hitUnit *UnitInstance, damage int, bodyPart util.DamageZone) bool {
	lethal := hitUnit.ApplyDamage(damage, bodyPart)
	if lethal {
		g.Kill(attacker, hitUnit)
		return true
	}
	return false
}

func (g *GameInstance) ApplyTargetedEffectFromMessage(msg MessageTargetedEffect) {
	switch msg.Effect {
	case TargetedEffectSmokeCloud:
		g.CreateSmokeCloudEffect(msg.Position, msg.Radius, msg.TurnsToLive)
	case TargetedEffectPoisonCloud:
        g.CreatePoisonCloudEffect(msg.Position, msg.Radius, msg.TurnsToLive)
	case TargetedEffectFire:
        g.AddFireAt(msg.Position, msg.TurnsToLive)
	case TargetedEffectExplosion:
		g.CreateExplodeEffect(msg.Position, msg.Radius)
	}
	if g.onTargetedEffect != nil {
		g.onTargetedEffect(msg.Position, msg.Effect, msg.Radius, msg.TurnsToLive)
	}
}
func (g *GameInstance) CreateExplodeEffect(position voxel.Int3, radius float64) {
	g.logGameInfo(fmt.Sprintf("[%s] Explosion at %s with radius %0.2f", g.environment, position.ToString(), radius))
	g.voxelMap.ForBlockInSphere(position, radius, g.applyExplosionToSingleBlock)
}

func (g *GameInstance) CreateSmokeCloudEffect(position voxel.Int3, radius float64, turns int) {
	g.logGameInfo(fmt.Sprintf("[%s] Smoke at %s with radius %0.2f", g.environment, position.ToString(), radius))
	g.voxelMap.ForBlockInHalfSphere(position, radius, func(origin voxel.Int3, radius float64, x int32, y int32, z int32) {
		g.AddSmokeAt(voxel.Int3{X: x, Y: y, Z: z}, turns)
	})
}

func (g *GameInstance) CreatePoisonCloudEffect(position voxel.Int3, radius float64, turns int) {
    g.logGameInfo(fmt.Sprintf("[%s] Poison at %s with radius %0.2f", g.environment, position.ToString(), radius))
    g.voxelMap.ForBlockInHalfSphere(position, radius, func(origin voxel.Int3, radius float64, x int32, y int32, z int32) {
        g.AddPoisonAt(voxel.Int3{X: x, Y: y, Z: z}, turns)
    })
}
func (g *GameInstance) AddSmokeAt(location voxel.Int3, turns int) {
	if !g.voxelMap.Contains(location.X, location.Y, location.Z) {
		return
	}
	if g.voxelMap.IsSolidBlockAt(location.X, location.Y, location.Z) {
		return
	}
	println(fmt.Sprintf("[GameInstance] Adding smoke at %s", location.ToString()))
	g.addBlockStatusEffect(location, BlockEffectSmoke, turns)
}

func (g *GameInstance) AddFireAt(location voxel.Int3, turns int) {
    if !g.voxelMap.Contains(location.X, location.Y, location.Z) {
        return
    }
    if g.voxelMap.IsSolidBlockAt(location.X, location.Y, location.Z) {
        return
    }
    println(fmt.Sprintf("[GameInstance] Adding smoke at %s", location.ToString()))
    g.addBlockStatusEffect(location, BlockEffectFire, turns)
}

func (g *GameInstance) AddPoisonAt(location voxel.Int3, turns int) {
    if !g.voxelMap.Contains(location.X, location.Y, location.Z) {
        return
    }
    if g.voxelMap.IsSolidBlockAt(location.X, location.Y, location.Z) {
        return
    }
    println(fmt.Sprintf("[GameInstance] Adding poison at %s", location.ToString()))
    g.addBlockStatusEffect(location, BlockEffectPoison, turns)
}
func (g *GameInstance) applyExplosionToSingleBlock(origin voxel.Int3, radius float64, x, y, z int32) {
	explodingBlock := g.voxelMap.GetGlobalBlock(x, y, z)
	if explodingBlock.IsOccupied() {
		affectedUnit := explodingBlock.GetOccupant().(*UnitInstance)
		g.ApplyDamage(nil, affectedUnit, 5, util.ZoneTorso) // TODO: can we do better with the damage zone? and value..
	}
	g.DestroyBlock(voxel.Int3{X: x, Y: y, Z: z})
}

func (g *GameInstance) DestroyBlock(pos voxel.Int3) {
	if !g.rules.IsGroundLayerDestructible && pos.Y == 0 { // don't allow in-game destruction of the last ground layer
		return
	}
	g.voxelMap.SetAir(pos)
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
	g.logGameInfo(fmt.Sprintf("[%s] Registering overwatch for %s(%d) on %v", g.environment, unit.GetName(), unit.UnitID(), targets))
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

func (g *GameInstance) GetAssets() *Assets {
	return g.assets
}

func (g *GameInstance) GetMissionDetails() *MissionDetails {
	return g.missionDetails
}

func (g *GameInstance) TryDeploy(playerID uint64, deployment map[uint64]voxel.Int3) bool {
	for unitID, pos := range deployment {
		unit, ok := g.units[unitID]
		if !ok {
			g.logGameError(fmt.Sprintf("[GameInstance] ERR - TryDeploy - Unit %d does not exist", unitID))
			return false
		}
		if unit.ControlledBy() != playerID {
			g.logGameError(fmt.Sprintf("[GameInstance] ERR - TryDeploy - Unit %d is not controlled by player %d", unitID, playerID))
			return false
		}
		// TODO: needs validation..
		unit.SetBlockPositionAndUpdateStance(pos)
		unit.StartStanceAnimation()
	}
	return true
}

func (g *GameInstance) DeploymentDone() {
	g.waitForDeployment = false
}

func (g *GameInstance) logGameInfo(text string) {
	switch g.environment {
	case "GL-Client":
		util.LogGraphicalClientGameInfo(text)
	case "AI-Client":
		util.LogAiClientGameInfo(text)
	case "Server":
		util.LogServerGameInfo(text)
	}
}

func (g *GameInstance) logGameError(text string) {
	switch g.environment {
	case "GL-Client":
		util.LogGraphicalClientGameError(text)
	case "AI-Client":
		util.LogAiClientGameError(text)
	case "Server":
		util.LogServerGameError(text)
	}
}

func (g *GameInstance) ClearSmokeMulti(blocks []voxel.Int3) {
	for _, block := range blocks {
		g.removeBlockStatusEffect(block, BlockEffectSmoke|BlockEffectPoison)
	}
}

func (g *GameInstance) addBlockStatusEffect(location voxel.Int3, effect BlockEffect, turnsToLive int) {
	currentEffect, exists := g.activeBlockEffects[location]
	if exists {
		if currentEffect.Effect == effect {
			// extend lifetime
			currentEffect.Turns = turnsToLive
			g.activeBlockEffects[location] = currentEffect
		} else {
			g.removeBlockStatusEffect(location, currentEffect.Effect)
		}
	}
	g.activeBlockEffects[location] = BlockStatusEffectInstance{Effect: effect, Turns: turnsToLive}
	g.blockEffectAdded(location, effect)
}

func (g *GameInstance) blockEffectAdded(location voxel.Int3, effect BlockEffect) {
	if g.onBlockEffectAdded != nil {
		g.onBlockEffectAdded(location, effect)
	}
}

func (g *GameInstance) blockEffectRemoved(location voxel.Int3, effect BlockEffect) {
	if g.onBlockEffectRemoved != nil {
		g.onBlockEffectRemoved(location, effect)
	}
}

func (g *GameInstance) removeBlockStatusEffect(location voxel.Int3, effect BlockEffect) {
	currentEffect, exists := g.activeBlockEffects[location]
	if !exists {
		return
	}
	// bitflag
	if currentEffect.Effect&effect == 0 {
		return
	}
	delete(g.activeBlockEffects, location)
	g.blockEffectRemoved(location, effect)
}
