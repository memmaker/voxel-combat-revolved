package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/game"
	"log"
	"net"
	"strings"
)

type BattleServer struct {
	// global state for the whole server
	availableMaps     map[string]string
	availableFactions map[string]*game.Faction
	availableUnits    []*game.UnitDefinition
	availableWeapons  map[string]*game.WeaponDefinition

	connectedClients map[uint64]*UserConnection

	// game instances
	runningGames map[string]*game.GameInstance
}

func (b *BattleServer) AddMap(name string, filename string) {
	b.availableMaps[name] = filename
}
func (b *BattleServer) AddFaction(def game.FactionDefinition) {
	faction := &game.Faction{Name: def.Name}
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

	endianess, err := util.GetSystemNativeEndianess()
	if err != nil {
		println(fmt.Sprintf("[GetSystemNativeEndianess] ERR -> %s", err.Error()))
	} else {
		println(fmt.Sprintf("[GetSystemNativeEndianess] %s", endianess.ToString()))
	}
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
		//println(fmt.Sprintf("[BattleServer] Client(%d)->Server msg(%s): %s", id, messageType, message))
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
		if FromJson(message, &loginMsg) {
			b.Login(con, id, loginMsg)
		}
	case "CreateGame":
		var createGameMsg game.CreateGameMessage
		if FromJson(message, &createGameMsg) {
			b.CreateGame(id, createGameMsg)
		}
	case "SelectFaction":
		var selectFactionMsg game.SelectFactionMessage
		if FromJson(message, &selectFactionMsg) {
			b.SelectFaction(id, selectFactionMsg)
		}
	case "SelectUnits":
		var selectUnitsMsg game.SelectUnitsMessage
		if FromJson(message, &selectUnitsMsg) {
			b.SelectUnits(id, selectUnitsMsg)
		}
	case "JoinGame":
		var joinGameMsg game.JoinGameMessage
		if FromJson(message, &joinGameMsg) {
			b.JoinGame(id, joinGameMsg)
		}
	case "UnitAction":
		var targetedUnitActionMsg game.TargetedUnitActionMessage
		if FromJson(message, &targetedUnitActionMsg) {
			b.UnitAction(id, targetedUnitActionMsg)
		}
	case "FreeAimAction":
		var freeAimActionMsg game.FreeAimActionMessage
		if FromJson(message, &freeAimActionMsg) {
			b.UnitAction(id, freeAimActionMsg)
		}
	case "MapLoaded":
		var mapLoadedMsg game.MapLoadedMessage
		if FromJson(message, &mapLoadedMsg) {
			b.MapLoaded(id, mapLoadedMsg)
		}
	case "Reload":
		var reloadMsg game.UnitMessage
		if FromJson(message, &reloadMsg) {
			b.Reload(id, reloadMsg.UnitID())
		}
	case "DebugRequest":
		var debugRequestMsg game.DebugRequest
		if FromJson(message, &debugRequestMsg) {
			b.DebugRequest(id, debugRequestMsg)
		}
	case "EndTurn":
		b.EndTurn(id)
	}
}

func FromJson(message string, msg interface{}) bool {
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
func (b *BattleServer) respondWithMessage(connection *UserConnection, response game.Message) {
	b.respond(connection, response.MessageType(), response)
}

func (b *BattleServer) writeFromBuffer(userID uint64, msgType, msg []byte) {
	connection := b.connectedClients[userID]
	if connection == nil {
		return
	}
	b.writeToClient(connection, msgType, msg)
}
func (b *BattleServer) writeToClient(connection *UserConnection, messageType, response []byte) {
	//println(fmt.Sprintf("[BattleServer] Server->Client(%d) msg(%s): %s", connection.id, string(messageType), string(response)))
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

	battleGame := game.NewGameInstanceWithMap(gameID, msg.Map)
	battleGame.SetEnvironment("Server")
	battleGame.AddPlayer(userId)

	listOfBlocks := game.GetDebugBlockNames()
	indexMap := util.CreateIndexMapFromDirectory("assets/textures/blocks/star_odyssey", listOfBlocks)

	bl := game.NewBlockLibrary(listOfBlocks, indexMap)
	bl.ApplyGameplayRules(battleGame)

	battleGame.SetBlockLibrary(bl)
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

func (b *BattleServer) startGame(battleGame *game.GameInstance) {
	println(fmt.Sprintf("[BattleServer] Starting game %s", battleGame.GetID()))
	battleGame.Start()
	playerNames := make(map[uint64]string)
	playerFactions := battleGame.GetPlayerFactions()

	for _, playerID := range battleGame.GetPlayerIDs() {
		user, exists := b.connectedClients[playerID]
		if !exists {
			println(fmt.Sprintf("[BattleServer] ERR -> Player %d does not exist", playerID))
			continue
		}
		playerNames[playerID] = user.name
	}

	// also send the initial LOS state
	battleGame.InitLOSAndPressure()

	for _, playerID := range battleGame.GetPlayerIDs() {
		user := b.connectedClients[playerID]

		// broadcast game started event to all players, tell everyone who's turn it is
		units := battleGame.GetPlayerUnits(playerID)
		whoCanSeeWho, visibleUnits := battleGame.GetLOSState(playerID)
		pressure := battleGame.GetPressureMatrix()

		b.respond(user, "GameStarted", game.GameStartedMessage{
			GameID:           battleGame.GetID(),
			PlayerNameMap:    playerNames,
			PlayerFactionMap: playerFactions,
			OwnID:            playerID,
			OwnUnits:         units,
			MapFile:          battleGame.GetMapFile(),
			LOSMatrix:        whoCanSeeWho,
			PressureMatrix:   pressure,
			VisibleUnits:     visibleUnits,
		})
	}
}

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
			b.respond(user, "SelectUnitsResponse", game.ActionResponse{Success: false, Message: fmt.Sprintf("Unit %d does not exist", unitRequest.UnitTypeID)})
			return
		}
	}
	// we are now adding the units to game world
	for _, unitRequest := range msg.Units {
		unitChoice := unitRequest
		spawnedUnitDef := b.availableUnits[unitChoice.UnitTypeID]
		unit := game.NewUnitInstance(unitChoice.Name, spawnedUnitDef)
		chosenWeapon, weaponIsOK := b.availableWeapons[unitChoice.Weapon]
		if weaponIsOK {
			unit.SetWeapon(game.NewWeapon(chosenWeapon))
		} else {
			println(fmt.Sprintf("[BattleServer] %d tried to select weapon '%s', but it does not exist", userID, unitChoice.Weapon))
		}
		unit.SetControlledBy(userID)
		unit.SetVoxelMap(gameInstance.GetVoxelMap())
		//unit.SetForward(voxel.Int3{Z: 1})

		unitID := gameInstance.ServerSpawnUnit(userID, unit) // sets the instance userID
		unit.AutoSetStanceAndForwardAndUpdateMap()
		unit.StartStanceAnimation()

		println(unit.DebugString("ServerSpawnUnit(+UpdateAnim)"))
		println(fmt.Sprintf("[BattleServer] User %d selected unit of type %d: %s(%d)", userID, spawnedUnitDef.ID, unitChoice.Name, unitID))
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
		b.respondWithMessage(user, game.ActionResponse{Success: false, Message: "You are not in a game"})
		return
	}

	gameInstance, exists := b.runningGames[gameID]
	if !exists {
		b.respondWithMessage(user, game.ActionResponse{Success: false, Message: "Game does not exist"})
		return
	}

	if msg.UnitID() >= uint64(len(gameInstance.GetAllUnits())) {
		b.respondWithMessage(user, game.ActionResponse{Success: false, Message: "Unit does not exist"})
		return
	}
	unit, unitExists := gameInstance.GetUnit(msg.UnitID())

	if !unitExists {
		b.respondWithMessage(user, game.ActionResponse{Success: false, Message: "Unit does not exist"})
		return
	}

	if unit.ControlledBy() != userID {
		b.respondWithMessage(user, game.ActionResponse{Success: false, Message: "You do not control this unit"})
		return
	}

	if !unit.CanAct() {
		b.respondWithMessage(user, game.ActionResponse{Success: false, Message: "Unit cannot act"})
		return
	}

	action := GetServerActionForUnit(gameInstance, msg, unit)

	// get action for this unit, check if the targets is valid
	if isValid, reason := action.IsValid(); !isValid {
		b.respondWithMessage(user, game.ActionResponse{Success: false, Message: "Action is not valid: " + reason})
		return
	}

	mb := game.NewMessageBuffer(gameInstance.GetPlayerIDs(), b.writeFromBuffer)
	action.Execute(mb)

	if action.IsTurnEnding() {
		unit.EndTurn()
	}

	mb.SendAll()
}
func (b *BattleServer) EndTurn(userID uint64) {
	user := b.connectedClients[userID]
	gameID := user.activeGame
	if gameID == "" {
		b.respondWithMessage(user, game.ActionResponse{Success: false, Message: "You are not in a game"})
		return
	}
	gameInstance, exists := b.runningGames[gameID]
	if !exists {
		b.respondWithMessage(user, game.ActionResponse{Success: false, Message: "Game does not exist"})
		return
	}

	if !gameInstance.IsPlayerTurn(userID) {
		b.respondWithMessage(user, game.ActionResponse{Success: false, Message: "It is not your turn"})
		return
	}
	isGameOver, winner := gameInstance.IsGameOver()
	if isGameOver {
		b.SendGameOver(gameInstance, winner)
	} else {
		b.SendNextPlayer(gameInstance)
	}
}

func (b *BattleServer) SendGameOver(instance *game.GameInstance, winner uint64) {
	println(fmt.Sprintf("[BattleServer] Game '%s' is over, winner is %d", instance.GetID(), winner))
	for _, playerID := range instance.GetPlayerIDs() {
		connectedUser := b.connectedClients[playerID]
		connectedUser.activeGame = ""
		connectedUser.isReady = false
		b.respond(connectedUser, "GameOver", game.GameOverMessage{
			WinnerID: winner,
			YouWon:   playerID == winner,
		})
	}
	delete(b.runningGames, instance.GetID())
	println(fmt.Sprintf("[BattleServer] Game '%s' removed", instance.GetID()))
}
func (b *BattleServer) SendNextPlayer(gameInstance *game.GameInstance) {
	//println("[BattleServer] Ending turn. New map state:")
	//gameInstance.GetVoxelMap().PrintArea2D(16, 16)
	/*
	for _, unit := range gameInstance.GetAllUnits() {
		println(fmt.Sprintf("[BattleServer] > Unit %s(%d): %v", unit.GetName(), unit.UnitID(), unit.GetBlockPosition()))
	}

	*/

	nextPlayer := gameInstance.NextPlayer()
	for _, playerID := range gameInstance.GetPlayerIDs() {
		connectedUser := b.connectedClients[playerID]
		b.respondWithMessage(connectedUser, game.NextPlayerMessage{
			CurrentPlayer: nextPlayer,
			YourTurn:      playerID == nextPlayer,
		})
	}
}

func (b *BattleServer) MapLoaded(userID uint64, msg game.MapLoadedMessage) {
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
	for _, playerID := range gameInstance.GetPlayerIDs() {
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

func (b *BattleServer) AddWeapon(weaponDefinition game.WeaponDefinition) {
	b.availableWeapons[weaponDefinition.UniqueName] = &weaponDefinition
}

func (b *BattleServer) Reload(userID uint64, unitID uint64) {
	user, gameInstance, exists := b.getUserAndGame(userID)
	if !exists {
		return
	}

	unit, unitExists := gameInstance.GetUnit(unitID)
	if !unitExists {
		b.respond(user, "ActionResponse", game.ActionResponse{Success: false, Message: "Unit does not exist"})
		return
	}

	if !unit.CanAct() {
		b.respond(user, "ActionResponse", game.ActionResponse{Success: false, Message: "Unit cannot act"})
		return
	}

	if !unit.CanReload() {
		b.respond(user, "ActionResponse", game.ActionResponse{Success: false, Message: "Unit cannot reload"})
		return
	}

	unit.Reload()

	b.respond(user, "Reload", game.UnitMessage{GameUnitID: unit.UnitID()})
}

func (b *BattleServer) DebugRequest(userID uint64, msg game.DebugRequest) {
	user, gameInstance, exists := b.getUserAndGame(userID)
	if !exists {
		return
	}

	debugState := gameInstance.DebugGetCompleteState()
	b.respond(user, "DebugResponse", debugState)
}

func (b *BattleServer) getUserAndGame(userID uint64) (*UserConnection, *game.GameInstance, bool) {
	user, exists := b.connectedClients[userID]
	if !exists {
		b.respond(user, "ActionResponse", game.ActionResponse{Success: false, Message: "User does not exist"})
		return nil, nil, false
	}

	gameID := user.activeGame

	if gameID == "" {
		b.respond(user, "ActionResponse", game.ActionResponse{Success: false, Message: "You are not in a game"})
		return nil, nil, false
	}

	gameInstance, exists := b.runningGames[gameID]
	if !exists {
		b.respond(user, "ActionResponse", game.ActionResponse{Success: false, Message: "Game does not exist"})
		return nil, nil, false
	}

	return user, gameInstance, true
}

func NewBattleServer() *BattleServer {
	return &BattleServer{
		availableMaps:     make(map[string]string),             // filename -> display name
		availableFactions: make(map[string]*game.Faction),      // faction name -> faction
		connectedClients:  make(map[uint64]*UserConnection),    // client id -> client
		runningGames:      make(map[string]*game.GameInstance), // game id -> game
		availableWeapons:  make(map[string]*game.WeaponDefinition),
	}
}
