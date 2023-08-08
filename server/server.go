package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/memmaker/battleground/engine/voxel"
	"github.com/memmaker/battleground/game"
	"log"
	"net"
	"strings"
)

type BattleServer struct {
	// global state for the whole server
	availableMaps     map[string]string
	availableFactions map[string]*Faction
	availableUnits    []*game.UnitDefinition
	connectedClients  map[uint64]*UserConnection

	// game instances
	runningGames map[string]*GameInstance
}

func (b *BattleServer) AddMap(name string, filename string) {
	b.availableMaps[name] = filename
}
func (b *BattleServer) AddFaction(def FactionDefinition) {
	faction := &Faction{name: def.Name}
	b.availableFactions[def.Name] = faction
	for _, u := range def.Units {
		unit := u
		b.AddUnitDefinition(&unit)
	}
}

func (b *BattleServer) AddUnitDefinition(unit *game.UnitDefinition) {
	b.availableUnits = append(b.availableUnits, unit)
}
func (b *BattleServer) ListenTCP(endpoint string) {
	clientID := uint64(0)
	listener, err := net.Listen("tcp", endpoint)
	if err != nil {
		log.Fatalln(err)
	}
	defer listener.Close()
	println(fmt.Sprintf("Server started on %s", endpoint))
	for {
		con, err := listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		// If you want, you can increment a counter here and inject to handleClientRequest below as client identifier
		go b.handleClientRequest(con, clientID)
		clientID++
	}
}

func (b *BattleServer) handleClientRequest(con net.Conn, id uint64) {
	connectionDropped := func() {
		println(fmt.Sprintf("[BattleServer] Client(%d) disconnected!", id))
		delete(b.connectedClients, id)
		con.Close()
	}
	defer connectionDropped()
	clientReader := bufio.NewReader(con)
	println(fmt.Sprintf("[BattleServer] New client(%d) connected! Waiting for client messages.", id))
	for {
		messageType, err := clientReader.ReadString('\n')
		if err != nil {
			println(fmt.Sprintf(err.Error()))
			return
		}
		message, err := clientReader.ReadString('\n')
		if err != nil {
			println(fmt.Sprintf(err.Error()))
			return
		}
		message = strings.TrimSpace(message)
		messageType = strings.TrimSpace(messageType)
		println(fmt.Sprintf("[BattleServer] Client(%d)->Server msg(%s): %s", id, messageType, message))
		b.GenerateResponse(con, id, messageType, message)
	}
}

type Header struct {
	Type string `json:"type"`
}

func (b *BattleServer) GenerateResponse(con net.Conn, id uint64, msgType string, message string) {
	// decode msg as json
	// check header
	switch msgType {
	case "Login":
		var loginMsg game.LoginMessage
		if toJson(message, &loginMsg) {
			b.Login(con, id, loginMsg)
		}
	case "CreateGame":
		var createGameMsg game.CreateGameMessage
		if toJson(message, &createGameMsg) {
			b.CreateGame(id, createGameMsg)
		}
	case "SelectFaction":
		var selectFactionMsg game.SelectFactionMessage
		if toJson(message, &selectFactionMsg) {
			b.SelectFaction(id, selectFactionMsg)
		}
	case "SelectUnits":
		var selectUnitsMsg game.SelectUnitsMessage
		if toJson(message, &selectUnitsMsg) {
			b.SelectUnits(id, selectUnitsMsg)
		}
	case "JoinGame":
		var joinGameMsg game.JoinGameMessage
		if toJson(message, &joinGameMsg) {
			b.JoinGame(id, joinGameMsg)
		}
	case "UnitAction":
		var targetedUnitActionMsg game.TargetedUnitActionMessage
		if toJson(message, &targetedUnitActionMsg) {
			b.UnitAction(id, targetedUnitActionMsg)
		}
	case "FreeAimAction":
		var freeAimActionMsg game.FreeAimActionMessage
		if toJson(message, &freeAimActionMsg) {
			b.FreeAimAction(id, freeAimActionMsg)
		}
	case "MapLoaded":
		b.MapLoaded(id)
	case "EndTurn":
		b.EndTurn(id)
	}
}

func toJson(message string, msg interface{}) bool {
	err := json.Unmarshal([]byte(message), &msg)
	if err != nil {
		println(err.Error())
		return false
	}
	return true
}

type UserConnection struct {
	raw        net.Conn
	id         uint64
	name       string
	activeGame string
	isReady    bool
}

func (b *BattleServer) respond(connection *UserConnection, messageType string, response any) {
	asJson, _ := json.Marshal(response)
	b.writeToClient(connection, []byte(messageType), asJson)
}
func (b *BattleServer) writeFromBuffer(userID uint64, msgType, msg []byte) {
	connection := b.connectedClients[userID]
	if connection == nil {
		return
	}
	b.writeToClient(connection, msgType, msg)
}
func (b *BattleServer) writeToClient(connection *UserConnection, messageType, response []byte) {
	println(fmt.Sprintf("[BattleServer] Server->Client(%d) msg(%s): %s", connection.id, string(messageType), string(response)))
	_, err := connection.raw.Write(append(messageType, '\n'))
	if err != nil {
		println(fmt.Sprintf(err.Error()))
		return
	}
	_, err = connection.raw.Write(append(response, '\n'))
	if err != nil {
		println(fmt.Sprintf(err.Error()))
		return
	}
}
func (b *BattleServer) Login(con net.Conn, userID uint64, msg game.LoginMessage) {
	userConnection := &UserConnection{raw: con, id: userID, name: msg.Username}
	b.connectedClients[userID] = userConnection
	b.respond(userConnection, "LoginResponse", game.ActionResponse{Success: true, Message: "Welcome to BattleGrounds"})
}

func (b *BattleServer) SelectFaction(userID uint64, msg game.SelectFactionMessage) {
	user := b.connectedClients[userID]
	gameID := user.activeGame
	if gameID == "" {
		b.respond(user, "SelectFactionResponse", game.ActionResponse{Success: false, Message: "You are not in a game"})
		return
	}
	gameInstance, exists := b.runningGames[gameID]
	if !exists {
		b.respond(user, "SelectFactionResponse", game.ActionResponse{Success: false, Message: "Game does not exist"})
		return
	}

	faction, exists := b.availableFactions[msg.FactionName]
	if !exists {
		b.respond(user, "SelectFactionResponse", game.ActionResponse{Success: false, Message: "Faction does not exist"})
		return
	}

	gameInstance.SetFaction(userID, faction)

	b.respond(user, "SelectFactionResponse", game.ActionResponse{Success: true, Message: "Faction selected"})

	if gameInstance.IsReady() {
		b.startGame(gameInstance)
	}
}

func (b *BattleServer) CreateGame(userId uint64, msg game.CreateGameMessage) {
	gameID := msg.GameIdentifier
	user := b.connectedClients[userId]
	if _, alreadyExists := b.runningGames[gameID]; alreadyExists {
		b.respond(user, "CreateGameResponse", game.ActionResponse{Success: false, Message: "Game already exists"})
		return
	}

	battleGame := NewGameInstance(userId, gameID, msg.Map, msg.IsPublic)
	battleGame.AddPlayer(userId)
	user.isReady = false
	user.activeGame = gameID

	b.runningGames[gameID] = battleGame

	b.respond(user, "CreateGameResponse", game.ActionResponse{Success: true, Message: "Game created"})
}

func (b *BattleServer) JoinGame(id uint64, msg game.JoinGameMessage) {
	gameID := msg.GameID
	user := b.connectedClients[id]
	if _, alreadyExists := b.runningGames[gameID]; !alreadyExists {
		b.respond(user, "JoinGameResponse", game.ActionResponse{Success: false, Message: "Game does not exist"})
		return
	}
	user.isReady = false
	user.activeGame = gameID
	gameInstance := b.runningGames[gameID]
	gameInstance.AddPlayer(id)

	b.respond(user, "JoinGameResponse", game.ActionResponse{Success: true, Message: "Game joined"})

	if gameInstance.IsReady() {
		b.startGame(gameInstance)
	}
}

func (b *BattleServer) startGame(battleGame *GameInstance) {
	println(fmt.Sprintf("[BattleServer] Starting game %s", battleGame.id))
	battleGame.Start()
	playerNames := make(map[uint64]string)
	playerFactions := battleGame.GetPlayerFactions()

	for _, playerID := range battleGame.players {
		user, exists := b.connectedClients[playerID]
		if !exists {
			println(fmt.Sprintf("[BattleServer] ERR -> Player %d does not exist", playerID))
			continue
		}
		playerNames[playerID] = user.name
	}

	for _, playerID := range battleGame.players {
		user := b.connectedClients[playerID]

		// broadcast game started event to all players, tell everyone who's turn it is
		units := battleGame.GetPlayerUnits(playerID)
		b.respond(user, "GameStarted", game.GameStartedMessage{
			GameID:           battleGame.id,
			PlayerNameMap:    playerNames,
			PlayerFactionMap: playerFactions,
			OwnID:            playerID,
			OwnUnits:         units,
			MapFile:          battleGame.mapFile,
		})
	}
}

var debugSpawnPositions = []voxel.Int3{
	{X: 2, Y: 1, Z: 2},
	{X: 6, Y: 1, Z: 2},
	{X: 4, Y: 1, Z: 13},
	{X: 4, Y: 1, Z: 11},
}
var debugSpawnCounter = 0

func (b *BattleServer) SelectUnits(userID uint64, msg game.SelectUnitsMessage) {
	user := b.connectedClients[userID]
	gameID := user.activeGame
	if gameID == "" {
		b.respond(user, "SelectUnitsResponse", game.ActionResponse{Success: false, Message: "You are not in a game"})
		return
	}
	gameInstance, exists := b.runningGames[gameID]
	if !exists {
		b.respond(user, "SelectUnitsResponse", game.ActionResponse{Success: false, Message: "Game does not exist"})
		return
	}
	for _, unitRequest := range msg.Units {
		if unitRequest.UnitTypeID >= uint64(len(b.availableUnits)) {
			b.respond(user, "SelectUnitsResponse", game.ActionResponse{Success: false, Message: fmt.Sprintf("Unit %d does not exist", unitRequest)})
			return
		}
	}

	for _, unitRequest := range msg.Units {
		unitChoice := unitRequest
		spawnedUnitDef := b.availableUnits[unitChoice.UnitTypeID]
		println(fmt.Sprintf("[BattleServer] %d selected unit %d", userID, spawnedUnitDef.ID))
		unit := game.NewUnitInstance(unitChoice.Name, spawnedUnitDef)
		unit.SetWeapon(unitChoice.Weapon)
		unit.SetControlledBy(userID)
		unit.SetVoxelMap(gameInstance.voxelMap)
		unit.SetSpawnPosition(debugSpawnPositions[debugSpawnCounter])
		debugSpawnCounter = (debugSpawnCounter + 1) % len(debugSpawnPositions)
		gameInstance.AddUnit(userID, unit) // sets the instance userID
	}

	b.respond(user, "SelectUnitsResponse", game.ActionResponse{Success: true, Message: "Units selected"})

	if gameInstance.IsReady() {
		b.startGame(gameInstance)
	}
}

func (b *BattleServer) UnitAction(userID uint64, msg game.UnitActionMessage) {
	user := b.connectedClients[userID]
	gameID := user.activeGame
	if gameID == "" {
		b.respond(user, "TargetedUnitActionResponse", game.ActionResponse{Success: false, Message: "You are not in a game"})
		return
	}
	gameInstance, exists := b.runningGames[gameID]
	if !exists {
		b.respond(user, "TargetedUnitActionResponse", game.ActionResponse{Success: false, Message: "Game does not exist"})
		return
	}

	if msg.UnitID() >= uint64(len(gameInstance.units)) {
		b.respond(user, "TargetedUnitActionResponse", game.ActionResponse{Success: false, Message: "Unit does not exist"})
		return
	}
	unit := gameInstance.units[msg.UnitID()]

	if unit.ControlledBy() != userID {
		b.respond(user, "TargetedUnitActionResponse", game.ActionResponse{Success: false, Message: "You do not control this unit"})
		return
	}

	if !unit.CanAct() {
		b.respond(user, "TargetedUnitActionResponse", game.ActionResponse{Success: false, Message: "Unit cannot act"})
		return
	}

	action := gameInstance.GetServerActionForUnit(msg, unit)

	// get action for this unit, check if the target is valid
	if !action.IsValid() {
		b.respond(user, "TargetedUnitActionResponse", game.ActionResponse{Success: false, Message: "Action is not valid"})
		return
	}
	// this setup won't work: the client cannot know how to execute the action
	// example movement: only the server knows if an reaction shot is triggered
	// so executing that same action on the client will not work
	// we need to adapt the response to include the result of the action
	mb := game.NewMessageBuffer(gameInstance.players, b.writeFromBuffer)
	action.Execute(mb)
	// so for the movement example we want to communicate
	// - the unit moved to another position than the client expected
	// - the unit can now see another unit

	unit.EndTurn()

	mb.SendAll()
}

func (b *BattleServer) FreeAimAction(userID uint64, msg game.FreeAimActionMessage) {
	user := b.connectedClients[userID]
	gameID := user.activeGame
	if gameID == "" {
		b.respond(user, "FreeAimActionResponse", game.ActionResponse{Success: false, Message: "You are not in a game"})
		return
	}
	gameInstance, exists := b.runningGames[gameID]
	if !exists {
		b.respond(user, "FreeAimActionResponse", game.ActionResponse{Success: false, Message: "Game does not exist"})
		return
	}

	if msg.GameUnitID >= uint64(len(gameInstance.units)) {
		b.respond(user, "FreeAimActionResponse", game.ActionResponse{Success: false, Message: "Unit does not exist"})
		return
	}
	unit := gameInstance.units[msg.GameUnitID]

	if unit.ControlledBy() != userID {
		b.respond(user, "FreeAimActionResponse", game.ActionResponse{Success: false, Message: "You do not control this unit"})
		return
	}

	if !unit.CanAct() {
		b.respond(user, "FreeAimActionResponse", game.ActionResponse{Success: false, Message: "Unit cannot act"})
		return
	}

	action := gameInstance.GetServerActionForUnit(msg, unit)

	// get action for this unit, check if the target is valid
	if !action.IsValid() {
		b.respond(user, "FreeAimActionResponse", game.ActionResponse{Success: false, Message: "Action is not valid"})
		return
	}
	// this setup won't work: the client cannot know how to execute the action
	// example movement: only the server knows if an reaction shot is triggered
	// so executing that same action on the client will not work
	// we need to adapt the response to include the result of the action
	mb := game.NewMessageBuffer(gameInstance.players, b.writeFromBuffer)
	action.Execute(mb)
	// so for the movement example we want to communicate
	// - the unit moved to another position than the client expected
	// - the unit can now see another unit

	unit.EndTurn()

	mb.SendAll()
}

func (b *BattleServer) EndTurn(userID uint64) {
	user := b.connectedClients[userID]
	gameID := user.activeGame
	if gameID == "" {
		b.respond(user, "EndTurnResponse", game.ActionResponse{Success: false, Message: "You are not in a game"})
		return
	}
	gameInstance, exists := b.runningGames[gameID]
	if !exists {
		b.respond(user, "EndTurnResponse", game.ActionResponse{Success: false, Message: "Game does not exist"})
		return
	}

	if !gameInstance.IsPlayerTurn(userID) {
		b.respond(user, "EndTurnResponse", game.ActionResponse{Success: false, Message: "It is not your turn"})
		return
	}

	b.SendNextPlayer(gameInstance)
}

func (b *BattleServer) SendNextPlayer(gameInstance *GameInstance) {
	nextPlayer := gameInstance.NextPlayer()
	for _, playerID := range gameInstance.players {
		connectedUser := b.connectedClients[playerID]
		b.respond(connectedUser, "NextPlayer", game.NextPlayerMessage{
			CurrentPlayer: nextPlayer,
			YourTurn:      playerID == nextPlayer,
		})
	}
}

func (b *BattleServer) MapLoaded(userID uint64) {
	// get the player and mark him as ready
	user, exists := b.connectedClients[userID]
	if !exists {
		println(fmt.Sprintf("[BattleServer] ERR -> Player %d does not exist", userID))
		return
	}

	gameID := user.activeGame
	if gameID == "" {
		b.respond(user, "MapLoadedResponse", game.ActionResponse{Success: false, Message: "You are not in a game"})
		return
	}

	gameInstance, exists := b.runningGames[gameID]
	if !exists {
		b.respond(user, "MapLoadedResponse", game.ActionResponse{Success: false, Message: "Game does not exist"})
		return
	}

	user.isReady = true

	allReady := true
	for _, playerID := range gameInstance.players {
		connectedUser := b.connectedClients[playerID]
		if !connectedUser.isReady {
			allReady = false
			break
		}
	}

	if allReady {
		b.SendNextPlayer(gameInstance)
	}
}
func NewBattleServer() *BattleServer {
	return &BattleServer{
		availableMaps:     make(map[string]string),          // filename -> display name
		availableFactions: make(map[string]*Faction, 0),     // faction name -> faction
		connectedClients:  make(map[uint64]*UserConnection), // client id -> client
		runningGames:      make(map[string]*GameInstance),   // game id -> game
	}
}

// Client -> Server
// Login(name, faction)
// JoinGame(id)
// GetGames()
// GetMaps()
// GetUnitTypes()
// CreateGame(id, public, map)
// AddUnitDefinition(availableUnits)
// StartGame()
// EndTurn()
// UnitMove(unit, target)
// UnitShot(unit, direction)

// Server -> Client
// ActionResponse(success, message)
// JoinGameResponse(success, message)
// MapList(maps)
// UnitList(availableUnits)
// GameList(games)
// GameStarted()
// TurnStarted()
// UnitMoved(unit, target)
// UnitShot(unit, direction)
