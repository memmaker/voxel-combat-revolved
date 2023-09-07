//go:build client

package main

import (
	"fmt"
	"github.com/faiface/mainthread"
	"github.com/memmaker/battleground/client"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/game"
	"golang.org/x/term"
	"os"
)

func runGame() {
	// new plan..
	// 1. Start the server in a background thread
	// 2. Start a headless client for the AI in a background thread
	// 3. Start the graphical client in the main thread
	if len(os.Args) > 2 {
		runNetworkClient(os.Args[1], os.Args[2])
	} else {
		runStandalone()
	}
}

func runStandalone() {
	battleServer := NewBattleServer()
	go battleServer.ListenTCP("127.0.0.1:9999")

	dummyClient := game.NewDummyClient("127.0.0.1:9999")
	dummyClient.CreateGameSequence()

	mainthread.Call(func() {
		connection := game.NewTCPConnection("127.0.0.1:9999")
		terminalClient(connection, "join")
	})
}

func runNetworkClient(createOrJoin string, endpoint string) {
	mainthread.Call(func() {
		connection := game.NewTCPConnection(endpoint)
		terminalClient(connection, createOrJoin)
	})
}
func startGraphicalClient(con *game.ServerConnection, gameInfo game.GameStartedMessage, settings client.ClientSettings) {
	gameClient := client.NewBattleGame(con, gameInfo, settings)
	gameClient.LoadMap(gameInfo.MapFile)

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
}
func terminalClient(con *game.ServerConnection, argOne string) {
	loginSuccess := false
	createSuccess := false
	joinSuccess := false
	factionSuccess := false
	unitSelectionSuccess := false
	gameStarted := false
	var gameInfo game.GameStartedMessage
	con.SetEventHandler(func(msgReceived game.StringMessage) {
		if msgReceived.MessageType == "LoginResponse" {
			var msg game.ActionResponse
			if util.FromJson(msgReceived.Message, &msg) {
				if msg.Success {
					loginSuccess = true
				}
				println(fmt.Sprintf("[Client] Login response: %s", msg.Message))
			}
		} else if msgReceived.MessageType == "CreateGameResponse" {
			var msg game.ActionResponse
			if util.FromJson(msgReceived.Message, &msg) {
				if msg.Success {
					createSuccess = true
				}
				println(fmt.Sprintf("[Client] Create game response: %s", msg.Message))
			}
		} else if msgReceived.MessageType == "JoinGameResponse" {
			var msg game.ActionResponse
			if util.FromJson(msgReceived.Message, &msg) {
				if msg.Success {
					joinSuccess = true
				}
				println(fmt.Sprintf("[Client] Join game response: %s", msg.Message))
			}
		} else if msgReceived.MessageType == "SelectFactionResponse" {
			var msg game.ActionResponse
			if util.FromJson(msgReceived.Message, &msg) {
				if msg.Success {
					factionSuccess = true
				}
				println(fmt.Sprintf("[Client] Select faction response: %s", msg.Message))
			}
		} else if msgReceived.MessageType == "SelectUnitsResponse" {
			var msg game.ActionResponse
			if util.FromJson(msgReceived.Message, &msg) {
				if msg.Success {
					unitSelectionSuccess = true
				}
				println(fmt.Sprintf("[Client] Select units response: %s", msg.Message))
			}
		} else if msgReceived.MessageType == "GameStarted" {
			util.LogGameInfo("Game started!")
			gameStarted = true
			util.FromJson(msgReceived.Message, &gameInfo)
		} else {
			println(fmt.Sprintf("[Client] Unhandled message type: %s", msgReceived.MessageType))
		}
	})

	createGameSequence := func() {
		util.MustSend(con.Login("creator"))
		util.WaitForTrue(&loginSuccess)
		util.MustSend(con.CreateGame("map", "fx's test game", game.MissionDetails{
			Placement: game.PlacementModeRandom,
		}, true))
		util.WaitForTrue(&createSuccess)
		util.MustSend(con.SelectFaction("X-Com"))
		util.WaitForTrue(&factionSuccess)
		util.MustSend(con.SelectUnits([]game.UnitChoice{
			{
				UnitTypeID: 0,
				Name:       "Jimmy",
				Weapon:     "Mossberg 500",
			},
			{
				UnitTypeID: 0,
				Name:       "Bimmy",
				Weapon:     "Steyr SSG 69",
			},
			{
				UnitTypeID: 0,
				Name:       "Timmy",
				Weapon:     "M16 Rifle",
			},
		}))
		util.WaitForTrue(&unitSelectionSuccess)
	}
	joinGameSequence := func() {
		util.MustSend(con.Login("joiner"))
		util.WaitForTrue(&loginSuccess)
		util.MustSend(con.JoinGame("fx's test game"))
		util.WaitForTrue(&joinSuccess)
		util.MustSend(con.SelectFaction("Deep Ones"))
		util.WaitForTrue(&factionSuccess)
		util.MustSend(con.SelectUnits([]game.UnitChoice{
			{
				UnitTypeID: 2,
				Name:       "Gnarg",
				Weapon:     "M16 Rifle",
			},

			{
				UnitTypeID: 2,
				Name:       "Gorn",
				Weapon:     "Steyr SSG 69",
			},
			{
				UnitTypeID: 2,
				Name:       "Grimbel",
				Weapon:     "Mossberg 500",
			},
		}))
		util.WaitForTrue(&unitSelectionSuccess)
	}

	// get first command line argument as string
	if argOne == "create" {
		createGameSequence()
	} else if argOne == "join" {
		joinGameSequence()
	} else {
		textMenu([]TextItem{
			{
				Text: "Create Game",
				Func: createGameSequence,
			},
			{
				Text: "Join Game",
				Func: joinGameSequence,
			},
		})
	}

	util.LogGameInfo("[Client] Waiting for game to start...")
	util.WaitForTrue(&gameStarted)
	util.LogGameInfo("[Client] Game started!")

	startGraphicalClient(con, gameInfo, client.NewClientSettingsFromFile("settings.json"))
}

type TextItem struct {
	Text string
	Func func()
}

func textMenu(items []TextItem) {
	for i, item := range items {
		println(fmt.Sprintf("%d: %s", i+1, item.Text))
	}
	// switch stdin into 'raw' mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	b := make([]byte, 1)
	_, err = os.Stdin.Read(b)
	if err != nil {
		fmt.Println(err)
		return
	}
	asciiNumbers := b[0] - 48

	if asciiNumbers < 1 || asciiNumbers > byte(len(items)) {
		println("Invalid input")
		textMenu(items)
		return
	}

	items[asciiNumbers-1].Func()
}
