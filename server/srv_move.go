package server

import (
	"fmt"
	"github.com/memmaker/battleground/engine/voxel"
	"github.com/memmaker/battleground/game"
)

type ServerActionMove struct {
	engine     *game.GameInstance
	gameAction *game.ActionMove
	unit       *game.UnitInstance
	targets    []voxel.Int3
}

func (a ServerActionMove) SetAPCost(newCost int) {

}

func (a ServerActionMove) IsTurnEnding() bool {
	return false
}

func (a ServerActionMove) IsValid() (bool, string) {
	if len(a.targets) != 1 {
		return false, fmt.Sprintf("Expected 1 movement target for now, got %d", len(a.targets))
	}
	for _, target := range a.targets {
		if !a.gameAction.IsValidTarget(target) {
			return false, fmt.Sprintf("Target %s is not valid", target.ToString())
		}

		dist := a.gameAction.GetCost(target)
		movesLeft := a.unit.MovesLeft()

		if dist > float64(movesLeft) {
			return false, fmt.Sprintf("Targets %s is too far away (dist: %0.2f, moves left: %d)", target.ToString(), dist, movesLeft)
		}
	}
	return true, ""
}

func NewServerActionMove(engine *game.GameInstance, unit *game.UnitInstance, targets []voxel.Int3) *ServerActionMove {
	return &ServerActionMove{
		engine:     engine,
		gameAction: game.NewActionMove(engine.GetVoxelMap(), unit),
		unit:       unit,
		targets:    targets,
	}
}
func (a ServerActionMove) Execute(mb *game.MessageBuffer) {
	currentPos := a.unit.GetBlockPosition()
	moveTarget := a.targets[0]
	distance := a.gameAction.GetCost(moveTarget)
	println(fmt.Sprintf("[ActionMove] Moving %s(%d): from %s to %s (dist: %0.2f)", a.unit.GetName(), a.unit.UnitID(), currentPos.ToString(), moveTarget.ToString(), distance))

	foundPath := a.gameAction.GetPath(moveTarget)
	destination := foundPath[len(foundPath)-1]
	controller := a.unit.ControlledBy()

	var triggeredOverwatchBy []*game.UnitInstance
	var visibles []*game.UnitInstance
	var invisibles []*game.UnitInstance
	var unitForward voxel.Int3

	if len(foundPath) > 1 {
		unitForward = destination.Sub(foundPath[len(foundPath)-2])
	} else {
		unitForward = destination.Sub(currentPos)
	}

	pathPartsPerUser := make(map[uint64][][]voxel.Int3)
	for _, enemyUserID := range mb.UserIDs() {
		if enemyUserID == controller {
			continue
		}
		pathPartsPerUser[enemyUserID] = make([][]voxel.Int3, 0)
		pathPartsPerUser[enemyUserID] = append(pathPartsPerUser[enemyUserID], make([]voxel.Int3, 0))
	}
	for index, pos := range foundPath { // simulate movement step by step
		//println(fmt.Sprintf(" -> Checking position %s", pos.ToString()))
		// for each player, that is not the controller
		// check if any unit he controls could spot the moving unit from here
		// if so, add this position to the current path part
		// if not, append the current path part to the path parts for this user and start a new part
		for enemyUserID, allPaths := range pathPartsPerUser {
			currentPathIndex := len(allPaths) - 1
			atLeastOneUnitCanSee := false
			for _, enemyUnit := range a.engine.GetPlayerUnits(enemyUserID) {
				if a.engine.CanSeeTo(enemyUnit, a.unit, pos.ToBlockCenterVec3()) {
					pathPartsPerUser[enemyUserID][currentPathIndex] = append(pathPartsPerUser[enemyUserID][currentPathIndex], pos)
					atLeastOneUnitCanSee = true
					println(fmt.Sprintf(" --> can be seen by %s(%d) of %d", enemyUnit.GetName(), enemyUnit.UnitID(), enemyUserID))
					break
				}
			}
			if !atLeastOneUnitCanSee && len(pathPartsPerUser[enemyUserID][currentPathIndex]) > 0 {
				pathPartsPerUser[enemyUserID] = append(pathPartsPerUser[enemyUserID], make([]voxel.Int3, 0))
			}
		}
		// check if somebody is watching this position
		var isBeingWatched bool
		triggeredOverwatchBy, isBeingWatched = a.engine.GetEnemiesWatchingPosition(a.unit.ControlledBy(), pos)
		if isBeingWatched {
			destination = pos
			foundPath = foundPath[:index+1]
			if len(foundPath) > 1 {
				unitForward = destination.Sub(foundPath[len(foundPath)-2])
			} else {
				unitForward = destination.Sub(currentPos)
			}
			break
		}

		// check if we can spot a new enemy unit from here
		var newContact bool
		if visibles, invisibles, newContact = a.engine.GetLOSChanges(a.unit, pos); newContact {
			println(fmt.Sprintf(" --> can spot %d new enemies from here: ", len(visibles)))
			destination = pos
			foundPath = foundPath[:index+1]
			if len(foundPath) > 1 {
				unitForward = destination.Sub(foundPath[len(foundPath)-2])
			} else {
				unitForward = destination.Sub(currentPos)
			}

			break
		}
	}

	// DO THE MOVEMENT
	moveCost := a.gameAction.GetCost(destination)
	a.unit.UseMovement(moveCost)
	a.unit.SetForward(unitForward)
	a.unit.SetBlockPositionAndUpdateStance(destination)

	println(fmt.Sprintf(" --> FINAL: %s(%d) is now at %s facing %s", a.unit.GetName(), a.unit.UnitID(), a.unit.GetBlockPosition().ToString(), a.unit.GetForward2DCardinal().ToString()))

	// apply changes to LOS
	for _, unit := range visibles {
		a.engine.SetLOS(a.unit.UnitID(), unit.UnitID(), true)
	}
	for _, unit := range invisibles {
		a.engine.SetLOS(a.unit.UnitID(), unit.UnitID(), false)
	}

	for _, enemyUserID := range mb.UserIDs() {
		if enemyUserID == a.unit.ControlledBy() {
			continue
		}
		seenByUser, hiddenToUser := a.engine.GetReverseLOSChangesForUser(enemyUserID, a.unit)
		// apply changes to LOS
		for _, unit := range seenByUser {
			a.engine.SetLOS(unit, a.unit.UnitID(), true)
		}
		for _, unit := range hiddenToUser {
			a.engine.SetLOS(unit, a.unit.UnitID(), false)
		}
	}

	a.engine.UpdatePressureAfterMove(a.unit)

	losMatrixForMovingPlayer, visibleEnemies := a.engine.GetLOSState(a.unit.ControlledBy())
	newPressureState := a.engine.GetPressureMatrix()

	mb.AddMessageFor(controller, game.VisualOwnUnitMoved{
		UnitID:         a.unit.UnitID(),
		Path:           foundPath,
		Cost:           moveCost,
		EndPosition:    destination,
		Spotted:        visibleEnemies,
		LOSMatrix:      losMatrixForMovingPlayer,
		PressureMatrix: newPressureState,
		Forward:        unitForward,
	})

	for enemyUserID, allPaths := range pathPartsPerUser {
		if len(allPaths[0]) > 0 {
			for len(allPaths) > 0 && len(allPaths[len(allPaths)-1]) == 0 {
				allPaths = allPaths[:len(allPaths)-1]
			}
			losMatrixForNonMovingPlayer, _ := a.engine.GetLOSState(enemyUserID)
			println(fmt.Sprintf(" --> sending path parts to %d: %v", enemyUserID, allPaths))
			enemyUnitMoved := game.VisualEnemyUnitMoved{
				MovingUnit:     a.unit.UnitID(),
				LOSMatrix:      losMatrixForNonMovingPlayer,
				PressureMatrix: a.engine.GetPressureMatrix(),
				PathParts:      allPaths,
			}
			if a.engine.UnitIsVisibleToPlayer(enemyUserID, a.unit.UnitID()) {
				enemyUnitMoved.UpdatedUnit = a.unit
			}
			mb.AddMessageFor(enemyUserID, enemyUnitMoved)
		}
	}

	// handle overwatch
	if len(triggeredOverwatchBy) > 0 {
		a.handleOverwatch(mb, a.unit, triggeredOverwatchBy)
	}
}

func (a ServerActionMove) handleOverwatch(mb *game.MessageBuffer, movingUnit *game.UnitInstance, watchers []*game.UnitInstance) {
	targetPos := movingUnit.GetBlockPosition()
	for _, watcher := range watchers {
		shot := NewServerActionSnapShot(a.engine, watcher, []voxel.Int3{targetPos})

		shot.SetAPCost(0) // paid in the previous turn

		shot.SetAccuracyModifier(a.engine.GetRules().OverwatchAccuracyModifier)
		shot.SetDamageModifier(a.engine.GetRules().OverwatchDamageModifier)

		if valid, reason := shot.IsValid(); valid {
			println(fmt.Sprintf(" --> %s(%d) triggered overwatch by %s(%d) at %s", movingUnit.GetName(), movingUnit.UnitID(), watcher.GetName(), watcher.UnitID(), targetPos.ToString()))
			shot.Execute(mb)
			a.engine.RemoveOverwatch(watcher.UnitID(), targetPos)
		} else {
			println(fmt.Sprintf(" --> ERR: %s(%d) triggered overwatch by %s(%d) at %s, but shot is not valid: %s", movingUnit.GetName(), movingUnit.UnitID(), watcher.GetName(), watcher.UnitID(), targetPos.ToString(), reason))
		}
	}
}

func getIDs(invisibles []*game.UnitInstance) []uint64 {
	var ids []uint64
	for _, unit := range invisibles {
		ids = append(ids, unit.UnitID())
	}
	return ids
}
