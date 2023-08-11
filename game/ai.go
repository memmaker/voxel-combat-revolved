package game

import (
	"fmt"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
	"math/rand"
	"time"
)

type DummyClient struct {
	connection           *ServerConnection
	ownUnits             []*UnitInstance
	voxelMap             *voxel.Map
	movementAcknowledged bool
}

func NewDummyClient(con *ServerConnection) *DummyClient {
	d := &DummyClient{connection: con}
	return d
}

func (c *DummyClient) OnServerMessage(msgType string, data string) {
	switch msgType {
	case "NextPlayer":
		// determine if it's our turn
		var turnInfo NextPlayerMessage
		util.FromJson(data, &turnInfo)
		if turnInfo.YourTurn {
			c.makeMove()
		}
	case "GameStarted":
		var gameInfo GameStartedMessage
		util.FromJson(data, &gameInfo)
		c.ownUnits = gameInfo.OwnUnits
		println("Game started!")
		loadedMap := voxel.NewMapFromFile(gameInfo.MapFile)
		c.voxelMap = loadedMap
		util.MustSend(c.connection.MapLoaded())
	case "TargetedUnitActionResponse":
		fallthrough
	case "OwnUnitMoved":
		c.movementAcknowledged = true
	}
}

func (c *DummyClient) makeMove() {
	moveAction := NewActionMove(c.voxelMap)
	for _, unit := range c.ownUnits {
		validMoves := moveAction.GetValidTargets(unit)
		if len(validMoves) > 0 {
			chosenDest := choseRandom(validMoves)
			println(fmt.Sprintf("[DummyClient] Moving unit %s(%d) to %s", unit.Name, unit.UnitID(), chosenDest.ToString()))
			util.MustSend(c.connection.TargetedUnitAction(unit.UnitID(), moveAction.GetName(), chosenDest))
		}
		for !c.movementAcknowledged {
			time.Sleep(100 * time.Millisecond)
			c.movementAcknowledged = false
		}
	}
	// JUST END TURN FOR NOW
	println(fmt.Sprintf("[DummyClient] Ending turn..."))
	util.MustSend(c.connection.EndTurn())
}

func choseRandom(moves []voxel.Int3) voxel.Int3 {
	randIndex := rand.Intn(len(moves))
	return moves[randIndex]
}

func (c *DummyClient) Run() {
	c.connection.SetEventHandler(c.OnServerMessage)
}

func (c *DummyClient) CreateGameSequence() {
	con := c.connection
	loginSuccess := false
	createSuccess := false
	factionSuccess := false
	unitSelectionSuccess := false
	con.SetEventHandler(func(msgType, data string) {
		if msgType == "LoginResponse" {
			var msg ActionResponse
			if util.FromJson(data, &msg) {
				if msg.Success {
					loginSuccess = true
				}
				println(fmt.Sprintf("[DummyClient] Login response: %s", msg.Message))
			}
		} else if msgType == "CreateGameResponse" {
			var msg ActionResponse
			if util.FromJson(data, &msg) {
				if msg.Success {
					createSuccess = true
				}
				println(fmt.Sprintf("[DummyClient] Create game response: %s", msg.Message))
			}
		} else if msgType == "SelectFactionResponse" {
			var msg ActionResponse
			if util.FromJson(data, &msg) {
				if msg.Success {
					factionSuccess = true
				}
				println(fmt.Sprintf("[DummyClient] Select faction response: %s", msg.Message))
			}
		} else if msgType == "SelectUnitsResponse" {
			var msg ActionResponse
			if util.FromJson(data, &msg) {
				if msg.Success {
					unitSelectionSuccess = true
				}
				println(fmt.Sprintf("[DummyClient] Select units response: %s", msg.Message))
			}
		} else {
			println(fmt.Sprintf("[DummyClient] Unhandled message type: %s", msgType))
		}
	})
	println("[DummyClient] Starting create game sequence...")
	util.MustSend(con.Login("creator"))
	util.WaitForTrue(&loginSuccess)
	util.MustSend(con.CreateGame("map.bin", "fx's test game", true))
	util.WaitForTrue(&createSuccess)
	util.MustSend(con.SelectFaction("X-Com"))
	util.WaitForTrue(&factionSuccess)
	util.MustSend(con.SelectUnits([]UnitChoice{
		{
			UnitTypeID: 0,
			Name:       "Steve",
			Weapon:     "Sniper",
		},
		{
			UnitTypeID: 1,
			Name:       "Walker",
			//Weapon:     "Sniper",
		},
	}))
	util.WaitForTrue(&unitSelectionSuccess)

	println("[DummyClient] Waiting for game to start...")
}
