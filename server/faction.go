package server

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/game"
)

type FactionDefinition struct {
    Name  string
    Color mgl32.Vec3
    Units []game.UnitDefinition
}

type Faction struct {
    name  string
}
