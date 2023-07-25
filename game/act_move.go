package game

import (
    "fmt"
    "github.com/memmaker/battleground/engine/voxel"
)

type ActionMove struct {
    engine *BattleGame
}

func (a ActionMove) GetName() string {
    return "Move"
}

func (a ActionMove) GetValidTargets(unit *Unit) []voxel.Int3 {
    return []voxel.Int3{}
}

func (a ActionMove) Execute(unit *Unit, target voxel.Int3) {
    println(fmt.Sprintf("Moving %s to %s", unit.GetName(), target.ToString()))
    unit.SetWaypoint(target.ToBlockCenterVec3())
}
