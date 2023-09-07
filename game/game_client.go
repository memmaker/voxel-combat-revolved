package game

import (
	"fmt"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
	"math"
)

type ClientUnit interface {
	SetUserControlled()
	GetName() string
	IsUserControlled() bool
	SetServerInstance(*UnitInstance)
	UnitID() uint64
	GetBlockPosition() voxel.Int3
	SetBlockPosition(voxel.Int3)
	SetForward(vec3 voxel.Int3)
	UseMovement(cost float64)
	ConsumeAP(cost int)
	EndTurn()
}
type GameClient[U ClientUnit] struct {
	*GameInstance
	controllingUserID uint64
	spawnIndex        uint64
	clientUnitMap     map[uint64]U
	newClientUnit     func(*UnitInstance) U
	deploymentQueue   []U
}

func NewGameClient[U ClientUnit](infos GameStartedMessage, newClientUnit func(*UnitInstance) U) *GameClient[U] {
	return &GameClient[U]{
		GameInstance:      NewGameInstanceWithMap(infos.GameID, infos.MapFile, infos.MissionDetails),
		newClientUnit:     newClientUnit,
		controllingUserID: infos.OwnID,
		spawnIndex:        infos.SpawnIndex,
		clientUnitMap:     make(map[uint64]U),
	}
}
func (a *GameClient[U]) GetDeploymentQueue() []U {
	return a.deploymentQueue
}
func (a *GameClient[U]) GetSpawnIndex() uint64 {
	return a.spawnIndex
}
func (a *GameClient[U]) OnTargetedUnitActionResponse(msg ActionResponse) {
	if !msg.Success {
		println(fmt.Sprintf("[%s] Action failed: %s", a.environment, msg.Message))
		a.Print(fmt.Sprintf("Action failed: %s", msg.Message))
	}
}

func (a *GameClient[U]) AddOrUpdateUnit(currentUnit *UnitInstance) {
	unitID := currentUnit.GameUnitID
	if _, ok := a.clientUnitMap[unitID]; ok {
		a.UpdateUnit(currentUnit)
	} else {
		a.AddUnitToGame(currentUnit)
	}
}
func (a *GameClient[U]) AddUnitToGame(currentUnit *UnitInstance) U {
	unitID := currentUnit.GameUnitID
	if _, ok := a.clientUnitMap[unitID]; ok {
		println(fmt.Sprintf("[%s] Unit %d already known", a.environment, unitID))
		return a.clientUnitMap[unitID]
	}
	//currentUnit.Transform.SetName(currentUnit.GetName())
	// add to game instance
	a.GameInstance.ClientAddUnit(currentUnit.ControlledBy(), currentUnit)
	currentUnit.UpdateMapPosition()

	unit := a.newClientUnit(currentUnit)

	currentUnit.AutoSetStanceAndForwardAndUpdateMap()
	currentUnit.StartStanceAnimation()

	a.clientUnitMap[unitID] = unit

	return unit
}
func (a *GameClient[U]) AddOwnedUnitToDeploymentQueue(currentUnit *UnitInstance) U {
	unitID := currentUnit.GameUnitID
	if _, ok := a.clientUnitMap[unitID]; ok {
		println(fmt.Sprintf("[%s] Unit %d already known", a.environment, unitID))
		return a.clientUnitMap[unitID]
	}
	a.GameInstance.ClientAddUnit(currentUnit.ControlledBy(), currentUnit)

	unit := a.newClientUnit(currentUnit)
	unit.SetUserControlled()
	
	a.clientUnitMap[unitID] = unit

	a.deploymentQueue = append(a.deploymentQueue, unit)
	util.LogGameInfo(fmt.Sprintf("[%s] Added unit %s(%d) to deployment queue", a.environment, unit.GetName(), unit.UnitID()))
	return unit
}
func (a *GameClient[U]) AddOwnedUnitToGame(unitInstance *UnitInstance) U {
	unit := a.AddUnitToGame(unitInstance)
	unit.SetUserControlled()
	return unit
}
func (a *GameClient[U]) UpdateUnit(currentUnit *UnitInstance) {
	unitID := currentUnit.GameUnitID
	knownUnit, ok := a.clientUnitMap[unitID]

	if !ok {
		println(fmt.Sprintf("[%s] ClientUpdateUnit: unit %d not found", a.environment, unitID))
		return
	}
	knownUnit.SetServerInstance(currentUnit)
	a.GameInstance.ClientUpdateUnit(currentUnit)
}

func (a *GameClient[U]) GetClientUnit(unitID uint64) (U, bool) {
	value, exists := a.clientUnitMap[unitID]
	return value, exists
}

func (a *GameClient[U]) GetControllingUserID() uint64 {
	return a.controllingUserID
}

func (a *GameClient[U]) GetMyUnits() []*UnitInstance {
	return a.GetPlayerUnits(a.controllingUserID)
}

func (a *GameClient[U]) GetAllClientUnits() map[uint64]U {
	return a.clientUnitMap
}

func (a *GameClient[U]) IsEnemy(unitID uint64) bool {
	unit, ok := a.GetClientUnit(unitID)
	if !ok {
		return false
	}
	return !unit.IsUserControlled()
}

func (a *GameClient[U]) GetNearestEnemy(unit U) (U, bool) {
	var returnValue U
	var nearestEnemy *UnitInstance
	nearestEnemyDistance := math.MaxFloat64
	for _, visibleUnit := range a.GetVisibleEnemyUnits(unit.UnitID()) {
		distance := float64(voxel.ManhattanDistance3(unit.GetBlockPosition(), visibleUnit.GetBlockPosition()))
		if distance < nearestEnemyDistance {
			nearestEnemy = visibleUnit
			nearestEnemyDistance = distance
		}
	}
	if nearestEnemy == nil {
		return returnValue, false
	}
	var ok bool
	returnValue, ok = a.GetClientUnit(nearestEnemy.UnitID())
	return returnValue, ok
}
func (a *GameClient[U]) IsMyUnit(unitID uint64) bool {
	unit, ok := a.GetUnit(unitID)
	if !ok {
		return false
	}
	return unit.ControlledBy() == a.controllingUserID
}

func (a *GameClient[U]) Print(text string) {
	println(fmt.Sprintf("[%s] %s", a.environment, text))
}
func (a *GameClient[U]) OnGameOver(msg GameOverMessage) {
	var printedMessage string
	if msg.YouWon {
		printedMessage = "You won!"
	} else {
		printedMessage = "You lost!"
	}
	a.Print(fmt.Sprintf("Game over! %s", printedMessage))
}
func (a *GameClient[U]) OnOwnUnitMoved(msg VisualOwnUnitMoved) {
	unit, exists := a.GetClientUnit(msg.UnitID)
	if !exists {
		println(fmt.Sprintf("[%s] Unknown unit %d", a.environment, msg.UnitID))
		return
	}
	//println(fmt.Sprintf("[BattleClient] Moving %s(%d): %v -> %v", unit.GetName(), unit.UnitID(), unit.GetBlockPosition(), msg.Path[len(msg.Path)-1]))

	destination := msg.Path[len(msg.Path)-1]

	unit.UseMovement(msg.Cost)

	a.SetLOSAndPressure(msg.LOSMatrix, msg.PressureMatrix)

	for _, acquiredLOSUnit := range msg.Spotted {
		a.AddOrUpdateUnit(acquiredLOSUnit)
	}

	unit.SetBlockPosition(destination)
}

func (a *GameClient[U]) SetLOSLost(observer, unitID uint64) {
	if a.IsMyUnit(unitID) {
		return
	}
	a.SetLOS(observer, unitID, false)
	// WHAT'S THIS DOING HERE?
	// ah, because of client side pathfinding..
	if !a.UnitIsVisibleToPlayer(a.GetControllingUserID(), unitID) {
		unit, exists := a.GetUnit(unitID)
		if exists {
			a.GetVoxelMap().RemoveUnit(unit)
		}
	}
}

func (a *GameClient[U]) SetLOSAcquired(observer, unitID uint64) {
	a.SetLOS(observer, unitID, true)
	unit, exists := a.GetUnit(unitID)
	if exists {
		a.GetVoxelMap().SetUnit(unit, unit.GetBlockPosition())
	}
}

func (a *GameClient[U]) OnNextPlayer(msg NextPlayerMessage) {
	println(fmt.Sprintf("[%s] NextPlayer: %v", a.environment, msg))
	//println(fmt.Sprintf("[%s] VoxelMap:", a.environment))
	//a.GetVoxelMap().PrintArea2D(16, 16)
	/*
	for _, unit := range a.GetAllUnits() {
		println(fmt.Sprintf("[%s] > Unit %s(%d): %v", a.environment, unit.GetName(), unit.UnitID(), unit.GetBlockPosition()))
	}

	*/
	if msg.YourTurn {
		a.ResetUnitsForNextTurn()
		println(fmt.Sprintf("[%s] It's your turn!", a.environment))
	}
}
func (a *GameClient[U]) ResetUnitsForNextTurn() {
	for _, unit := range a.GetMyUnits() {
		unit.NextTurn()
	}
}

func (a *GameClient[U]) OnBeginOverwatch(msg VisualBeginOverwatch) {
	unit, exists := a.GetClientUnit(msg.Watcher)
	if !exists {
		println(fmt.Sprintf("[%s] Unknown unit %d", a.environment, msg.Watcher))
		return
	}
	unit.ConsumeAP(msg.APCost)
	unit.EndTurn()
}

func (a *GameClient[U]) OnEnemyUnitMoved(msg VisualEnemyUnitMoved) {
	// When an enemy unit is leaving the LOS of a player owned unit,
	// the space where the unit was standing is cleared on the client side map.
	movingUnit, exists := a.GetClientUnit(msg.MovingUnit)
	if !exists && msg.UpdatedUnit == nil {
		println(fmt.Sprintf("[%s] Received LOS update for unknown unit %d", a.environment, msg.MovingUnit))
		return
	}
	if msg.UpdatedUnit != nil { // we lost LOS, so no update is sent
		a.AddOrUpdateUnit(msg.UpdatedUnit)
		movingUnit, _ = a.GetClientUnit(msg.MovingUnit)
	}
	println(fmt.Sprintf("[%s] Enemy unit %s(%d) moving", a.environment, movingUnit.GetName(), movingUnit.UnitID()))
	for i, path := range msg.PathParts {
		println(fmt.Sprintf("[%s] Path %d", a.environment, i))
		for _, pathPos := range path {
			println(fmt.Sprintf("[%s] --> %s", a.environment, pathPos.ToString()))
		}
	}
	hasPath := len(msg.PathParts) > 0 && len(msg.PathParts[0]) > 0
	changeLOS := func() {
		a.SetLOSAndPressure(msg.LOSMatrix, msg.PressureMatrix)
		if msg.UpdatedUnit != nil { // we lost LOS, so no update is sent
			movingUnit.SetForward(msg.UpdatedUnit.GetForward())
		}
		if hasPath && a.UnitIsVisibleToPlayer(a.GetControllingUserID(), movingUnit.UnitID()) { // if the unit has actually moved further, but we lost LOS, this will set a wrong position
			// even worse: if we lost the LOS, the unit was removed from the map, but this will add it again.
			movingUnit.SetBlockPosition(msg.PathParts[len(msg.PathParts)-1][len(msg.PathParts[len(msg.PathParts)-1])-1])
		}
	}

	if !hasPath {
		if msg.UpdatedUnit != nil {
			movingUnit.SetBlockPosition(msg.UpdatedUnit.GetBlockPosition())
		}
		changeLOS()
		return
	}
	currentPos := movingUnit.GetBlockPosition()
	destination := msg.PathParts[len(msg.PathParts)-1][len(msg.PathParts[len(msg.PathParts)-1])-1]
	if currentPos == destination {
		changeLOS()
	} else {
		movingUnit.SetBlockPosition(destination)
		changeLOS()
	}
}

func (a *GameClient[U]) OnRangedAttack(msg VisualRangedAttack) {
	attacker, knownAttacker := a.GetUnit(msg.Attacker)
	var attackerUnit *UnitInstance
	if knownAttacker {
		attackerUnit = attacker
		attackerUnit.SetForward(voxel.DirectionToGridInt3(msg.AimDirection))
		attackerUnit.GetWeapon().ConsumeAmmo(msg.AmmoCost)
		attackerUnit.ConsumeAP(msg.APCostForAttacker)
		if msg.IsTurnEnding {
			attackerUnit.EndTurn()
		}
	}
	for _, p := range msg.Projectiles {
		projectile := p
		if projectile.UnitHit >= 0 {
			victim, ok := a.GetUnit(uint64(projectile.UnitHit))
			if !ok {
				println(fmt.Sprintf("[%s] Projectile hit unknown unit %d, but unit not found", a.environment, projectile.UnitHit))
				return
			}
			a.ApplyDamage(attackerUnit, victim, projectile.Damage, projectile.BodyPart)
			println(fmt.Sprintf("[%s] Projectile hit unit %s(%d)", a.environment, victim.GetName(), victim.UnitID()))
		}

		for _, damagedBlock := range projectile.BlocksHit {
			blockDef := a.GetBlockDefAt(damagedBlock)
			blockDef.OnDamageReceived(damagedBlock, projectile.Damage)
		}
	}
}
