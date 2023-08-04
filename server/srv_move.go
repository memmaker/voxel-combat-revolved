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
func (a ServerActionMove) Execute() ([]string, []any) {
	currentPos := voxel.ToGridInt3(a.unit.GetFootPosition())
	distance := a.gameAction.GetCost(a.target)
	println(fmt.Sprintf("[ActionMove] Moving %s: from %s to %s (dist: %d)", a.unit.GetName(), currentPos.ToString(), a.target.ToString(), distance))

	foundPath := a.gameAction.GetPath(a.target)
	destination := foundPath[len(foundPath)-1]

	msgTypes := []string{}
	messages := []any{}

	var visibles []*game.UnitInstance
	var invisibles []*game.UnitInstance
	var exist bool
	spottedEnemies := false
	for index, pos := range foundPath {
		// check if we can spot an enemy unit from here
		if visibles, exist = a.engine.acquiredLOS(a.unit, pos); exist {
			println(" -> can spot new enemy from here: ", pos.ToString())
			destination = pos
			foundPath = foundPath[:index+1]
			spottedEnemies = true
			break
		}
	}

	msgTypes = append(msgTypes, "UnitMoved")
	messages = append(messages, game.VisualUnitMoved{
		UnitID: a.unit.UnitID(),
		Path:   foundPath,
	})

	if invisibles, exist = a.engine.lostLOS(a.unit, destination); exist || spottedEnemies {
		msgTypes = append(msgTypes, "UnitLOSUpdated")
		messages = append(messages, game.VisualUnitLOSUpdated{
			ObserverPosition: destination,
			Observer:         a.unit.UnitID(),
			Spotted:          visibles,
			Lost:             invisibles,
		})
	}

	a.engine.voxelMap.MoveUnitTo(a.unit, a.unit.GetFootPosition(), destination.ToBlockCenterVec3())

	return msgTypes, messages
}