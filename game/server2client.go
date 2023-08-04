package game

type ActionResponse struct {
	Success bool
	Message string
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
