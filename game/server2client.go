package game

import "github.com/memmaker/battleground/engine/voxel"

type ActionResponse struct {
	Success bool
	Message string
}

func (a ActionResponse) MessageType() string {
	return "ActionResponse"
}

type LoginResponse struct {
	UserID  uint64
	Success bool
	Message string
}

type GameStartedMessage struct {
	OwnID            uint64
	SpawnIndex       uint64
	OwnUnits         []*UnitInstance
	GameID           string
	PlayerFactionMap map[uint64]string
	PlayerNameMap    map[uint64]string
	MapFile          string
	LOSMatrix        map[uint64]map[uint64]bool
	PressureMatrix   map[uint64]map[uint64]float64
	VisibleUnits     []*UnitInstance
	MissionDetails   *MissionDetails
	// set when a player rejoins a running match after a disconnect
	Rejoin          bool         `json:",omitempty"`
	DestroyedBlocks []voxel.Int3 `json:",omitempty"`
	Round           int          `json:",omitempty"`
}

// PlayerConnectionMessage tells the other players that someone dropped out or came back.
type PlayerConnectionMessage struct {
	Name      string
	Connected bool
	Seconds   int // grace period before the absent player forfeits
}

func (m PlayerConnectionMessage) MessageType() string {
	return "PlayerConnection"
}

type NextPlayerMessage struct {
	CurrentPlayer uint64
	YourTurn      bool
	TurnSeconds   int `json:",omitempty"` // time limit for this turn, 0 = unlimited
	// Resume re-announces the current turn to a rejoined player: the units keep their AP
	Resume bool `json:",omitempty"`
}

func (n NextPlayerMessage) MessageType() string {
	return "NextPlayer"
}

type GameOverMessage struct {
	WinnerID uint64
	YouWon   bool
}

type StartDeploymentMessage struct {
}

func (s StartDeploymentMessage) MessageType() string {
	return "StartDeployment"
}
