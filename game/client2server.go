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
    Items      []string
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

type DeploymentMessage struct {
	Deployment map[uint64]voxel.Int3
}
type UnitActionMessage interface {
	UnitID() uint64
}
type TargetedUnitActionMessage struct {
	UnitMessage
	Action  string
	Targets []voxel.Int3
}

type ThrownUnitActionMessage struct {
    UnitMessage
    Action   string
    Targets  []mgl32.Vec3
    ItemName string
}

type FreeAimActionMessage struct {
	UnitMessage
	Action       string
	TargetAngles [][2]float32
	CamPos       mgl32.Vec3
}
type DebugRequest struct {
    Command string
}
type MapLoadedMessage struct {
}

type CreateGameMessage struct {
	Map            string
	GameIdentifier string
	IsPublic       bool
	MissionDetails *MissionDetails
}

type JoinGameMessage struct {
	GameID string
}

type DebugGetServerStateMessage struct {
}