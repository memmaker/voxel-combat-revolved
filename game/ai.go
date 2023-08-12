package game

import (
	"fmt"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
	"math/rand"
)

type DummyClient struct {
	connection     *ServerConnection
	ownUnits       []*UnitInstance
	voxelMap       *voxel.Map
	movedUnits     map[uint64]bool
	waitingForUnit uint64
}

func NewDummyClient(endpoint string) *DummyClient {
	d := &DummyClient{connection: nil}
	d.connection = NewTCPConnectionWithHandler(endpoint, d.OnServerMessage)
	return d
}

func (c *DummyClient) OnServerMessage(msg StringMessage) {
	switch msg.MessageType {
	case "NextPlayer":
		// determine if it's our turn
		var turnInfo NextPlayerMessage
		util.FromJson(msg.Message, &turnInfo)
		if turnInfo.YourTurn {
			c.resetTurn()
			c.makeMove()
		}
	case "GameStarted":
		var gameInfo GameStartedMessage
		util.FromJson(msg.Message, &gameInfo)
		c.ownUnits = gameInfo.OwnUnits
		println("Game started!")
		loadedMap := voxel.NewMapFromFile(gameInfo.MapFile)
		c.voxelMap = loadedMap
		util.MustSend(c.connection.MapLoaded())
	case "TargetedUnitActionResponse":
		var actionResponse ActionResponse
		if util.FromJson(msg.Message, &actionResponse) {
			if !actionResponse.Success {
				println(fmt.Sprintf("[DummyClient] Action failed: %s", actionResponse.Message))
				c.movedUnits[c.waitingForUnit] = true
			}
		}
		c.makeMove()
	case "OwnUnitMoved":
		var unitMoved VisualOwnUnitMoved
		if util.FromJson(msg.Message, &unitMoved) {
			println(fmt.Sprintf("[DummyClient] Unit %d moved to %s", unitMoved.UnitID, unitMoved.EndPosition.ToString()))
			c.movedUnits[unitMoved.UnitID] = true
		}
		c.makeMove()
	}
}

func (c *DummyClient) makeMove() {
	moveAction := NewActionMove(c.voxelMap)
	unit, unitLeft := c.getNextUnit()

	if !unitLeft {
		// JUST END TURN FOR NOW
		println(fmt.Sprintf("[DummyClient] Ending turn..."))
		util.MustSend(c.connection.EndTurn())
		return
	}

	validMoves := moveAction.GetValidTargets(unit)
	if len(validMoves) > 0 {
		chosenDest := choseRandom(validMoves)
		println(fmt.Sprintf("[DummyClient] Moving unit %s(%d) to %s", unit.Name, unit.UnitID(), chosenDest.ToString()))
		util.MustSend(c.connection.TargetedUnitAction(unit.UnitID(), moveAction.GetName(), chosenDest))
		c.waitingForUnit = unit.UnitID()
	} else {
		println(fmt.Sprintf("[DummyClient] No valid moves for unit %s(%d)", unit.Name, unit.UnitID()))
		c.movedUnits[unit.UnitID()] = true
	}
}

func choseRandom(moves []voxel.Int3) voxel.Int3 {
	randIndex := rand.Intn(len(moves))
	return moves[randIndex]
}

func (c *DummyClient) CreateGameSequence() {
	con := c.connection
	loginSuccess := false
	createSuccess := false
	factionSuccess := false
	unitSelectionSuccess := false
	con.SetEventHandler(func(msgReceived StringMessage) {
		if msgReceived.MessageType == "LoginResponse" {
			var msg ActionResponse
			if util.FromJson(msgReceived.Message, &msg) {
				if msg.Success {
					loginSuccess = true
				}
				println(fmt.Sprintf("[DummyClient] Login response: %s", msg.Message))
			}
		} else if msgReceived.MessageType == "CreateGameResponse" {
			var msg ActionResponse
			if util.FromJson(msgReceived.Message, &msg) {
				if msg.Success {
					createSuccess = true
				}
				println(fmt.Sprintf("[DummyClient] Create game response: %s", msg.Message))
			}
		} else if msgReceived.MessageType == "SelectFactionResponse" {
			var msg ActionResponse
			if util.FromJson(msgReceived.Message, &msg) {
				if msg.Success {
					factionSuccess = true
				}
				println(fmt.Sprintf("[DummyClient] Select faction response: %s", msg.Message))
			}
		} else if msgReceived.MessageType == "SelectUnitsResponse" {
			var msg ActionResponse
			if util.FromJson(msgReceived.Message, &msg) {
				if msg.Success {
					unitSelectionSuccess = true
				}
				println(fmt.Sprintf("[DummyClient] Select units response: %s", msg.Message))
			}
		} else {
			println(fmt.Sprintf("[DummyClient] Unhandled message type: %s", msgReceived.MessageType))
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
	c.connection.SetEventHandler(c.OnServerMessage)
}

func (c *DummyClient) resetTurn() {
	c.movedUnits = make(map[uint64]bool)
}

func (c *DummyClient) getNextUnit() (*UnitInstance, bool) {
	for _, unit := range c.ownUnits {
		if _, ok := c.movedUnits[unit.UnitID()]; !ok {
			return unit, true
		}
	}
	return nil, false
}
