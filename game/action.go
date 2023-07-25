package game

import "github.com/memmaker/battleground/engine/voxel"

type Action interface {
    GetValidTargets(unit *Unit) []voxel.Int3
    Execute(unit *Unit, target voxel.Int3)
    GetName() string
}
