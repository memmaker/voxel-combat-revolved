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
