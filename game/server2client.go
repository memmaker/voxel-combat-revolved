package game

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
	OwnUnits         []*UnitInstance
	GameID           string
	PlayerFactionMap map[uint64]string
	PlayerNameMap    map[uint64]string
	MapFile          string
	LOSMatrix        map[uint64]map[uint64]bool
	PressureMatrix   map[uint64]map[uint64]float64
	VisibleUnits     []*UnitInstance
}

type NextPlayerMessage struct {
	CurrentPlayer uint64
	YourTurn      bool
}

func (n NextPlayerMessage) MessageType() string {
	return "NextPlayer"
}

type GameOverMessage struct {
	WinnerID uint64
	YouWon   bool
}
