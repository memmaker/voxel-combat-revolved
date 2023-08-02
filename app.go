//go:build client

package main

import (
	"fmt"
	"github.com/faiface/mainthread"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/client"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
	"github.com/memmaker/battleground/game"
	"github.com/memmaker/battleground/server"
	"golang.org/x/term"
	"os"
	"time"
)

func runGame() {
	// new plan..
	// 1. Start the server in a background thread
	// 2. Start a headless client for the AI in a background thread
	// 3. Start the graphical client in the main thread
	battleServer := NewBattleServer()

	dummyClientChannel := util.NewChannelWrapper()
	dummyClient := game.NewDummyClient(game.NewChannelConnection(dummyClientChannel))
	battleServer.AddChannelClient(util.NewReverseChannelWrapper(dummyClientChannel))
	time.Sleep(1 * time.Second) // hacky way to wait for the server to start
	dummyClient.CreateGameSequence()
	go dummyClient.Run()

	mainthread.Call(func() {
		realClientChannel := util.NewChannelWrapper()
		battleServer.AddChannelClient(util.NewReverseChannelWrapper(realClientChannel))
		terminalClient(game.NewChannelConnection(realClientChannel), "join")
	})
}
func startGraphicalClient(con *game.ServerConnection, gameInfo game.GameStartedMessage) {
	width := 800
	height := 600
	gameClient := client.NewBattleGame("BattleGrounds", width, height)
	gameClient.SetConnection(con)
	con.SetEventHandler(gameClient.OnServerMessage)
	gameClient.LoadMap(gameInfo.MapFile)

	for _, unit := range gameInfo.OwnUnits {
		unitDef := unit.UnitDefinition
		gameClient.AddOwnedUnit(unit.SpawnPos.ToBlockCenterVec3(), unit.GameUnitID, unitDef, unit.Name)
	}

	if gameInfo.YourTurn {
		gameClient.Print("It's your turn!")
		gameClient.SwitchToUnit(gameClient.FirstUnit())
	} else {
		gameClient.Print("Waiting for other player...")
		gameClient.SwitchToWaitForEvents()
	}

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
		util.MustSend(con.SelectUnits([]game.UnitChoices{
			{
				UnitTypeID: 0,
				Name:       "Steve",
			},
			{
				UnitTypeID: 1,
				Name:       "Soldier X",
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
		util.MustSend(con.SelectUnits([]game.UnitChoices{
			{
				UnitTypeID: 2,
				Name:       "Deep One",
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

func NewBattleServer() *server.BattleServer {
	defaultCoreStats := game.UnitCoreStats{
		Health: 10,
		Speed:  5,
		OccupiedBlockOffsets: []voxel.Int3{
			{0, 0, 0},
			{0, 1, 0},
		},
	}

	battleServer := server.NewBattleServer()

	battleServer.AddMap("Dev Map", "./assets/maps/map.bin")

	battleServer.AddFaction(server.FactionDefinition{
		Name:  "X-Com",
		Color: mgl32.Vec3{0, 0, 1},
		Units: []game.UnitDefinition{
			{
				ID:        0,
				CoreStats: defaultCoreStats,
				ClientRepresentation: game.UnitClientDefinition{
					TextureFile: "./assets/textures/skins/steve.png",
				},
			},
			{
				ID:        1,
				CoreStats: defaultCoreStats,
				ClientRepresentation: game.UnitClientDefinition{
					TextureFile: "./assets/textures/skins/soldier4.png",
				},
			},
		},
	})
	battleServer.AddFaction(server.FactionDefinition{
		Name:  "Deep Ones",
		Color: mgl32.Vec3{1, 0, 0},
		Units: []game.UnitDefinition{
			{
				ID:        2,
				CoreStats: defaultCoreStats,
				ClientRepresentation: game.UnitClientDefinition{
					TextureFile: "./assets/textures/skins/deep_monster2.png",
				},
			},
		},
	})
	return battleServer
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
