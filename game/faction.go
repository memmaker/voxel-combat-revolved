package game

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/voxel"
)

type UnitDefinition struct {
	Name        string
	TextureFile string
	SpawnPos    voxel.Int3
}

type FactionDefinition struct {
	Name  string
	Color mgl32.Vec3
	Units []UnitDefinition
}

type Faction struct {
	name  string
	units []*Unit
}

func (a *BattleGame) AddFaction(def FactionDefinition) {
	faction := &Faction{name: def.Name}
	a.factions = append(a.factions, faction)

	var units []*Unit
	for _, unitDef := range def.Units {
		spawnedUnit := a.SpawnUnit(unitDef.SpawnPos.ToBlockCenterVec3(), unitDef.TextureFile, unitDef.Name)
		units = append(units, spawnedUnit)
	}
	for _, unit := range units {
		unit.SetFaction(faction)
	}
	faction.units = units
}
