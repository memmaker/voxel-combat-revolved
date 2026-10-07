//go:build client

package main

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/faiface/mainthread"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/memmaker/battleground/client"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/game"
	"github.com/memmaker/battleground/server"
)

const hostPort = "9999"

func runGame() {
	content, err := game.LoadContent()
	if err != nil {
		panic(err)
	}
	mainthread.Call(func() {
		settings := client.NewClientSettingsFromFile("settings.json")
		window, terminate := util.InitOpenGLWindow(settings.Title, settings.Width, settings.Height, settings.FullScreen)
		defer terminate()

		// command line mode for scripted tests: bgclient create|join <host:port>
		if len(os.Args) > 2 {
			runCommandLineMatch(window, settings, content, os.Args[1], os.Args[2])
			return
		}

		menu := client.NewMenu(window, &settings, content)
		for {
			var config client.MatchConfig
			if os.Getenv("BT_QUICKSTART") != "" {
				config = quickstartConfig(content, 1)
			} else {
				config = menu.Run()
			}
			if config.Mode == client.ModeQuit {
				return
			}
			for {
				outcome := playMatch(window, menu, settings, content, config)
				if outcome == client.OutcomeQuit {
					return
				}
				if outcome != client.OutcomeRematch {
					break
				}
			}
		}
	})
}

// quickstartConfig is a default match against the AI (BT_QUICKSTART=1), used for testing.
func quickstartConfig(content game.Content, factionIndex int) client.MatchConfig {
	faction := content.Factions[factionIndex]
	config := client.MatchConfig{
		Mode:            client.ModeVsAI,
		PlayerName:      "Player",
		MapFile:         game.WithRandomSeed(content.Maps[0].File, rand.New(rand.NewSource(time.Now().UnixNano()))),
		Scenario:        game.MissionScenarioDeathmatch,
		Placement:       game.PlacementModeRandom,
		FactionName:     faction.Name,
		OpponentFaction: content.Factions[(factionIndex+1)%len(content.Factions)].Name,
		Difficulty:      game.AIDifficultyFromEnv(),
	}
	unitType := faction.Units[0].ID
	config.Squad = []game.UnitChoice{
		{UnitTypeID: unitType, Name: "Gnarg", Weapon: "M1911 Pistol", Items: []string{"Frag Grenade"}},
		{UnitTypeID: unitType, Name: "Gorn", Weapon: "M16 Rifle", Items: []string{"Frag Grenade"}},
		{UnitTypeID: unitType, Name: "Grimbel", Weapon: "Mossberg 500", Items: []string{"Smoke Grenade"}},
	}
	return config
}

// matchSetup drives the lobby protocol (login, create/join, faction, units) for one connection
// and reports when the game has started.
type matchSetup struct {
	started chan game.GameStartedMessage
	failed  chan string
	created chan struct{}
}

func startMatchSetup(con *game.ServerConnection, setup game.AIGameSetup, alreadyLoggedIn bool) *matchSetup {
	m := &matchSetup{
		started: make(chan game.GameStartedMessage, 1),
		failed:  make(chan string, 1),
		created: make(chan struct{}, 1),
	}
	steps := []func() error{
		func() error { return con.Login(setup.Username) },
		func() error {
			if setup.CreateGame {
				return con.CreateGame(setup.MapFile, setup.GameID, setup.Mission, true)
			}
			return con.JoinGame(setup.GameID)
		},
		func() error { return con.SelectFaction(setup.FactionName) },
		func() error { return con.SelectUnits(setup.Units) },
	}
	if alreadyLoggedIn {
		steps = steps[1:]
	}
	next := 0
	advance := func() {
		if next < len(steps) {
			step := steps[next]
			next++
			util.MustSend(step())
		}
	}
	con.SetEventHandler(func(msg game.StringMessage) {
		switch msg.MessageType {
		case "LoginResponse", "CreateGameResponse", "JoinGameResponse", "SelectFactionResponse", "SelectUnitsResponse":
			var response game.ActionResponse
			if !util.FromJson(msg.Message, &response) {
				return
			}
			if !response.Success {
				m.failed <- response.Message
				return
			}
			if msg.MessageType == "CreateGameResponse" {
				m.created <- struct{}{}
			}
			advance()
		case "GameStarted":
			var info game.GameStartedMessage
			if util.FromJson(msg.Message, &info) {
				m.started <- info
			}
		}
	})
	advance()
	return m
}

// waitForStart shows a status screen until the game starts, fails or the player cancels.
func (m *matchSetup) waitForStart(menu *client.Menu, message string) (game.GameStartedMessage, bool) {
	var info game.GameStartedMessage
	started := false
	failure := ""
	ok := menu.ShowStatus(message, func() bool {
		select {
		case info = <-m.started:
			started = true
		case failure = <-m.failed:
		default:
		}
		return started || failure != ""
	})
	if failure != "" {
		menu.ShowStatus("Could not start the match: "+failure, func() bool { return false })
		return info, false
	}
	return info, ok && started
}

func playMatch(window *glfw.Window, menu *client.Menu, settings client.ClientSettings, content game.Content, config client.MatchConfig) client.MatchOutcome {
	switch config.Mode {
	case client.ModeVsAI:
		return playAgainstAI(window, menu, settings, content, config)
	case client.ModeHost:
		return hostMatch(window, menu, settings, config)
	case client.ModeJoin:
		return joinMatch(window, menu, settings, config)
	}
	return client.OutcomeMenu
}

var matchCounter int

func playerSetup(config client.MatchConfig, gameID string, create bool) game.AIGameSetup {
	return game.AIGameSetup{
		Username:    config.PlayerName,
		GameID:      gameID,
		CreateGame:  create,
		MapFile:     config.MapFile,
		Mission:     config.Mission(),
		FactionName: config.FactionName,
		Units:       config.Squad,
	}
}

func playAgainstAI(window *glfw.Window, menu *client.Menu, settings client.ClientSettings, content game.Content, config client.MatchConfig) client.MatchOutcome {
	// singleplayer runs the server in-process, connected through in-memory pipes
	battleServer := NewBattleServer()
	game.Dial = battleServer.DialMem
	matchCounter++
	gameID := fmt.Sprintf("local-%d", matchCounter)

	conn, _ := battleServer.DialMem("")
	connection := game.NewServerConnectionFromConn(conn, nil)
	setup := startMatchSetup(connection, playerSetup(config, gameID, true), false)

	// the AI joins once the game exists
	go func() {
		select {
		case <-setup.created:
		case <-time.After(10 * time.Second):
			return
		}
		ai := game.NewDummyClient("mem")
		ai.SetDifficulty(config.Difficulty)
		ai.RunSetupSequence(game.AIGameSetup{
			Username:    "AI",
			GameID:      gameID,
			FactionName: config.OpponentFaction,
			Units:       config.OpponentSquad(content),
		})
	}()

	info, ok := setup.waitForStart(menu, "Preparing the battlefield...")
	if !ok {
		return client.OutcomeMenu
	}
	return runBattle(window, connection, info, settings, true)
}

var hostServer *server.BattleServer

func hostMatch(window *glfw.Window, menu *client.Menu, settings client.ClientSettings, config client.MatchConfig) client.MatchOutcome {
	if hostServer == nil {
		hostServer = NewBattleServer()
		go hostServer.ListenTCP("0.0.0.0:" + hostPort)
	}
	var conn net.Conn
	var err error
	for attempt := 0; attempt < 20; attempt++ { // the listener may need a moment to come up
		if conn, err = net.Dial("tcp", "127.0.0.1:"+hostPort); err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		menu.ShowStatus("Could not start the server: "+err.Error(), func() bool { return false })
		return client.OutcomeMenu
	}
	connection := game.NewServerConnectionFromConn(conn, nil)
	matchCounter++
	gameID := fmt.Sprintf("%s's game #%d", config.PlayerName, matchCounter)
	setup := startMatchSetup(connection, playerSetup(config, gameID, true), false)
	message := fmt.Sprintf("Waiting for an opponent to join on port %s (%s)...", hostPort, localAddresses())
	info, ok := setup.waitForStart(menu, message)
	if !ok {
		_ = connection.Close()
		return client.OutcomeMenu
	}
	return runBattle(window, connection, info, settings, false)
}

func joinMatch(window *glfw.Window, menu *client.Menu, settings client.ClientSettings, config client.MatchConfig) client.MatchOutcome {
	if config.Connection == nil {
		return client.OutcomeMenu
	}
	setup := startMatchSetup(config.Connection, playerSetup(config, config.GameID, false), true)
	info, ok := setup.waitForStart(menu, "Joining "+config.GameID+"...")
	if !ok {
		_ = config.Connection.Close()
		return client.OutcomeMenu
	}
	return runBattle(window, config.Connection, info, settings, false)
}

// localAddresses lists this machine's LAN addresses, so the host can tell a friend where to connect.
func localAddresses() string {
	addresses, err := net.InterfaceAddrs()
	if err != nil {
		return "unknown address"
	}
	result := ""
	for _, address := range addresses {
		if ipNet, ok := address.(*net.IPNet); ok && !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
			if result != "" {
				result += ", "
			}
			result += ipNet.IP.String()
		}
	}
	if result == "" {
		return "no network"
	}
	return result
}

func runBattle(window *glfw.Window, con *game.ServerConnection, gameInfo game.GameStartedMessage, settings client.ClientSettings, rematchAvailable bool) client.MatchOutcome {
	gameClient := client.NewBattleGame(window, con, gameInfo, settings)
	gameClient.SetRematchAvailable(rematchAvailable)
	gameClient.LoadMapBlocks()

	if gameInfo.MissionDetails.Placement == game.PlacementModeManual {
		for _, unit := range gameInfo.OwnUnits {
			gameClient.AddOwnedUnitToDeploymentQueue(unit)
		}
	} else {
		for _, unit := range gameInfo.OwnUnits {
			gameClient.AddOwnedUnitToGame(unit)
		}
	}

	for _, unit := range gameInfo.VisibleUnits {
		gameClient.AddUnitToGame(unit)
	}

	gameClient.SetLOSAndPressure(gameInfo.LOSMatrix, gameInfo.PressureMatrix)

	gameClient.SwitchToWaitForEvents()

	util.MustSend(con.MapLoaded())

	gameClient.Run()
	_ = con.Close()
	return gameClient.Outcome()
}

// runCommandLineMatch keeps the `bgclient create|join host:port` flow for scripted multiplayer tests.
func runCommandLineMatch(window *glfw.Window, settings client.ClientSettings, content game.Content, createOrJoin, endpoint string) {
	connection := game.NewTCPConnection(endpoint)
	isCreator := createOrJoin == "create"
	factionIndex := 1
	if isCreator {
		factionIndex = 0
	}
	config := quickstartConfig(content, factionIndex)
	setup := startMatchSetup(connection, playerSetup(config, "fx's test game", isCreator), false)
	menu := client.NewMenu(window, &settings, content)
	info, ok := setup.waitForStart(menu, "Waiting for the game to start...")
	if !ok {
		return
	}
	runBattle(window, connection, info, settings, false)
}
