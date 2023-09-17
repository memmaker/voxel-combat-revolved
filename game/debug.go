package game

import (
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/voxel"
)

type CompleteUnitState struct {
	ID            uint64
	Name          string
	Forward       mgl32.Vec3
	Position      mgl32.Vec3
	Stance        Stance
	AP            float64
	CanMove       bool
	OccupiedOnMap []voxel.Int3
}

func (s CompleteUnitState) ToString() string {
	stateString := "\n"
	stateString += fmt.Sprintf("ID: %v\n", s.ID)
	stateString += fmt.Sprintf("Name: %v\n", s.Name)
	stateString += fmt.Sprintf("AimDirection: %v\n", s.Forward)
	stateString += fmt.Sprintf("Origin: %v\n", s.Position)
	stateString += fmt.Sprintf("Stance: %v\n", s.Stance)
	stateString += fmt.Sprintf("AP: %v\n", s.AP)
	stateString += fmt.Sprintf("CanMove: %v\n", s.CanMove)
	stateString += fmt.Sprintf("OccupiedOnMap: %v\n", s.OccupiedOnMap)
	return stateString
}

func (s CompleteUnitState) Diff(other CompleteUnitState) string {
	stateString := "\n"
	if s.ID != other.ID {
		stateString += fmt.Sprintf("ID: %v != %v\n", s.ID, other.ID)
	} else {
		stateString += fmt.Sprintf("ID: %v\n", s.ID)
	}
	if s.Name != other.Name {
		stateString += fmt.Sprintf("Name: %v != %v\n", s.Name, other.Name)
	} else {
		stateString += fmt.Sprintf("Name: %v\n", s.Name)
	}

	if s.Forward.ApproxEqualThreshold(other.Forward, PositionalTolerance) == false {
		stateString += fmt.Sprintf("AimDirection: %v != %v\n", s.Forward, other.Forward)
	} else {
		stateString += fmt.Sprintf("AimDirection: %v\n", s.Forward)
	}

	if s.Position.ApproxEqualThreshold(other.Position, PositionalTolerance) == false {
		stateString += fmt.Sprintf("Origin: %v != %v\n", s.Position, other.Position)
	} else {
		stateString += fmt.Sprintf("Origin: %v\n", s.Position)
	}

	if s.Stance != other.Stance {
		stateString += fmt.Sprintf("Stance: %v != %v\n", s.Stance, other.Stance)
	} else {
		stateString += fmt.Sprintf("Stance: %v\n", s.Stance)
	}

	if s.AP != other.AP {
		stateString += fmt.Sprintf("AP: %v != %v\n", s.AP, other.AP)
	} else {
		stateString += fmt.Sprintf("AP: %v\n", s.AP)
	}

	if s.CanMove != other.CanMove {
		stateString += fmt.Sprintf("CanMove: %v != %v\n", s.CanMove, other.CanMove)
	} else {
		stateString += fmt.Sprintf("CanMove: %v\n", s.CanMove)
	}

	if len(s.OccupiedOnMap) != len(other.OccupiedOnMap) {
		stateString += fmt.Sprintf("OccupiedOnMap: %v != %v\n", s.OccupiedOnMap, other.OccupiedOnMap)
	} else {
		stateString += fmt.Sprintf("OccupiedOnMap: %v\n", s.OccupiedOnMap)
	}
	occupiedDiffers := false
	for i := range s.OccupiedOnMap {
		if s.OccupiedOnMap[i] != other.OccupiedOnMap[i] {
			stateString += fmt.Sprintf("OccupiedOnMap[%d]: %v != %v\n", i, s.OccupiedOnMap, other.OccupiedOnMap)
			occupiedDiffers = true
			break
		}
	}
	if !occupiedDiffers {
		stateString += fmt.Sprintf("OccupiedOnMap: %v\n", s.OccupiedOnMap)
	}
	return stateString
}

func (s CompleteUnitState) Equals(other CompleteUnitState) (bool, string) {
	if s.ID != other.ID {
		return false, fmt.Sprintf("ID: %v != %v", s.ID, other.ID)
	}
	if s.Name != other.Name {
		return false, fmt.Sprintf("Name: %v != %v", s.Name, other.Name)
	}
	if s.Forward.ApproxEqualThreshold(other.Forward, PositionalTolerance) == false {
		return false, fmt.Sprintf("AimDirection: %v != %v", s.Forward, other.Forward)
	}
	if s.Position.ApproxEqualThreshold(other.Position, PositionalTolerance) == false {
		return false, fmt.Sprintf("Origin: %v != %v", s.Position, other.Position)
	}
	if s.Stance != other.Stance {
		return false, fmt.Sprintf("Stance: %v != %v", s.Stance, other.Stance)
	}
	if s.AP != other.AP {
		return false, fmt.Sprintf("AP: %v != %v", s.AP, other.AP)
	}
	if s.CanMove != other.CanMove {
		return false, fmt.Sprintf("CanMove: %v != %v", s.CanMove, other.CanMove)
	}
	if len(s.OccupiedOnMap) != len(other.OccupiedOnMap) {
		return false, fmt.Sprintf("OccupiedOnMap: %v != %v", s.OccupiedOnMap, other.OccupiedOnMap)
	}
	for i := range s.OccupiedOnMap {
		if s.OccupiedOnMap[i] != other.OccupiedOnMap[i] {
			return false, fmt.Sprintf("OccupiedOnMap: %v != %v", s.OccupiedOnMap, other.OccupiedOnMap)
		}
	}
	return true, ""
}

type CompleteGameState struct {
	AllUnits map[uint64]CompleteUnitState
}

func (g *GameInstance) DebugGetCompleteState() CompleteGameState {
	gs := CompleteGameState{}
	allUnits := make(map[uint64]CompleteUnitState)
	for _, unit := range g.units {
		us := CompleteUnitState{
			ID:            unit.UnitID(),
			Name:          unit.Name,
			Forward:       unit.Transform.GetForward(),
			Position:      unit.Transform.GetPosition(),
			Stance:        unit.CurrentStance,
			AP:            unit.ActionPoints,
			CanMove:       unit.CanMove(),
			OccupiedOnMap: g.voxelMap.DebugGetOccupiedBlocks(unit.UnitID()),
		}
		allUnits[unit.UnitID()] = us
	}
	gs.AllUnits = allUnits
	return gs
}
