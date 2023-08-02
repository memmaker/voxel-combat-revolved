package game

import "github.com/memmaker/battleground/engine/voxel"

type LoginMessage struct {
	Username string
}

type SelectFactionMessage struct {
	FactionName string
}

type UnitChoices struct {
	UnitTypeID uint64
	Name       string
}

type SelectUnitsMessage struct {
	Units []UnitChoices
}
type TargetedUnitActionMessage struct {
	GameUnitID uint64
	Action     string
	Target     voxel.Int3
}
type CreateGameMessage struct {
	Map            string
	GameIdentifier string
	IsPublic       bool
}

type JoinGameMessage struct {
	GameID string
}
