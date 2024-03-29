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
	availableItems    map[string]*game.ItemDefinition

	connectedClients map[uint64]*UserConnection

	// game instances
	runningGames map[string]*game.GameInstance
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
	case "SelectDeployment":
		var deployMsg game.DeploymentMessage
		if FromJson(message, &deployMsg) {
			b.SelectDeployment(id, deployMsg)
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
	case "ThrownUnitAction":
		var thrownUnitAction game.ThrownUnitActionMessage
		if FromJson(message, &thrownUnitAction) {
			b.UnitAction(id, thrownUnitAction)
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
	util.LogNetworkInfo(fmt.Sprintf("Server started on %s", endpoint))

	endianess, err := util.GetSystemNativeEndianess()
	if err != nil {
		util.LogSystemInfo(fmt.Sprintf("[GetSystemNativeEndianess] ERR -> %s", err.Error()))
	} else {
		util.LogSystemInfo(fmt.Sprintf("[GetSystemNativeEndianess] %s", endianess.ToString()))
	}
	for {
		con, listenError := listener.Accept()
		if listenError != nil {
			log.Println(listenError)
			continue
		}
		// If you want, you can increment a counter here and inject to handleClientRequest below as client identifier
		go b.handleClientRequest(con, clientID)
		clientID++
	}
}

func (b *BattleServer) handleClientRequest(con net.Conn, id uint64) {
	connectionDropped := func() {
		util.LogNetworkInfo(fmt.Sprintf("[BattleServer] Client(%d) disconnected!", id))
		delete(b.connectedClients, id)
		con.Close()
	}
	defer connectionDropped()
	clientReader := bufio.NewReader(con)
	util.LogNetworkInfo(fmt.Sprintf("[BattleServer] New client(%d) connected! Waiting for client messages.", id))
	for {
		messageType, err := clientReader.ReadString('\n')
		if err != nil {
			util.LogNetworkError(fmt.Sprintf(err.Error()))
			return
		}
		message, err := clientReader.ReadString('\n')
		if err != nil {
			util.LogNetworkError(fmt.Sprintf(err.Error()))
			return
		}
		message = strings.TrimSpace(message)
		messageType = strings.TrimSpace(messageType)
		//println(fmt.Sprintf("[BattleServer] Client(%d)->Server msg(%s): %s", id, messageType, message))
		util.LogNetworkDebug(fmt.Sprintf("\n[Server] FROM Client(%d) msg(%s):\n%s\n", id, messageType, message))

		b.GenerateResponse(con, id, messageType, message)
	}
}

func FromJson(message string, msg interface{}) bool {
	err := json.Unmarshal([]byte(message), &msg)
	if err != nil {
		util.LogIOError(err.Error())
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
	util.LogNetworkDebug(fmt.Sprintf("\n[Server] TO Client(%d) msg(%s):\n%v\n", connection.id, string(messageType), string(response)))
	_, err := connection.raw.Write(append(messageType, '\n'))
	if err != nil {
		util.LogNetworkError(fmt.Sprintf(err.Error()))
		return
	}
	_, err = connection.raw.Write(append(response, '\n'))
	if err != nil {
		util.LogNetworkError(fmt.Sprintf(err.Error()))
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

	battleGame := game.NewGameInstanceWithDetails(gameID, msg.Map, msg.MissionDetails)
	//battleGame := game.NewGameInstanceWithBiome(gameID, game.NewBiomeDesert())
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
	util.LogGameInfo(fmt.Sprintf("[BattleServer] Starting game %s", battleGame.GetID()))
	battleGame.Start()
	playerNames := make(map[uint64]string)
	playerFactions := battleGame.GetPlayerFactions()

	for _, playerID := range battleGame.GetPlayerIDs() {
		user, exists := b.connectedClients[playerID]
		if !exists {
			util.LogGameError(fmt.Sprintf("[BattleServer] ERR -> Player %d does not exist", playerID))
			continue
		}
		playerNames[playerID] = user.name
	}

	// also send the initial LOS state
	battleGame.InitLOSAndPressure()

	for spawnIndex, playerID := range battleGame.GetPlayerIDs() {
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
			SpawnIndex:       uint64(spawnIndex),
			OwnUnits:         units,
			MapFile:          battleGame.GetMapFile(),
			LOSMatrix:        whoCanSeeWho,
			PressureMatrix:   pressure,
			VisibleUnits:     visibleUnits,
			MissionDetails:   battleGame.GetMissionDetails(),
		})
	}
}
func (b *BattleServer) SelectDeployment(userID uint64, msg game.DeploymentMessage) {
	user, gameInstance, exists := b.getUserAndGame(userID)
	if !exists {
		return
	}

	util.LogNetworkDebug(fmt.Sprintf("[BattleServer] %d selected deployment: %v", userID, msg.Deployment))

	if !gameInstance.TryDeploy(userID, msg.Deployment) {
		b.respond(user, "DeploymentResponse", game.ActionResponse{Success: false, Message: "Deployment failed"})
		return
	}

	b.respond(user, "DeploymentResponse", game.ActionResponse{Success: true, Message: "Deployment successful"})

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
		gameInstance.DeploymentDone()
		b.SendNextPlayer(gameInstance)
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
	assetLoader := gameInstance.GetAssets()
	// ** We are now adding the units to game world **
	for _, unitRequest := range msg.Units {
		unitChoice := unitRequest
		// get unit definition
		spawnedUnitDef := b.availableUnits[unitChoice.UnitTypeID]

		// create unit
		unit := game.NewUnitInstance(assetLoader, unitChoice.Name, spawnedUnitDef)

		// assign weapon
		chosenWeapon, weaponIsOK := b.availableWeapons[unitChoice.Weapon]
		if weaponIsOK {
			unit.SetWeapon(game.NewWeapon(chosenWeapon))
		} else {
			util.LogGameError(fmt.Sprintf("[BattleServer] %d tried to select weapon '%s', but it does not exist", userID, unitChoice.Weapon))
		}

		// assign items
		for _, itemName := range unitChoice.Items {
			chosenItem, itemIsOK := b.availableItems[itemName]
			if itemIsOK {
				unit.AddItem(game.NewItem(chosenItem))
			} else {
				util.LogGameError(fmt.Sprintf("[BattleServer] %d tried to select item '%s', but it does not exist", userID, itemName))
			}
		}

		unit.SetControlledBy(userID)
		unit.SetVoxelMap(gameInstance.GetVoxelMap())
		//unit.SetForward(voxel.Int3{Z: 1})

		unitID := gameInstance.ServerSpawnUnit(userID, unit) // sets the instance userID
		unit.AutoSetStanceAndForwardAndUpdateMap()
		unit.StartStanceAnimation()

		util.LogGlobalUnitDebug(unit.DebugString("ServerSpawnUnit(+UpdateAnim)"))
		util.LogGameInfo(fmt.Sprintf("[BattleServer] User %d selected unit of type %d: %s(%d)", userID, spawnedUnitDef.ID, unitChoice.Name, unitID))
	}

	b.respond(user, "SelectUnitsResponse", game.ActionResponse{Success: true, Message: "Units selected"})

	if gameInstance.IsReady() {
		b.startGame(gameInstance)
	}
}
func (b *BattleServer) UnitAction(userID uint64, msg game.UnitActionMessage) {
	user, gameInstance, isValid := b.getUserAndGame(userID)
	if !isValid {
		return
	}

	unit, isValidUnit := b.getUnit(gameInstance, user, msg.UnitID())
	if !isValidUnit {
		return
	}

	action := GetServerActionForUnit(gameInstance, msg, unit)

	// get action for this unit, check if the targets is valid
	if isValidAction, reason := action.IsValid(); !isValidAction {
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

func (b *BattleServer) getUnit(gameInstance *game.GameInstance, user *UserConnection, unitID uint64) (*game.UnitInstance, bool) {
	if unitID >= uint64(len(gameInstance.GetAllUnits())) {
		b.respondWithMessage(user, game.ActionResponse{Success: false, Message: "Unit does not exist"})
		return nil, false
	}
	unit, unitExists := gameInstance.GetUnit(unitID)
	if !unitExists {
		b.respondWithMessage(user, game.ActionResponse{Success: false, Message: "Unit does not exist"})
		return nil, false
	}

	if unit.ControlledBy() != user.id {
		b.respondWithMessage(user, game.ActionResponse{Success: false, Message: "You do not control this unit"})
		return nil, false
	}

	if !unit.CanAct() {
		b.respondWithMessage(user, game.ActionResponse{Success: false, Message: "Unit cannot act"})
		return nil, false
	}
	return unit, true
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

	if !gameInstance.AllUnitsDeployed() {
		b.respondWithMessage(user, game.ActionResponse{Success: false, Message: "Not all units are deployed"})
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
	util.LogGameInfo(fmt.Sprintf("[BattleServer] Game '%s' removed", instance.GetID()))
}
func (b *BattleServer) SendNextPlayer(gameInstance *game.GameInstance) {
	//println("[BattleServer] Ending turn. New map state:")
	//gameInstance.GetVoxelMap().PrintArea2D(16, 16)
	/*
		for _, unit := range gameInstance.GetAllUnits() {
				println(fmt.Sprintf("[BattleServer] > Unit %s(%d): %v", unit.GetName(), unit.Attacker(), unit.GetBlockPosition()))
		}

	*/
	util.LogNetworkInfo(fmt.Sprintf("[BattleServer] New turn for game %s", gameInstance.GetID()))

	nextPlayer := gameInstance.NextPlayer()
	for _, playerID := range gameInstance.GetPlayerIDs() {
		connectedUser := b.connectedClients[playerID]
		b.respondWithMessage(connectedUser, game.NextPlayerMessage{
			CurrentPlayer: nextPlayer,
			YourTurn:      playerID == nextPlayer,
		})
	}

	// the server would now wait for messages from the next player
	// if it is an AI player, we could generate the moves for it right here instead.
	// but that would blur the line and we would lose interesting options
}

func (b *BattleServer) SendStartDeployment(gameInstance *game.GameInstance) {
	for _, playerID := range gameInstance.GetPlayerIDs() {
		connectedUser := b.connectedClients[playerID]
		b.respondWithMessage(connectedUser, game.StartDeploymentMessage{})
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
		if gameInstance.GetMissionDetails().Placement == game.PlacementModeManual {
			for _, playerID := range gameInstance.GetPlayerIDs() {
				connectedUser := b.connectedClients[playerID]
				connectedUser.isReady = false // reset ready, wait for deployment
			}
			b.SendStartDeployment(gameInstance)
		} else {
			b.SendNextPlayer(gameInstance) // this is the right thing, if the place is pre-determined..
		}
	}
}

func (b *BattleServer) AddWeapon(weaponDefinition game.WeaponDefinition) {
	b.availableWeapons[weaponDefinition.UniqueName] = &weaponDefinition
}

func (b *BattleServer) AddItem(itemDefinition game.ItemDefinition) {
	b.availableItems[itemDefinition.UniqueName] = &itemDefinition
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
		availableItems:    make(map[string]*game.ItemDefinition),
	}
}
