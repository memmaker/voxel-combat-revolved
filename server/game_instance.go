package server

import (
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
	"github.com/memmaker/battleground/game"
	"path"
)

func NewGameInstance(ownerID uint64, gameID string, mapFile string, public bool) *GameInstance {
	mapDir := "./assets/maps"
	mapFile = path.Join(mapDir, mapFile)
	loadedMap := voxel.NewMap(0, 0, 0)
	loadedMap.LoadFromDisk(mapFile)
	println(fmt.Sprintf("[GameInstance] %d created game %s", ownerID, gameID))
	return &GameInstance{
		owner:                 ownerID,
		id:                    gameID,
		mapFile:               mapFile,
		public:                public,
		players:               make([]uint64, 0),
		factionMap:            make(map[*game.UnitInstance]*Faction),
		playerFactions:        make(map[uint64]*Faction),
		currentVisibleEnemies: make(map[*game.UnitInstance][]*game.UnitInstance),
		playerUnits:           make(map[uint64][]*game.UnitInstance),
		playersNeeded:         2,
		voxelMap:              loadedMap,
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
	currentVisibleEnemies map[*game.UnitInstance][]*game.UnitInstance
	voxelMap              *voxel.Map
	players               []uint64
	playerFactions        map[uint64]*Faction
	playerUnits           map[uint64][]*game.UnitInstance
	playersNeeded         int
}

func (g *GameInstance) GetPlayerFactions() map[uint64]string {
	result := make(map[uint64]string)
	for playerID, faction := range g.playerFactions {
		result[playerID] = faction.name
	}
	return result
}

func (g *GameInstance) EndTurn() uint64 {
	println(fmt.Sprintf("[GameInstance] Ending turn for %s", g.currentPlayerFaction().name))
	g.currentPlayerIndex = (g.currentPlayerIndex + 1) % len(g.players)
	g.NextTurn()
	return g.currentPlayerID()
}

func (g *GameInstance) NextTurn() {
	g.UpdateVisibleEnemies()
	println(fmt.Sprintf("[GameInstance] Starting turn for %s", g.currentPlayerFaction().name))

	for _, unit := range g.currentPlayerUnits() {
		if !unit.IsActive() {
			continue
		}
		unit.NextTurn()
	}
	// TODO: Send turn started event to connectedClients
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

func (g *GameInstance) UpdateVisibleEnemies() {
	own := g.currentPlayerFaction()
	visibleEnemies := make(map[*game.UnitInstance][]*game.UnitInstance)
	for _, ownUnit := range g.currentPlayerUnits() {
		for _, unit := range g.units {
			if g.factionMap[unit] == own {
				continue
			}
			if g.CanSee(ownUnit, unit) {
				if _, ok := visibleEnemies[ownUnit]; !ok {
					visibleEnemies[ownUnit] = make([]*game.UnitInstance, 0)
				}
				visibleEnemies[ownUnit] = append(visibleEnemies[ownUnit], unit)
			}
		}
	}
	g.currentVisibleEnemies = visibleEnemies
}

func (g *GameInstance) CanSee(one, another *game.UnitInstance) bool {
	if one == another {
		return true
	}
	source := one.GetEyePosition()
	targetOne := another.GetEyePosition()
	targetTwo := another.GetFootPosition()

	rayOne := g.RayCastUnits(source, targetOne, one, another)
	rayTwo := g.RayCastUnits(source, targetTwo, one, another)

	return rayOne.UnitHit == another || rayTwo.UnitHit == another
}

func (g *GameInstance) OnUnitMoved(unitMapObject voxel.MapObject) {
	unit := unitMapObject.(*game.UnitInstance)
	own := g.currentPlayerFaction()
	if g.factionMap[unit] == own {
		g.currentVisibleEnemies[unit] = make([]*game.UnitInstance, 0)
		for _, enemy := range g.units {
			if g.factionMap[enemy] == own {
				continue
			}
			if g.CanSee(unit, enemy) {
				g.currentVisibleEnemies[unit] = append(g.currentVisibleEnemies[unit], enemy)
			}
		}
	}
}

func (g *GameInstance) RayCastUnits(rayStart, rayEnd mgl32.Vec3, sourceUnit, targetUnit voxel.MapObject) *game.RayCastHit {
	voxelMap := g.voxelMap
	var visitedBlocks []voxel.Int3
	var unitHit *game.UnitInstance
	stopRay := func(x, y, z int32) bool {
		visitedBlocks = append(visitedBlocks, voxel.Int3{X: x, Y: y, Z: z})
		if voxelMap.Contains(x, y, z) {
			block := voxelMap.GetGlobalBlock(x, y, z)
			if block != nil && !block.IsAir() {
				return true
			} else if block.IsOccupied() && (block.GetOccupant() != sourceUnit && block.GetOccupant() == targetUnit) {
				unitHit = block.GetOccupant().(*game.UnitInstance)
				return true
			}
		}
		return false
	}
	hitInfo := util.DDARaycast(rayStart, rayEnd, stopRay)
	if hitInfo.Hit && (voxelMap.ContainsGrid(hitInfo.CollisionGridPosition) || voxelMap.ContainsGrid(hitInfo.PreviousGridPosition)) {
		return &game.RayCastHit{HitInfo3D: hitInfo, VisitedBlocks: visitedBlocks, UnitHit: unitHit}
	}
	return nil
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
	unitInstanceID := uint64(len(g.playerUnits[userID]))
	unit.SetGameUnitID(unitInstanceID)
	println(fmt.Sprintf("[GameInstance] Adding unit %d -> %s of type %d for player %d", unitInstanceID, unit.Name, unit.UnitDefinition.ID, userID))
	g.playerUnits[userID] = append(g.playerUnits[userID], unit)
	g.units = append(g.units, unit)
	return unitInstanceID
}

func (g *GameInstance) GetUnitTypes(userID uint64) []uint64 {
	var result []uint64
	for _, unit := range g.playerUnits[userID] {
		result = append(result, unit.UnitDefinition.ID)
	}
	return result
}

func (g *GameInstance) GetPlayerUnits(userID uint64) []*game.UnitInstance {
	return g.playerUnits[userID]
}

func (g *GameInstance) GetAction(action string) game.Action {
	switch action {
	case "Move":
		return game.NewActionMove(g.voxelMap)
	}
	return nil
}

func (g *GameInstance) IsPlayerTurn(id uint64) bool {
	return g.currentPlayerID() == id
}
