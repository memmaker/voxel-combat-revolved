package game

import (
	"fmt"
	"github.com/memmaker/battleground/engine/util"
)

type DummyClient struct {
	connection *ServerConnection
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
		// determine if it's our turn
		println("Game started!")
		util.MustSend(c.connection.MapLoaded())
	}
}

func (c *DummyClient) makeMove() {
	// JUST END TURN FOR NOW
	println(fmt.Sprintf("[DummyClient] Ending turn..."))
	util.MustSend(c.connection.EndTurn())
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
	util.MustSend(con.SelectUnits([]UnitChoices{
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

	println("[DummyClient] Waiting for game to start...")
}
