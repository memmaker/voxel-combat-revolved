package server

import (
	"fmt"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
	"github.com/memmaker/battleground/game"
	"path"
)

type ServerAction interface {
	IsValid() (bool, string)
	Execute(mb *game.MessageBuffer)
	IsTurnEnding() bool
}

func NewGameInstance(ownerID uint64, gameID string, mapFile string, public bool) *GameInstance {
	mapDir := "./assets/maps"
	mapFile = path.Join(mapDir, mapFile)
	loadedMap := voxel.NewMapFromFile(mapFile)
	println(fmt.Sprintf("[GameInstance] %d created game %s", ownerID, gameID))
	return &GameInstance{
		owner:                 ownerID,
		id:                    gameID,
		mapFile:               mapFile,
		public:                public,
		players:               make([]uint64, 0),
		factionMap:            make(map[*game.UnitInstance]*Faction),
		playerFactions:        make(map[uint64]*Faction),
		currentVisibleEnemies: make(map[uint64]map[uint64]bool),
		playerUnits:           make(map[uint64][]*game.UnitInstance),
		playersNeeded:         2,
		voxelMap:              loadedMap,
		cameras:               make(map[uint64]*util.FPSCamera),
	}
}

type GameInstance struct {
	id      string
	owner   uint64
	mapFile string
	public  bool

	// game instance state
	currentPlayerIndex    int
	units                 []*game.UnitInstance
	factionMap            map[*game.UnitInstance]*Faction
	currentVisibleEnemies map[uint64]map[uint64]bool
	voxelMap              *voxel.Map
	players               []uint64
	playerFactions        map[uint64]*Faction
	playerUnits           map[uint64][]*game.UnitInstance
	playersNeeded         int
	cameras               map[uint64]*util.FPSCamera
}

func (g *GameInstance) GetPlayerFactions() map[uint64]string {
	result := make(map[uint64]string)
	for playerID, faction := range g.playerFactions {
		result[playerID] = faction.name
	}
	return result
}

func (g *GameInstance) NextPlayer() uint64 {
	println(fmt.Sprintf("[GameInstance] Ending turn for %s", g.currentPlayerFaction().name))
	g.currentPlayerIndex = (g.currentPlayerIndex + 1) % len(g.players)
	println(fmt.Sprintf("[GameInstance] Starting turn for %s", g.currentPlayerFaction().name))

	for _, unit := range g.currentPlayerUnits() {
		if !unit.IsActive() {
			continue
		}
		unit.NextTurn()
	}
	return g.currentPlayerID()
}

func (g *GameInstance) currentPlayerUnits() []*game.UnitInstance {
	return g.playerUnits[g.currentPlayerID()]
}

func (g *GameInstance) currentPlayerFaction() *Faction {
	return g.playerFactions[g.currentPlayerID()]
}

func (g *GameInstance) currentPlayerID() uint64 {
	return g.players[g.currentPlayerIndex]
}

func (g *GameInstance) OnUnitMoved(unitMapObject voxel.MapObject) {
	unit := unitMapObject.(*game.UnitInstance)
	own := g.currentPlayerFaction()
	if g.factionMap[unit] == own {
		if _, notExists := g.currentVisibleEnemies[unit.UnitID()]; notExists {
			g.currentVisibleEnemies[unit.UnitID()] = make(map[uint64]bool)
		}
		for _, enemy := range g.units {
			if g.factionMap[enemy] == own {
				continue
			}
			if g.CanSee(unit, enemy) {
				g.currentVisibleEnemies[unit.UnitID()][enemy.UnitID()] = true
			} else {
				g.currentVisibleEnemies[unit.UnitID()][enemy.UnitID()] = false
			}
		}
	}
}

func (g *GameInstance) AddPlayer(id uint64) {
	println(fmt.Sprintf("[GameInstance] Adding player %d to game %s", id, g.id))
	g.players = append(g.players, id)
}

func (g *GameInstance) IsReady() bool {
	return len(g.players) == g.playersNeeded && len(g.playerFactions) == g.playersNeeded && len(g.playerUnits) == g.playersNeeded
}

func (g *GameInstance) Start() uint64 {
	firstPlayer := g.players[0]
	return firstPlayer
}

func (g *GameInstance) SetFaction(userID uint64, faction *Faction) {
	g.playerFactions[userID] = faction
	println(fmt.Sprintf("[GameInstance] Player %d is now in faction %s", userID, faction.name))
}

func (g *GameInstance) AddUnit(userID uint64, unit *game.UnitInstance) uint64 {
	if _, unitsExist := g.playerUnits[userID]; !unitsExist {
		g.playerUnits[userID] = make([]*game.UnitInstance, 0)
	}
	unitInstanceID := uint64(len(g.units))
	unit.SetGameUnitID(unitInstanceID)
	println(fmt.Sprintf("[GameInstance] Adding unit %d -> %s of type %d for player %d", unitInstanceID, unit.Name, unit.Definition.ID, userID))
	g.playerUnits[userID] = append(g.playerUnits[userID], unit)
	g.units = append(g.units, unit)
	g.factionMap[unit] = g.playerFactions[userID]
	g.voxelMap.SetUnit(unit, unit.GetBlockPosition())
	return unitInstanceID
}

func (g *GameInstance) GetUnitTypes(userID uint64) []uint64 {
	var result []uint64
	for _, unit := range g.playerUnits[userID] {
		result = append(result, unit.Definition.ID)
	}
	return result
}

func (g *GameInstance) GetPlayerUnits(userID uint64) []*game.UnitInstance {
	return g.playerUnits[userID]
}

func (g *GameInstance) GetServerActionForUnit(actionMessage game.UnitActionMessage, unit *game.UnitInstance) ServerAction {
	switch typedMsg := actionMessage.(type) {
	case game.TargetedUnitActionMessage:
		return g.GetTargetedAction(typedMsg, unit)
	case game.FreeAimActionMessage:
		return g.GetFreeAimAction(typedMsg, unit)
	}
	return nil
}

func (g *GameInstance) GetTargetedAction(targetAction game.TargetedUnitActionMessage, unit *game.UnitInstance) ServerAction {
	switch targetAction.Action {
	case "Move":
		return NewServerActionMove(g, game.NewActionMove(g.voxelMap), unit, targetAction.Target)
	case "Shot":
		return NewServerActionSnapShot(g, unit, targetAction.Target)
	}
	println(fmt.Sprintf("[GameInstance] ERR -> Unknown action %s", targetAction.Action))
	return nil
}

func (g *GameInstance) GetFreeAimAction(msg game.FreeAimActionMessage, unit *game.UnitInstance) ServerAction {
	switch msg.Action {
	case "Shot":
		camera := g.cameras[unit.ControlledBy()]
		camera.Reposition(msg.CamPos, msg.CamRotX, msg.CamRotY)
		return NewServerActionFreeShot(g, unit, camera)
	}
	println(fmt.Sprintf("[GameInstance] ERR -> Unknown action %s", msg.Action))
	return nil
}

func (g *GameInstance) IsPlayerTurn(id uint64) bool {
	return g.currentPlayerID() == id
}

func (g *GameInstance) SetLOS(observer uint64, target uint64, canSee bool) {
	if _, ok := g.currentVisibleEnemies[observer]; !ok {
		g.currentVisibleEnemies[observer] = make(map[uint64]bool)
	}
	g.currentVisibleEnemies[observer][target] = canSee
}

func (g *GameInstance) IsGameOver() (bool, uint64) {
	playersWithActiveUnits := make(map[uint64]bool)
	for playerID, units := range g.playerUnits {
		for _, unit := range units {
			if unit.IsActive() {
				playersWithActiveUnits[playerID] = true
				break
			}
		}
	}
	if len(playersWithActiveUnits) == 1 {
		for playerID := range playersWithActiveUnits {
			return true, playerID
		}
	}
	return false, 0
}

func (g *GameInstance) Kill(killer, victim *game.UnitInstance) {
	println(fmt.Sprintf("[GameInstance] %s(%d) killed %s(%d)", killer.Name, killer.UnitID(), victim.Name, victim.UnitID()))
	victim.Kill()
}

func (g *GameInstance) SetCamera(userID uint64, camera *util.FPSCamera) {
	g.cameras[userID] = camera
}
