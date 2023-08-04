package server

import (
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
	"github.com/memmaker/battleground/game"
	"math"
	"path"
)

type ServerAction interface {
	IsValid() bool
	Execute() ([]string, []any)
}

func NewGameInstance(ownerID uint64, gameID string, mapFile string, public bool) *GameInstance {
	mapDir := "./assets/maps"
	mapFile = path.Join(mapDir, mapFile)
	loadedMap := voxel.NewMap(0, 0, 0)
	loadedMap.LoadFromDisk(mapFile)
	println(fmt.Sprintf("[GameInstance] %d created game %s", ownerID, gameID))
	return &GameInstance{
		owner:                 ownerID,
		id:                    gameID,
		mapFile:               mapFile,
		public:                public,
		players:               make([]uint64, 0),
		factionMap:            make(map[*game.UnitInstance]*Faction),
		playerFactions:        make(map[uint64]*Faction),
		currentVisibleEnemies: make(map[*game.UnitInstance]map[*game.UnitInstance]bool),
		playerUnits:           make(map[uint64][]*game.UnitInstance),
		playersNeeded:         2,
		voxelMap:              loadedMap,
	}
}

type GameInstance struct {
	id      string
	owner   uint64
	mapFile string
	public  bool

	// game instance state
	currentPlayerIndex    int
	units                 []*game.UnitInstance
	factionMap            map[*game.UnitInstance]*Faction
	currentVisibleEnemies map[*game.UnitInstance]map[*game.UnitInstance]bool
	voxelMap              *voxel.Map
	players               []uint64
	playerFactions        map[uint64]*Faction
	playerUnits           map[uint64][]*game.UnitInstance
	playersNeeded         int
}

func (g *GameInstance) GetPlayerFactions() map[uint64]string {
	result := make(map[uint64]string)
	for playerID, faction := range g.playerFactions {
		result[playerID] = faction.name
	}
	return result
}

func (g *GameInstance) NextPlayer() uint64 {
	println(fmt.Sprintf("[GameInstance] Ending turn for %s", g.currentPlayerFaction().name))
	g.currentPlayerIndex = (g.currentPlayerIndex + 1) % len(g.players)
	g.UpdateVisibleEnemies()
	println(fmt.Sprintf("[GameInstance] Starting turn for %s", g.currentPlayerFaction().name))

	for _, unit := range g.currentPlayerUnits() {
		if !unit.IsActive() {
			continue
		}
		unit.NextTurn()
	}
	return g.currentPlayerID()
}

func (g *GameInstance) currentPlayerUnits() []*game.UnitInstance {
	return g.playerUnits[g.currentPlayerID()]
}

func (g *GameInstance) currentPlayerFaction() *Faction {
	return g.playerFactions[g.currentPlayerID()]
}

func (g *GameInstance) currentPlayerID() uint64 {
	return g.players[g.currentPlayerIndex]
}

func (g *GameInstance) UpdateVisibleEnemies() {
	own := g.currentPlayerFaction()
	for _, ownUnit := range g.currentPlayerUnits() { // for all own units
		for _, unit := range g.units {
			if g.factionMap[unit] == own {
				continue
			}
			// check against all other units
			if _, ok := g.currentVisibleEnemies[ownUnit]; !ok {
				g.currentVisibleEnemies[ownUnit] = make(map[*game.UnitInstance]bool)
			}
			if g.CanSee(ownUnit, unit) {
				g.currentVisibleEnemies[ownUnit][unit] = true
			} else {
				g.currentVisibleEnemies[ownUnit][unit] = false
			}
		}
	}
}

func (g *GameInstance) CanSee(one, another *game.UnitInstance) bool {
	return g.CanSeeFrom(one, another, one.GetEyePosition())
}

func (g *GameInstance) CanSeeFrom(observer, another *game.UnitInstance, observerEyePosition mgl32.Vec3) bool {
	if observer == another || observer.ControlledBy() == another.ControlledBy() {
		return true
	}
	targetOne := another.GetEyePosition()
	targetTwo := another.GetFootPosition()

	rayOne := g.RayCastUnits(observerEyePosition, targetOne, observer, another)
	rayTwo := g.RayCastUnits(observerEyePosition, targetTwo, observer, another)

	lineOfSight := rayOne.UnitHit == another || rayTwo.UnitHit == another

	println(fmt.Sprintf("[GameInstance] Could %s at (%v) see %s?: %t", observer.Name, observerEyePosition, another.Name, lineOfSight))

	return lineOfSight
}

func (g *GameInstance) OnUnitMoved(unitMapObject voxel.MapObject) {
	unit := unitMapObject.(*game.UnitInstance)
	own := g.currentPlayerFaction()
	if g.factionMap[unit] == own {
		if _, notExists := g.currentVisibleEnemies[unit]; notExists {
			g.currentVisibleEnemies[unit] = make(map[*game.UnitInstance]bool)
		}
		for _, enemy := range g.units {
			if g.factionMap[enemy] == own {
				continue
			}
			if g.CanSee(unit, enemy) {
				g.currentVisibleEnemies[unit][enemy] = true
			} else {
				g.currentVisibleEnemies[unit][enemy] = false
			}
		}
	}
}

func (g *GameInstance) acquiredLOS(unit *game.UnitInstance, pos voxel.Int3) ([]*game.UnitInstance, bool) {
	currentlyVisibleEnemies := g.currentVisibleEnemies[unit]
	newEnemies := make([]*game.UnitInstance, 0)
	own := g.currentPlayerFaction()
	for _, enemy := range g.units {
		if g.factionMap[enemy] == own {
			continue
		}
		unitWasVisible, wasKnown := currentlyVisibleEnemies[enemy]
		if wasKnown && unitWasVisible {
			continue
		}
		if g.CanSeeFrom(unit, enemy, pos.ToBlockCenterVec3().Add(unit.GetEyeOffset())) {
			newEnemies = append(newEnemies, enemy)
		}
	}
	return newEnemies, len(newEnemies) > 0
}

func (g *GameInstance) lostLOS(unit *game.UnitInstance, pos voxel.Int3) ([]*game.UnitInstance, bool) {
	currentlyVisibleEnemies := g.currentVisibleEnemies[unit]
	lostEnemies := make([]*game.UnitInstance, 0)
	for enemy, wasVisible := range currentlyVisibleEnemies {
		if !wasVisible {
			continue
		}
		if g.CanSeeFrom(unit, enemy, pos.ToBlockCenterVec3().Add(unit.GetEyeOffset())) {
			continue
		}
		lostEnemies = append(lostEnemies, enemy)
	}
	return lostEnemies, len(lostEnemies) > 0
}

func (g *GameInstance) RayCastUnits(rayStart, rayEnd mgl32.Vec3, sourceUnit, targetUnit voxel.MapObject) *game.RayCastHit {
	voxelMap := g.voxelMap
	var visitedBlocks []voxel.Int3
	var unitHit *game.UnitInstance
	stopRay := func(x, y, z int32) bool {
		visitedBlocks = append(visitedBlocks, voxel.Int3{X: x, Y: y, Z: z})
		if voxelMap.Contains(x, y, z) {
			block := voxelMap.GetGlobalBlock(x, y, z)
			if block != nil && !block.IsAir() {
				println(fmt.Sprintf("[GameInstance] Raycast hit block at %d, %d, %d", x, y, z))
				return true
			} else if block.IsOccupied() && (block.GetOccupant().ControlledBy() != sourceUnit.ControlledBy() || block.GetOccupant() == targetUnit) {
				unitHit = block.GetOccupant().(*game.UnitInstance)
				println(fmt.Sprintf("[GameInstance] Raycast hit unit %s at %d, %d, %d", unitHit.Name, x, y, z))
				return true
			}
		} else {
			println(fmt.Sprintf("[GameInstance] Raycast hit out of bounds at %d, %d, %d", x, y, z))
			return true
		}
		return false
	}
	hitInfo := util.DDARaycast(rayStart, rayEnd, stopRay)
	insideMap := voxelMap.ContainsGrid(hitInfo.CollisionGridPosition) || voxelMap.ContainsGrid(hitInfo.PreviousGridPosition)

	return &game.RayCastHit{HitInfo3D: hitInfo, VisitedBlocks: visitedBlocks, UnitHit: unitHit, InsideMap: insideMap}
}

func (g *GameInstance) RayCastFreeAim(rayStart, rayEnd mgl32.Vec3, sourceUnit *game.UnitInstance) *game.FreeAimHit {
	rayHitObject := false
	var hitPart util.Collider
	var hitPoint mgl32.Vec3
	var hitUnit voxel.MapObject
	var visitedBlocks []voxel.Int3
	checkedCollision := make(map[voxel.MapObject]bool)
	rayHitInfo := util.DDARaycast(rayStart, rayEnd, func(x, y, z int32) bool {
		visitedBlocks = append(visitedBlocks, voxel.Int3{X: x, Y: y, Z: z})
		if g.voxelMap.IsSolidBlockAt(x, y, z) || !g.voxelMap.Contains(x, y, z) {
			return true
		}
		block := g.voxelMap.GetGlobalBlock(x, y, z)

		if block != nil && block.IsOccupied() {
			collidingObject := block.GetOccupant().(*game.UnitInstance)
			if collidingObject == sourceUnit {
				return false
			}
			var rayPoint mgl32.Vec3
			rayHit := false
			if _, checkedBefore := checkedCollision[collidingObject]; checkedBefore {
				return false
			}
			minDistance := float32(math.MaxFloat32)
			//println(fmt.Sprintf("Checking %s against %s", obj.GetName(), collidingObject.GetName()))
			for _, meshPartCollider := range collidingObject.GetColliders() {
				//meshsCollided, _ = util.GJK(projectile.GetCollider(), meshPartCollider) // we made this sweeping for the projectiles only for now
				rayHit, rayPoint = meshPartCollider.IntersectsRay(rayStart, rayEnd)
				if rayHit {
					rayHitObject = true
					dist := rayPoint.Sub(rayStart).Len()
					if dist < minDistance {
						minDistance = dist
						hitPart = meshPartCollider
						hitPoint = rayPoint
						hitUnit = collidingObject
					}
				}
				checkedCollision[collidingObject] = true
			}
			if rayHitObject {
				return true
			}
		}
		return false
	})
	if rayHitObject {
		rayHitInfo = rayHitInfo.WithCollisionWorldPosition(hitPoint)
	}
	insideMap := g.voxelMap.ContainsGrid(rayHitInfo.CollisionGridPosition) || g.voxelMap.ContainsGrid(rayHitInfo.PreviousGridPosition)
	partName := util.BodyPartNone
	if hitPart != nil {
		partName = hitPart.GetName()
	}
	return &game.FreeAimHit{RayCastHit: game.RayCastHit{HitInfo3D: rayHitInfo, VisitedBlocks: visitedBlocks, UnitHit: hitUnit, InsideMap: insideMap}, BodyPart: partName}
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
	println(fmt.Sprintf("[GameInstance] Player %d is now in faction %s", userID, faction.name))
}

func (g *GameInstance) AddUnit(userID uint64, unit *game.UnitInstance) uint64 {
	if _, unitsExist := g.playerUnits[userID]; !unitsExist {
		g.playerUnits[userID] = make([]*game.UnitInstance, 0)
	}
	unitInstanceID := uint64(len(g.units))
	unit.SetGameUnitID(unitInstanceID)
	println(fmt.Sprintf("[GameInstance] Adding unit %d -> %s of type %d for player %d", unitInstanceID, unit.Name, unit.UnitDefinition.ID, userID))
	g.playerUnits[userID] = append(g.playerUnits[userID], unit)
	g.units = append(g.units, unit)
	g.factionMap[unit] = g.playerFactions[userID]
	g.voxelMap.AddUnit(unit, unit.SpawnPos.ToBlockCenterVec3())
	return unitInstanceID
}

func (g *GameInstance) GetUnitTypes(userID uint64) []uint64 {
	var result []uint64
	for _, unit := range g.playerUnits[userID] {
		result = append(result, unit.UnitDefinition.ID)
	}
	return result
}

func (g *GameInstance) GetPlayerUnits(userID uint64) []*game.UnitInstance {
	return g.playerUnits[userID]
}

func (g *GameInstance) GetServerActionForUnit(actionMessage game.UnitActionMessage, unit *game.UnitInstance) ServerAction {
	switch typedMsg := actionMessage.(type) {
	case game.TargetedUnitActionMessage:
		return g.GetTargetedAction(typedMsg, unit)
	case game.FreeAimActionMessage:
		return g.GetFreeAimAction(typedMsg, unit)
	}
	return nil
}

func (g *GameInstance) GetTargetedAction(targetAction game.TargetedUnitActionMessage, unit *game.UnitInstance) ServerAction {
	switch targetAction.Action {
	case "Move":
		return NewServerActionMove(g, game.NewActionMove(g.voxelMap), unit, targetAction.Target)
	case "Shot":
		return NewServerActionSnapShot(g, unit, targetAction.Target)
	}
	println(fmt.Sprintf("[GameInstance] ERR -> Unknown action %s", targetAction.Action))
	return nil
}

func (g *GameInstance) GetFreeAimAction(msg game.FreeAimActionMessage, unit *game.UnitInstance) ServerAction {
	switch msg.Action {
	case "Shot":
		return NewServerActionFreeShot(g, unit, msg.Origin, msg.Velocity)
	}
	println(fmt.Sprintf("[GameInstance] ERR -> Unknown action %s", msg.Action))
	return nil
}

func (g *GameInstance) IsPlayerTurn(id uint64) bool {
	return g.currentPlayerID() == id
}
