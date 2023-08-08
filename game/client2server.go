package game

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/voxel"
)

type LoginMessage struct {
	Username string
}

type SelectFactionMessage struct {
	FactionName string
}

type UnitChoice struct {
	UnitTypeID uint64
	Name       string
	Weapon     string
}

type SelectUnitsMessage struct {
	Units []UnitChoice
}

type UnitMessage struct {
	GameUnitID uint64
}

func (m UnitMessage) UnitID() uint64 {
	return m.GameUnitID
}

type UnitActionMessage interface {
	UnitID() uint64
}
type TargetedUnitActionMessage struct {
	UnitMessage
	Action string
	Target voxel.Int3
}

type FreeAimActionMessage struct {
	UnitMessage
	Action   string
	Origin   mgl32.Vec3
	Velocity mgl32.Vec3
}
type CreateGameMessage struct {
	Map            string
	GameIdentifier string
	IsPublic       bool
}

type JoinGameMessage struct {
	GameID string
}
