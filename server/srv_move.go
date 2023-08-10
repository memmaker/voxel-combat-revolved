package server

import (
	"fmt"
	"github.com/memmaker/battleground/engine/voxel"
	"github.com/memmaker/battleground/game"
)

type ServerActionMove struct {
	engine     *GameInstance
	gameAction *game.ActionMove
	unit       *game.UnitInstance
	target     voxel.Int3
}

func (a ServerActionMove) IsValid() bool {
	return a.gameAction.IsValidTarget(a.unit, a.target)
}

func NewServerActionMove(engine *GameInstance, action *game.ActionMove, unit *game.UnitInstance, target voxel.Int3) *ServerActionMove {
	return &ServerActionMove{
		engine:     engine,
		gameAction: action,
		unit:       unit,
		target:     target,
	}
}
func (a ServerActionMove) Execute(mb *game.MessageBuffer) {
	currentPos := voxel.ToGridInt3(a.unit.GetFootPosition())
	distance := a.gameAction.GetCost(a.target)
	println(fmt.Sprintf("[ActionMove] Moving %s(%d): from %s to %s (dist: %d)", a.unit.GetName(), a.unit.UnitID(), currentPos.ToString(), a.target.ToString(), distance))

	foundPath := a.gameAction.GetPath(a.target)
	destination := foundPath[len(foundPath)-1]
	controller := a.unit.ControlledBy()

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
	for index, pos := range foundPath {
		println(fmt.Sprintf(" -> Checking position %s", pos.ToString()))
		// for each player, that is not the controller
		// check if any unit he controls could spot the moving unit from here
		// if so, add this position to the current path part
		// if not, append the current path part to the path parts for this user and start a new part
		for enemyUserID, allPaths := range pathPartsPerUser {
			currentPathIndex := len(allPaths) - 1
			atLeastOneUnitCanSee := false
			for _, enemyUnit := range a.engine.playerUnits[enemyUserID] {
				if a.engine.CanSeeTo(enemyUnit, a.unit, pos.ToBlockCenterVec3()) {
					pathPartsPerUser[enemyUserID][currentPathIndex] = append(pathPartsPerUser[enemyUserID][currentPathIndex], pos)
					atLeastOneUnitCanSee = true
					println(fmt.Sprintf(" --> can be seen by %s(%d) of %d", enemyUnit.GetName(), enemyUnit.UnitID(), enemyUserID))
					break
				}
			}
			if !atLeastOneUnitCanSee {
				println(fmt.Sprintf(" --> can't be seen by any unit of %d", enemyUserID))
			}
			if !atLeastOneUnitCanSee && len(pathPartsPerUser[enemyUserID][currentPathIndex]) > 0 {
				pathPartsPerUser[enemyUserID] = append(pathPartsPerUser[enemyUserID], make([]voxel.Int3, 0))
			}
		}

		// check if we can spot an enemy unit from here
		if visibles, invisibles = a.engine.GetLOSChanges(a.unit, pos); len(visibles) > 0 {
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

	// PROBLEM: We actually need to send a partial path to the players who can't see the unit
	// for the whole path. For every step on the path, we need to check if the unit is visible
	// to the other players.
	// PROBLEM #2: We would also need to send the current orientation of the unit, so that the
	// client can display it correctly.
	mb.AddMessageFor(controller, game.VisualOwnUnitMoved{
		UnitID:      a.unit.UnitID(),
		Path:        foundPath,
		EndPosition: destination,
		Spotted:     visibles,
		Lost:        getIDs(invisibles),
		Forward:     unitForward,
	})

	var seenBy []uint64
	var hiddenTo []uint64
	for enemyUserID, allPaths := range pathPartsPerUser {
		seenByUser, hiddenToUser := a.engine.GetReverseLOSChangesForUser(enemyUserID, a.unit, destination, visibles, invisibles)
		if len(seenByUser) > 0 || len(hiddenToUser) > 0 || len(allPaths[0]) > 0 {
			// Send only to the players who didn't move the unit
			// NOTE: This client MUST only apply these changes, after the movement animation
			// has finished.
			for len(allPaths) > 0 && len(allPaths[len(allPaths)-1]) == 0 {
				allPaths = allPaths[:len(allPaths)-1]
			}
			println(fmt.Sprintf(" --> sending path parts to %d: %v", enemyUserID, allPaths))
			enemyUnitMoved := game.VisualEnemyUnitMoved{
				MovingUnit:    a.unit.UnitID(),
				Forward:       unitForward,
				LOSAcquiredBy: seenByUser,
				LOSLostBy:     hiddenToUser,
				PathParts:     allPaths,
			}
			if len(seenByUser) > 0 {
				enemyUnitMoved.UpdatedUnit = a.unit
			}
			mb.AddMessageFor(enemyUserID, enemyUnitMoved)
			seenBy = append(seenBy, seenByUser...)
			hiddenTo = append(hiddenTo, hiddenToUser...)
		}
	}

	for _, unit := range visibles {
		a.engine.SetLOS(a.unit.UnitID(), unit.UnitID(), true)
	}
	for _, unit := range invisibles {
		a.engine.SetLOS(a.unit.UnitID(), unit.UnitID(), false)
	}
	for _, unit := range seenBy {
		a.engine.SetLOS(unit, a.unit.UnitID(), true)
	}
	for _, unit := range hiddenTo {
		a.engine.SetLOS(unit, a.unit.UnitID(), false)
	}
	// TODO: Rotate unit to face the destination
	a.unit.SetForward(unitForward)
	a.unit.SetBlockPosition(destination)
}

func getIDs(invisibles []*game.UnitInstance) []uint64 {
	var ids []uint64
	for _, unit := range invisibles {
		ids = append(ids, unit.UnitID())
	}
	return ids
}
