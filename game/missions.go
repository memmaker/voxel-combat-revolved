package game

import "github.com/memmaker/battleground/engine/voxel"

type PlacementMode string

const (
    PlacementModeRandom PlacementMode = "random"
    PlacementModeManual PlacementMode = "manual"
)

type MissionScenario string

const (
    MissionScenarioDeathmatch MissionScenario = "deathmatch"
    MissionScenarioDefend     MissionScenario = "defend"
)

type MissionDetails struct {
    Placement             PlacementMode
    Scenario              MissionScenario
    DestroyableObjectives []voxel.Int3
    ObjectiveLife         int
    damage                map[voxel.Int3]int
    TurnLimit             int
}

func NewRandomDeathmatch() *MissionDetails {
    return &MissionDetails{
        Placement: PlacementModeRandom,
        Scenario:  MissionScenarioDeathmatch,
    }
}

func NewManualDeathmatch() *MissionDetails {
    return &MissionDetails{
        Placement: PlacementModeManual,
        Scenario:  MissionScenarioDeathmatch,
    }
}

func NewRandomDefend() *MissionDetails {
    return &MissionDetails{
        Placement:     PlacementModeRandom,
        Scenario:      MissionScenarioDefend,
        ObjectiveLife: 10,
        TurnLimit: 10,
        damage:        make(map[voxel.Int3]int),
    }
}

func (d *MissionDetails) SyncFromMap(mapData MapMetadata) {
    if d.Scenario != MissionScenarioDefend {
        return
    }
    objectives := make([]voxel.Int3, len(mapData.PoIPlacements))
    for i, pos := range mapData.PoIPlacements {
        objectives[i] = pos.Sub(voxel.Int3{X: 0, Y: 1, Z: 0}) // HACKY TODO, because the POIs are on the ground level..
    }
    d.DestroyableObjectives = objectives
}

// TryDamageObjective returns true if the objective was destroyed
func (d *MissionDetails) TryDamageObjective(atPos voxel.Int3, damage int) bool {
    if d.damage == nil {
        d.damage = make(map[voxel.Int3]int)
    }
    if _, exists := d.damage[atPos]; !exists {
        d.damage[atPos] = 0
    }
    d.damage[atPos] += damage
    return d.damage[atPos] >= d.ObjectiveLife
}

func (d *MissionDetails) GetObjectiveDamage(atPos voxel.Int3) int {
    if _, exists := d.damage[atPos]; !exists {
        return 0
    }
    return d.damage[atPos]
}

func (d *MissionDetails) IsObjectiveDestroyed(atPos voxel.Int3) bool {
    return d.GetObjectiveDamage(atPos) >= d.ObjectiveLife
}

func (d *MissionDetails) AllObjectivesDestroyed() bool {
    for _, pos := range d.DestroyableObjectives {
        if !d.IsObjectiveDestroyed(pos) {
            return false
        }
    }
    return true
}
