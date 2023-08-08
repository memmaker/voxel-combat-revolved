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
	if len(os.Args) > 1 {
		runNetworkClient(os.Args[1])
	} else {
		runStandalone()
	}
}

func runStandalone() {
	battleServer := NewBattleServer()
	go battleServer.ListenTCP("127.0.0.1:9999")

	dummyConnection := game.NewTCPConnection("127.0.0.1:9999")
	dummyClient := game.NewDummyClient(dummyConnection)
	dummyClient.CreateGameSequence()
	go dummyClient.Run()

	mainthread.Call(func() {
		connection := game.NewTCPConnection("127.0.0.1:9999")
		terminalClient(connection, "join")
	})
}

func runNetworkClient(argOne string) {
	mainthread.Call(func() {
		connection := game.NewTCPConnection("0.0.0.0:9999")
		terminalClient(connection, argOne)
	})
}
func startGraphicalClient(con *game.ServerConnection, gameInfo game.GameStartedMessage) {
	width := 800
	height := 600
	gameClient := client.NewBattleGame("BattleGrounds", width, height)
	gameClient.SetConnection(con)
	gameClient.LoadMap(gameInfo.MapFile)

	for _, unit := range gameInfo.OwnUnits {
		gameClient.AddOwnedUnit(unit, gameInfo.OwnID)
	}
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
	con.SetEventHandler(func(msgType, data string) {
		if msgType == "LoginResponse" {
			var msg game.ActionResponse
			if util.FromJson(data, &msg) {
				if msg.Success {
					loginSuccess = true
				}
				println(fmt.Sprintf("[Client] Login response: %s", msg.Message))
			}
		} else if msgType == "CreateGameResponse" {
			var msg game.ActionResponse
			if util.FromJson(data, &msg) {
				if msg.Success {
					createSuccess = true
				}
				println(fmt.Sprintf("[Client] Create game response: %s", msg.Message))
			}
		} else if msgType == "JoinGameResponse" {
			var msg game.ActionResponse
			if util.FromJson(data, &msg) {
				if msg.Success {
					joinSuccess = true
				}
				println(fmt.Sprintf("[Client] Join game response: %s", msg.Message))
			}
		} else if msgType == "SelectFactionResponse" {
			var msg game.ActionResponse
			if util.FromJson(data, &msg) {
				if msg.Success {
					factionSuccess = true
				}
				println(fmt.Sprintf("[Client] Select faction response: %s", msg.Message))
			}
		} else if msgType == "SelectUnitsResponse" {
			var msg game.ActionResponse
			if util.FromJson(data, &msg) {
				if msg.Success {
					unitSelectionSuccess = true
				}
				println(fmt.Sprintf("[Client] Select units response: %s", msg.Message))
			}
		} else if msgType == "GameStarted" {
			println("Game started!")
			gameStarted = true
			util.FromJson(data, &gameInfo)
		} else {
			println(fmt.Sprintf("[Client] Unhandled message type: %s", msgType))
		}
	})

	createGameSequence := func() {
		util.MustSend(con.Login("creator"))
		util.WaitForTrue(&loginSuccess)
		util.MustSend(con.CreateGame("map.bin", "fx's test game", true))
		util.WaitForTrue(&createSuccess)
		util.MustSend(con.SelectFaction("X-Com"))
		util.WaitForTrue(&factionSuccess)
		util.MustSend(con.SelectUnits([]game.UnitChoice{
			{
				UnitTypeID: 0,
				Name:       "Steve",
				Weapon:     "Mossberg",
			},
			{
				UnitTypeID: 1,
				Name:       "Soldier X",
				Weapon:     "Sniper",
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
				Name:       "Support Guy",
				Weapon:     "Rifle",
			},
			{
				UnitTypeID: 3,
				Name:       "Deep One",
				//Weapon:     "Rifle",
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

	println("[Client] Waiting for game to start...")
	util.WaitForTrue(&gameStarted)
	println("[Client] Game started!")
	startGraphicalClient(con, gameInfo)
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
