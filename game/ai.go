package game

import "C"
import (
	"fmt"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
	"math/rand"
)

type DummyClientUnit struct {
	*UnitInstance
	isUserControlled bool
}

func (d *DummyClientUnit) IsUserControlled() bool {
	return d.isUserControlled
}

func NewDummyClientUnit(unit *UnitInstance) *DummyClientUnit {
	return &DummyClientUnit{UnitInstance: unit}
}
func (d *DummyClientUnit) SetUserControlled() {
	d.isUserControlled = true
}

func (d *DummyClientUnit) SetServerInstance(instance *UnitInstance) {
	oldModel := d.UnitInstance.GetModel()
	oldVoxelMap := d.GetVoxelMap()

	instance.SetModel(oldModel)
	instance.SetVoxelMap(oldVoxelMap)

	d.UnitInstance = instance
	d.UpdateMapAndModelAndAnimation()
}

type DummyClient struct {
	*GameClient[*DummyClientUnit]
	connection     *ServerConnection
	movedUnits     map[uint64]bool
	waitingForUnit uint64
	turnCounter    int
}

func NewDummyClient(endpoint string) *DummyClient {
	d := &DummyClient{connection: nil, movedUnits: make(map[uint64]bool), turnCounter: 0}
	d.connection = NewTCPConnectionWithHandler(endpoint, d.OnServerMessage)
	return d
}

func (c *DummyClient) OnServerMessage(incomingMessage StringMessage) {
	msgType, messageAsJson := incomingMessage.MessageType, incomingMessage.Message
	switch msgType {
	case "GameStarted":
		var gameInfo GameStartedMessage
		util.FromJson(messageAsJson, &gameInfo)
		c.GameClient = NewGameClient[*DummyClientUnit](gameInfo.OwnID, gameInfo.GameID, c.createDummyUnit)
		println("Game started!")
		loadedMap := voxel.NewMapFromFile(gameInfo.MapFile)
		c.GameClient.SetVoxelMap(loadedMap)

		for _, unit := range gameInfo.OwnUnits {
			c.AddOwnedUnit(unit)
		}
		for _, unit := range gameInfo.VisibleUnits {
			c.AddUnit(unit)
		}

		c.SetLOSMatrix(gameInfo.LOSMatrix)

		util.MustSend(c.connection.MapLoaded())
	case "OwnUnitMoved":
		var msg VisualOwnUnitMoved
		if util.FromJson(messageAsJson, &msg) {
			c.OnOwnUnitMoved(msg)
			println(fmt.Sprintf("[DummyClient] Unit %d moved to %s", msg.UnitID, msg.EndPosition.ToString()))
			c.movedUnits[msg.UnitID] = true
			c.makeMove()
		}
	case "EnemyUnitMoved":
		var msg VisualEnemyUnitMoved
		if util.FromJson(messageAsJson, &msg) {
			c.OnEnemyUnitMoved(msg)
		}
	case "RangedAttack":
		var msg VisualRangedAttack
		if util.FromJson(messageAsJson, &msg) {
			c.OnRangedAttack(msg)
			if c.IsMyUnit(msg.Attacker) {
				println(fmt.Sprintf("[DummyClient] Unit %d shot", msg.Attacker))
				c.movedUnits[msg.Attacker] = true
				c.makeMove()
			}
		}
	case "ActionResponse":
		var msg ActionResponse
		if util.FromJson(messageAsJson, &msg) {
			c.OnTargetedUnitActionResponse(msg)
		}
	case "NextPlayer":
		var msg NextPlayerMessage
		if util.FromJson(messageAsJson, &msg) {
			c.OnNextPlayer(msg)
			if msg.YourTurn {
				c.resetTurn()
				c.makeMove()
				c.turnCounter++
			}
		}
	case "GameOver":
		var msg GameOverMessage
		if util.FromJson(messageAsJson, &msg) {
			c.OnGameOver(msg)
		}
	}

}
func (c *DummyClient) OnTargetedUnitActionResponse(msg ActionResponse) {
	if !msg.Success {
		println(fmt.Sprintf("[DummyClient] Action failed: %s", msg.Message))
		c.movedUnits[c.waitingForUnit] = true
	}
	c.makeMove()
}

func (c *DummyClient) makeMove() {
	unit, unitLeft := c.getNextUnit()

	if !unitLeft {
		// JUST END TURN FOR NOW
		println(fmt.Sprintf("[DummyClient] Ending turn..."))
		util.MustSend(c.connection.EndTurn())
		return
	}
	enemy, available := c.GetNearestEnemy(unit)
	if available {
		c.attackUnit(unit, enemy)
	} else {
		c.moveUnit(unit)
	}
}

func (c *DummyClient) attackUnit(attacker, target *DummyClientUnit) {
	shotAction := NewActionShot(c.GameInstance)
	util.MustSend(c.connection.TargetedUnitAction(attacker.UnitID(), shotAction.GetName(), target.GetBlockPosition()))
	c.waitingForUnit = attacker.UnitID()
}

func (c *DummyClient) moveUnit(unit *DummyClientUnit) bool {
	moveAction := NewActionMove(c.voxelMap)
	validMoves := moveAction.GetValidTargets(unit)
	if len(validMoves) > 0 {
		chosenDest := choseRandom(validMoves)
		moves := int32(4)
		if c.turnCounter%2 == 1 {
			moves *= -1
		}
		chosenDest = unit.GetBlockPosition().Add(voxel.Int3{Z: moves})
		println(fmt.Sprintf("[DummyClient] Moving unit %s(%d) to %s", unit.Name, unit.UnitID(), chosenDest.ToString()))
		util.MustSend(c.connection.TargetedUnitAction(unit.UnitID(), moveAction.GetName(), chosenDest))
		// HACK: assume this works
		unit.SetBlockPositionAndUpdateMapAndModelAndAnimations(chosenDest)
		c.waitingForUnit = unit.UnitID()
		return true
	} else {
		return false
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
			Name:       "Jimmy",
			Weapon:     "Mossberg 500",
		},
		/*
			{
				UnitTypeID: 0,
				name:       "Bimmy",
				Weapon:     "Steyr SSG 69",
			},
			{
				UnitTypeID: 0,
				name:       "Timmy",
				Weapon:     "M16 Rifle",
			},
		*/
		/*
			{
				UnitTypeID: 1,
				name:       "Walker",
				//Weapon:     "Sniper",
			},

		*/
	}))
	util.WaitForTrue(&unitSelectionSuccess)

	println("[DummyClient] Waiting for game to start...")
	c.connection.SetEventHandler(c.OnServerMessage)
}

func (c *DummyClient) resetTurn() {
	for _, unit := range c.GetMyUnits() {
		unit.NextTurn()
	}
	c.movedUnits = make(map[uint64]bool)
}

func (c *DummyClient) getNextUnit() (*DummyClientUnit, bool) {
	for _, unit := range c.GetMyUnits() {
		if _, ok := c.movedUnits[unit.UnitID()]; !ok {
			clientUnit, available := c.GetClientUnit(unit.UnitID())
			if !available {
				continue
			}
			if !clientUnit.IsActive() {
				continue
			}
			return clientUnit, available
		}
	}
	return nil, false
}

func (c *DummyClient) createDummyUnit(instance *UnitInstance) *DummyClientUnit {
	return NewDummyClientUnit(instance)
}
