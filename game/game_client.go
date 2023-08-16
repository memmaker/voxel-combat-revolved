package game

import (
	"fmt"
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
	SetForward(voxel.Int3)
	UseMovement(cost int)
	SetBlockPositionAndUpdateMap(voxel.Int3)
	SetBlockPositionAndUpdateMapAndModel(voxel.Int3)
}
type GameClient[U ClientUnit] struct {
	*GameInstance
	controllingUserID uint64
	clientUnitMap     map[uint64]U
	newClientUnit     func(*UnitInstance) U
}

func NewGameClient[U ClientUnit](controllingUserID uint64, gameID string, newClientUnit func(*UnitInstance) U) *GameClient[U] {
	return &GameClient[U]{
		GameInstance:      NewGameInstance(gameID),
		newClientUnit:     newClientUnit,
		controllingUserID: controllingUserID,
		clientUnitMap:     make(map[uint64]U),
	}
}

func (a *GameClient[U]) OnTargetedUnitActionResponse(msg ActionResponse) {
	if !msg.Success {
		println(fmt.Sprintf("[BattleClient] Action failed: %s", msg.Message))
		a.Print(fmt.Sprintf("Action failed: %s", msg.Message))
	}
}

func (a *GameClient[U]) AddOrUpdateUnit(currentUnit *UnitInstance) {
	unitID := currentUnit.GameUnitID
	if _, ok := a.clientUnitMap[unitID]; ok {
		a.UpdateUnit(currentUnit)
	} else {
		a.AddUnit(currentUnit)
	}
}
func (a *GameClient[U]) AddUnit(currentUnit *UnitInstance) U {
	unitID := currentUnit.GameUnitID
	if _, ok := a.clientUnitMap[unitID]; ok {
		println(fmt.Sprintf("[GameClient] Unit %d already known", unitID))
		return a.clientUnitMap[unitID]
	}
	// add to game instance
	a.GameInstance.ClientAddUnit(currentUnit.ControlledBy(), currentUnit)

	currentUnit.SetVoxelMap(a.GetVoxelMap())

	unit := a.newClientUnit(currentUnit)

	a.clientUnitMap[unitID] = unit

	return unit
}

func (a *GameClient[U]) AddOwnedUnit(unitInstance *UnitInstance) U {
	unit := a.AddUnit(unitInstance)
	unit.SetUserControlled()
	return unit
}
func (a *GameClient[U]) UpdateUnit(currentUnit *UnitInstance) {
	unitID := currentUnit.GameUnitID
	knownUnit, ok := a.clientUnitMap[unitID]

	if !ok {
		println(fmt.Sprintf("[GameClient] ClientUpdateUnit: unit %d not found", unitID))
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
	println(fmt.Sprintf("[GameClient] %s", text))
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
		println(fmt.Sprintf("[BattleClient] Unknown unit %d", msg.UnitID))
		return
	}
	//println(fmt.Sprintf("[BattleClient] Moving %s(%d): %v -> %v", unit.GetName(), unit.UnitID(), unit.GetBlockPosition(), msg.Path[len(msg.Path)-1]))

	destination := msg.Path[len(msg.Path)-1]

	unit.UseMovement(msg.Cost)

	for _, lostLOSUnit := range msg.Lost {
		a.SetLOSLost(msg.UnitID, lostLOSUnit)
	}
	for _, acquiredLOSUnit := range msg.Spotted {
		a.AddOrUpdateUnit(acquiredLOSUnit)
		a.SetLOSAcquired(msg.UnitID, acquiredLOSUnit.UnitID())
	}

	unit.SetBlockPositionAndUpdateMap(destination)
}

func (a *GameClient[U]) SetLOSLost(observer, unitID uint64) {
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
	println(fmt.Sprintf("[GameClient] NextPlayer: %v", msg))
	println("[GameClient] Map State:")
	a.GetVoxelMap().PrintArea2D(16, 16)
	for _, unit := range a.GetAllUnits() {
		println(fmt.Sprintf("[GameClient] > Unit %s(%d): %v", unit.GetName(), unit.UnitID(), unit.GetBlockPosition()))
	}
	if msg.YourTurn {
		a.ResetUnitsForNextTurn()
		println("[GameClient] Your turn!")
	}
}
func (a *GameClient[U]) ResetUnitsForNextTurn() {
	for _, unit := range a.GetMyUnits() {
		unit.NextTurn()
	}
}

func (a *GameClient[U]) OnEnemyUnitMoved(msg VisualEnemyUnitMoved) {
	// When an enemy unit is leaving the LOS of a player owned unit,
	// the space where the unit was standing is cleared on the client side map.
	movingUnit, exists := a.GetClientUnit(msg.MovingUnit)
	if !exists && msg.UpdatedUnit == nil {
		println(fmt.Sprintf("[BattleClient] Received LOS update for unknown unit %d", msg.MovingUnit))
		return
	}
	if msg.UpdatedUnit != nil { // we lost LOS, so no update is sent
		a.AddOrUpdateUnit(msg.UpdatedUnit)
		movingUnit, _ = a.GetClientUnit(msg.MovingUnit)
	}
	println(fmt.Sprintf("[BattleClient] Enemy unit %s(%d) moving", movingUnit.GetName(), movingUnit.UnitID()))
	for i, path := range msg.PathParts {
		println(fmt.Sprintf("[BattleClient] Path %d", i))
		for _, pathPos := range path {
			println(fmt.Sprintf("[BattleClient] --> %s", pathPos.ToString()))
		}
	}
	hasPath := len(msg.PathParts) > 0 && len(msg.PathParts[0]) > 0
	changeLOS := func() {
		for _, unit := range msg.LOSLostBy {
			a.SetLOSLost(unit, movingUnit.UnitID())
		}
		for _, unit := range msg.LOSAcquiredBy {
			a.SetLOSAcquired(unit, movingUnit.UnitID())
		}
		if msg.UpdatedUnit != nil { // we lost LOS, so no update is sent
			movingUnit.SetForward(msg.UpdatedUnit.ForwardVector)
		}
		if hasPath && a.UnitIsVisibleToPlayer(a.GetControllingUserID(), movingUnit.UnitID()) { // if the unit has actually moved further, but we lost LOS, this will set a wrong position
			// even worse: if we lost the LOS, the unit was removed from the map, but this will add it again.
			movingUnit.SetBlockPositionAndUpdateMap(msg.PathParts[len(msg.PathParts)-1][len(msg.PathParts[len(msg.PathParts)-1])-1])
		}
	}

	if !hasPath {
		if msg.UpdatedUnit != nil {
			movingUnit.SetBlockPositionAndUpdateMapAndModel(msg.UpdatedUnit.GetBlockPosition())
		}
		changeLOS()
		return
	}
	currentPos := movingUnit.GetBlockPosition()
	destination := msg.PathParts[len(msg.PathParts)-1][len(msg.PathParts[len(msg.PathParts)-1])-1]
	if currentPos == destination {
		changeLOS()
	} else {
		movingUnit.SetBlockPositionAndUpdateMapAndModel(destination)
		changeLOS()
	}
}

func (a *GameClient[U]) OnRangedAttack(msg VisualRangedAttack) {
	attacker, knownAttacker := a.GetUnit(msg.Attacker)
	var attackerUnit *UnitInstance
	if knownAttacker {
		attackerUnit = attacker
		attackerUnit.SetForward(msg.AimDirection)
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
			a.ApplyDamage(attackerUnit, victim, projectile.Damage, projectile.BodyPart)
			if !ok {
				println(fmt.Sprintf("[BattleClient] Projectile hit unit %d, but unit not found", projectile.UnitHit))
				return
			}
			println(fmt.Sprintf("[BattleClient] Projectile hit unit %s(%d)", victim.GetName(), victim.UnitID()))
		}
	}
}
