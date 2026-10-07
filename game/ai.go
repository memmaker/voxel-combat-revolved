package game

import (
	"fmt"
	"os"
	"sync/atomic"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
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

func (d *DummyClientUnit) UpdateFromServerInstance(instance *UnitInstance) {
	oldModel := d.UnitInstance.GetModel()
	oldVoxelMap := d.GetVoxelMap()

	instance.SetModel(oldModel)
	instance.SetVoxelMap(oldVoxelMap)

	d.UnitInstance = instance
	d.AutoSetStanceAndForwardAndUpdateMap()
}

type DummyClient struct {
	*GameClient[*DummyClientUnit]
	connection *ServerConnection
	brain      *AIBrain
	difficulty AIDifficulty
	isMyTurn   bool
	onGameOver func(msg GameOverMessage)
	passive    bool
	rejoined   bool
	closed     atomic.Bool
}

// send panics on a failed send, unless the connection was closed on purpose.
func (c *DummyClient) send(err error) {
	if !c.closed.Load() {
		util.MustSend(err)
	}
}

// Rejoined reports whether this client took over a seat in a running match.
func (c *DummyClient) Rejoined() bool {
	return c.rejoined
}

// Close drops the connection (used to test reconnecting).
func (c *DummyClient) Close() {
	c.closed.Store(true)
	_ = c.connection.Close()
}

// Join takes this client into an existing game, e.g. to rejoin after a disconnect.
func (c *DummyClient) Join(username, gameID string) {
	c.send(c.connection.Login(username))
	c.send(c.connection.JoinGame(gameID))
}

func NewDummyClient(endpoint string) *DummyClient {
	d := &DummyClient{connection: nil, difficulty: AIDifficultyFromEnv()}
	d.connection = NewTCPConnectionWithHandler(endpoint, d.OnServerMessage)
	return d
}

// AIDifficultyFromEnv reads BT_AI_DIFFICULTY (easy, normal, hard); normal is the default.
func AIDifficultyFromEnv() AIDifficulty {
	switch os.Getenv("BT_AI_DIFFICULTY") {
	case "easy":
		return AIEasy
	case "hard":
		return AIHard
	}
	return AINormal
}

func (c *DummyClient) SetDifficulty(difficulty AIDifficulty) {
	c.difficulty = difficulty
}

func (c *DummyClient) OnServerMessage(incomingMessage StringMessage) {
	msgType, messageAsJson := incomingMessage.MessageType, incomingMessage.Message
	switch msgType {
	case "GameStarted":
		var gameInfo GameStartedMessage
		util.FromJson(messageAsJson, &gameInfo)
		c.GameClient = NewGameClient[*DummyClientUnit](gameInfo, c.createDummyUnit)
		c.GameClient.SetEnvironment("AI-Client")
		println("Game started!")
		loadedMap := c.GetAssets().LoadVoxelMap(gameInfo.MapFile)
		c.GameClient.SetVoxelMap(loadedMap)
		blockList := GetDebugBlockNames()
		indexMap := util.CreateIndexMapFromDirectory("assets/textures/blocks/star_odyssey", blockList)
		bl := NewBlockLibrary(blockList, indexMap)
		bl.ApplyGameplayRules(c.GameInstance)
		c.SetBlockLibrary(bl)

		if gameInfo.MissionDetails.Placement == PlacementModeManual {
			for _, unit := range gameInfo.OwnUnits {
				c.AddOwnedUnitToDeploymentQueue(unit)
			}
		} else {
			for _, unit := range gameInfo.OwnUnits {
				c.AddOwnedUnitToGame(unit)
			}
		}
		for _, unit := range gameInfo.VisibleUnits {
			c.AddUnitToGame(unit)
		}

		c.SetLOSAndPressure(gameInfo.LOSMatrix, gameInfo.PressureMatrix)
		for _, pos := range gameInfo.DestroyedBlocks { // rejoining: replay the terrain changes
			c.GetVoxelMap().SetAir(pos)
		}
		c.rejoined = gameInfo.Rejoin
		c.brain = NewAIBrain(c.GameInstance, c.GetControllingUserID(), int(c.GetSpawnIndex()), c.difficulty)

		c.send(c.connection.MapLoaded())
	case "StartDeployment":
		c.OnDeploy()
	case "OwnUnitMoved":
		var msg VisualOwnUnitMoved
		if util.FromJson(messageAsJson, &msg) {
			c.OnOwnUnitMoved(msg)
			c.continueTurn()
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
				c.continueTurn()
			}
		}
	case "Throw":
		var msg VisualThrow
		if util.FromJson(messageAsJson, &msg) {
			c.OnThrow(msg)
			if c.IsMyUnit(msg.Attacker) {
				c.continueTurn()
			}
		}
	case "BeginOverwatch":
		var msg VisualBeginOverwatch
		if util.FromJson(messageAsJson, &msg) {
			if c.IsMyUnit(msg.Watcher) {
				c.OnBeginOverwatch(msg)
				c.continueTurn()
			}
		}
	case "Heal":
		var msg VisualHeal
		if util.FromJson(messageAsJson, &msg) {
			if !c.IsMyUnit(msg.Healer) {
				c.ApplyHeal(msg)
			} else {
				c.ApplyHeal(msg)
				c.continueTurn()
			}
		}
	case "Melee":
		var msg VisualMelee
		if util.FromJson(messageAsJson, &msg) {
			c.ApplyMelee(msg)
			if c.IsMyUnit(msg.Attacker) {
				c.continueTurn()
			}
		}
	case "Reaction":
		var msg VisualReaction
		if util.FromJson(messageAsJson, &msg) {
			c.OnReaction(msg)
			c.continueTurn()
		}
	case "Reload":
		var msg UnitMessage
		if util.FromJson(messageAsJson, &msg) {
			if unit, known := c.GetUnit(msg.UnitID()); known {
				unit.Reload()
			}
			c.continueTurn()
		}
	case "PlayerConnection":
		// the opponent dropped out or came back; nothing to do for the AI
	case "ActionResponse":
		var msg ActionResponse
		if util.FromJson(messageAsJson, &msg) {
			c.OnTargetedUnitActionResponse(msg)
		}
	case "NextPlayer":
		var msg NextPlayerMessage
		if util.FromJson(messageAsJson, &msg) {
			c.OnNextPlayer(msg)
			c.isMyTurn = msg.YourTurn
			if msg.YourTurn {
				c.brain.StartTurn()
				c.continueTurn()
			}
		}
	case "GameOver":
		var msg GameOverMessage
		if util.FromJson(messageAsJson, &msg) {
			c.isMyTurn = false
			c.OnGameOver(msg)
			if c.onGameOver != nil {
				c.onGameOver(msg)
			}
		}
	}
}

// SetPassive makes the client never act or end its turn (used to test the turn timer).
func (c *DummyClient) SetPassive(passive bool) {
	c.passive = passive
}

func (c *DummyClient) continueTurn() {
	if c.isMyTurn && !c.passive {
		c.brain.Step(c)
	}
}

func (c *DummyClient) OnTargetedUnitActionResponse(msg ActionResponse) {
	if !msg.Success {
		println(fmt.Sprintf("[DummyClient] Action failed: %s", msg.Message))
		if c.brain != nil {
			c.brain.OnActionFailed()
		}
		c.continueTurn()
	}
}

// AICommander implementation

func (c *DummyClient) Move(unitID uint64, target voxel.Int3) {
	c.send(c.connection.TargetedUnitAction(unitID, "Move", []voxel.Int3{target}))
}

func (c *DummyClient) Shoot(unitID uint64, target voxel.Int3) {
	c.send(c.connection.TargetedUnitAction(unitID, "Shot", []voxel.Int3{target}))
}

func (c *DummyClient) Throw(unitID uint64, itemName string, target mgl32.Vec3) {
	c.send(c.connection.ThrownUnitAction(unitID, "Throw", itemName, []mgl32.Vec3{target}))
}

func (c *DummyClient) Overwatch(unitID uint64, targets []voxel.Int3) {
	c.send(c.connection.TargetedUnitAction(unitID, "Overwatch", targets))
}

func (c *DummyClient) Heal(unitID uint64, target voxel.Int3) {
	c.send(c.connection.TargetedUnitAction(unitID, "Heal", []voxel.Int3{target}))
}

func (c *DummyClient) Melee(unitID uint64, target voxel.Int3) {
	c.send(c.connection.TargetedUnitAction(unitID, "Melee", []voxel.Int3{target}))
}

func (c *DummyClient) React(unitID uint64, reaction Reaction) {
	c.send(c.connection.TargetedUnitAction(unitID, string(reaction), nil))
}

func (c *DummyClient) Reload(unitID uint64) {
	c.send(c.connection.ReloadAction(unitID))
}

func (c *DummyClient) EndTurn() {
	c.isMyTurn = false
	c.send(c.connection.EndTurn())
}

// AIGameSetup describes how an AI client enters a game: it either creates it or joins an existing one.
type AIGameSetup struct {
	Username    string
	GameID      string
	CreateGame  bool
	MapFile     string
	Mission     *MissionDetails
	FactionName string
	Units       []UnitChoice
}

// DefaultAISetup is the opponent used by the standalone singleplayer mode.
func DefaultAISetup() AIGameSetup {
	return AIGameSetup{
		Username:    "creator",
		GameID:      "fx's test game",
		CreateGame:  true,
		MapFile:     "map",
		Mission:     NewRandomDeathmatch(),
		FactionName: "X-Com",
		Units: []UnitChoice{
			{UnitTypeID: 4, Name: "Jimmy", Weapon: "Mossberg 500", Items: []string{"Smoke Grenade"}},
			{UnitTypeID: 5, Name: "Bimmy", Weapon: "Steyr SSG 69", Items: []string{"Frag Grenade"}},
			{UnitTypeID: 7, Name: "Timmy", Weapon: "M16 Rifle", Items: []string{"Frag Grenade"}},
		},
	}
}

func (c *DummyClient) CreateGameSequence() {
	c.RunSetupSequence(DefaultAISetup())
}

// SetOnGameOver registers a callback for the end of the match (used by automated matches).
func (c *DummyClient) SetOnGameOver(onGameOver func(msg GameOverMessage)) {
	c.onGameOver = onGameOver
}

// RunSetupSequence logs in, creates or joins the game, picks faction and units and then
// hands control to the AI once the game starts. It blocks until the units are selected.
func (c *DummyClient) RunSetupSequence(setup AIGameSetup) {
	con := c.connection
	responses := make(chan ActionResponse, 4)
	con.SetEventHandler(func(msgReceived StringMessage) {
		switch msgReceived.MessageType {
		case "LoginResponse", "CreateGameResponse", "JoinGameResponse", "SelectFactionResponse", "SelectUnitsResponse":
			var msg ActionResponse
			if util.FromJson(msgReceived.Message, &msg) {
				println(fmt.Sprintf("[DummyClient] %s: %s", msgReceived.MessageType, msg.Message))
				responses <- msg
			}
		default:
			// the game may start as soon as the last player picked a faction
			c.OnServerMessage(msgReceived)
		}
	})
	expectSuccess := func(step string, err error) {
		util.MustSend(err)
		if response := <-responses; !response.Success {
			panic(fmt.Sprintf("[DummyClient] %s failed: %s", step, response.Message))
		}
	}
	println("[DummyClient] Starting setup sequence...")
	expectSuccess("login", con.Login(setup.Username))
	if setup.CreateGame {
		expectSuccess("create game", con.CreateGame(setup.MapFile, setup.GameID, setup.Mission, true))
	} else {
		expectSuccess("join game", con.JoinGame(setup.GameID))
	}
	expectSuccess("select faction", con.SelectFaction(setup.FactionName))
	expectSuccess("select units", con.SelectUnits(setup.Units))
	println("[DummyClient] Waiting for game to start...")
	c.connection.SetEventHandler(c.OnServerMessage)
}

func (c *DummyClient) createDummyUnit(instance *UnitInstance) *DummyClientUnit {
	return NewDummyClientUnit(instance)
}

func (c *DummyClient) OnDeploy() {
	deployment := make(map[uint64]voxel.Int3)
	spawnIndex := c.GetSpawnIndex()
	spawns := c.mapMeta.SpawnPositions[spawnIndex]
	for _, unit := range c.GetDeploymentQueue() {
		targetPos := c.choseRandom(unit, spawns)
		unit.SetBlockPositionAndUpdateStance(targetPos)
		deployment[unit.UnitID()] = targetPos
	}

	c.send(c.connection.SelectDeployment(deployment))
}

func (c *DummyClient) choseRandom(unit *DummyClientUnit, spawns []voxel.Int3) voxel.Int3 {
	current := util.RandomChoice(spawns)
	placeable, _ := c.voxelMap.IsUnitPlaceable(unit, current)
	for !placeable {
		current = util.RandomChoice(spawns)
		placeable, _ = c.voxelMap.IsUnitPlaceable(unit, current)
	}
	return current
}
