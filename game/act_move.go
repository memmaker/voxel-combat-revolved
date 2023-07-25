package game

import "github.com/memmaker/battleground/engine/voxel"

type ActionMove struct {
    engine *BattleGame
}

func (a ActionMove) GetValidTargets(unit *Unit) []voxel.Int3 {
    return []voxel.Int3{}
}

func (a ActionMove) Execute(unit *Unit, target voxel.Int3) {
    unit.SetWaypoint(target.ToBlockCenterVec3())
}
