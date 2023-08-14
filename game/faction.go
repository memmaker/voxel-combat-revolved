package game

import (
	"github.com/go-gl/mathgl/mgl32"
)

type FactionDefinition struct {
	Name  string
	Color mgl32.Vec3
	Units []UnitDefinition
}

type Faction struct {
	Name string
}
