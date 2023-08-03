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
	GameID           string
	PlayerFactionMap map[uint64]string
	PlayerNameMap    map[uint64]string
	OwnUnits         []*UnitInstance
	MapFile          string
}

type NextPlayerMessage struct {
	CurrentPlayer uint64
	YourTurn      bool
}
