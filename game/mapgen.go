package game

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"github.com/memmaker/battleground/engine/voxel"
)

// GeneratedMapPrefix marks map names that are generated from a seed instead of loaded from a file,
// e.g. "gen:urban:1234". Server and clients generate identical maps from the same name.
const GeneratedMapPrefix = "gen:"

const generatedBlockSet = "star_odyssey_01"

func IsGeneratedMap(mapName string) bool {
	return strings.HasPrefix(mapName, GeneratedMapPrefix)
}

// ParseGeneratedMap splits "gen:<style>:<seed>" into its parts.
func ParseGeneratedMap(mapName string) (string, int64, error) {
	parts := strings.Split(strings.TrimPrefix(mapName, GeneratedMapPrefix), ":")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("generated map names look like gen:<style>:<seed>, got %q", mapName)
	}
	seed, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("invalid seed in %q: %w", mapName, err)
	}
	return parts[0], seed, nil
}

// WithRandomSeed turns "gen:<style>:random" into a concrete seed; other names are returned unchanged.
func WithRandomSeed(mapName string, rng *rand.Rand) string {
	if IsGeneratedMap(mapName) && strings.HasSuffix(mapName, ":random") {
		return strings.TrimSuffix(mapName, "random") + strconv.FormatInt(rng.Int63n(1_000_000), 10)
	}
	return mapName
}

type mapStyle struct {
	buildings     int     // attempts to place buildings
	twoStoreys    float64 // chance for a second floor
	wallDamage    float64 // chance for a wall block to be missing (ruins)
	roofChance    float64
	coverObjects  int
	explosives    int
	wallMaterials []string
	floorPatches  []string
}

var mapStyles = map[string]mapStyle{
	"urban": {
		buildings: 40, twoStoreys: 0.5, wallDamage: 0.02, roofChance: 1.0, coverObjects: 26, explosives: 4,
		wallMaterials: []string{"bricks", "sandstone", "granite", "birch_planks", "deepslate_tiles", "red_nether_bricks"},
		floorPatches:  []string{"gravel", "granite", "clay"},
	},
	"ruins": {
		buildings: 30, twoStoreys: 0.4, wallDamage: 0.18, roofChance: 0.4, coverObjects: 34, explosives: 5,
		wallMaterials: []string{"bricks", "cracked_nether_bricks", "sandstone", "granite"},
		floorPatches:  []string{"gravel", "sandstone", "granite"},
	},
	"empty": {wallMaterials: []string{"bricks"}},
	"outpost": {
		buildings: 12, twoStoreys: 0.3, wallDamage: 0.0, roofChance: 1.0, coverObjects: 44, explosives: 6,
		wallMaterials: []string{"iron_block", "weathered_copper", "copper_block", "exposed_copper"},
		floorPatches:  []string{"weathered_cut_copper", "gravel", "black_terracotta"},
	},
}

const (
	genSizeX       = 64
	genSizeZ       = 64
	genChunkWidth  = 16
	genChunkHeight = 8
	genChunksY     = 2
	storeyHeight   = 4 // floor slab + 3 blocks of air: units are 2 blocks tall and need headroom
)

type rect struct{ x, z, w, d int32 }

func (r rect) overlaps(o rect, margin int32) bool {
	return r.x-margin < o.x+o.w && o.x-margin < r.x+r.w && r.z-margin < o.z+o.d && o.z-margin < r.z+r.d
}

func (r rect) contains(x, z int32) bool {
	return x >= r.x && x < r.x+r.w && z >= r.z && z < r.z+r.d
}

type battlefieldGenerator struct {
	rng       *rand.Rand
	style     mapStyle
	library   *BlockLibrary
	voxelMap  *voxel.Map
	buildings []rect
	reserved  []rect // spawn zones and objective areas stay clear of buildings
	lamps     []voxel.Int3
}

// GenerateBattlefield builds a map from a style and seed. The block library must be the generatedBlockSet.
func GenerateBattlefield(styleName string, seed int64, library *BlockLibrary) (*voxel.Map, MapMetadata, error) {
	style, known := mapStyles[styleName]
	if !known {
		return nil, MapMetadata{}, fmt.Errorf("unknown map style %q", styleName)
	}
	g := &battlefieldGenerator{
		rng:      rand.New(rand.NewSource(seed)),
		style:    style,
		library:  library,
		voxelMap: voxel.NewMapWithEmptyChunks(genSizeX/genChunkWidth, genChunksY, genSizeZ/genChunkWidth, genChunkWidth, genChunkHeight),
	}
	spawnA := rect{x: 10, z: 1, w: genSizeX - 20, d: 7}
	spawnB := rect{x: 10, z: genSizeZ - 8, w: genSizeX - 20, d: 7}
	objectiveArea := rect{x: genSizeX/2 - 9, z: 11, w: 18, d: 6}
	g.reserved = []rect{spawnA, spawnB, objectiveArea}

	g.ground()
	g.placeBuildings()
	g.scatterCover()
	var objectives []voxel.Int3
	if styleName != "empty" {
		objectives = g.placeObjectives(objectiveArea)
	}

	meta := MapMetadata{
		Name:           fmt.Sprintf("%s %d", strings.ToUpper(styleName[:1])+styleName[1:], seed),
		SpawnPositions: [][]voxel.Int3{g.freePositions(spawnA), g.freePositions(spawnB)},
		PoIPlacements:  objectives,
		Blocks:         generatedBlockSet,
		Lamps:          g.lamps,
	}
	return g.voxelMap, meta, nil
}

func (g *battlefieldGenerator) set(x, y, z int32, blockName string) {
	if x < 0 || z < 0 || x >= genSizeX || z >= genSizeZ || y < 0 || y >= genChunksY*genChunkHeight {
		return
	}
	g.voxelMap.SetBlock(x, y, z, g.library.NewBlockFromName(blockName))
}

func (g *battlefieldGenerator) clear(x, y, z int32) {
	g.voxelMap.SetBlock(x, y, z, voxel.NewAirBlock())
}

func (g *battlefieldGenerator) solid(x, y, z int32) bool {
	return g.voxelMap.IsSolidBlockAt(x, y, z)
}

func (g *battlefieldGenerator) pick(options []string) string {
	return options[g.rng.Intn(len(options))]
}

func (g *battlefieldGenerator) ground() {
	for x := int32(0); x < genSizeX; x++ {
		for z := int32(0); z < genSizeZ; z++ {
			g.set(x, 0, z, "bricks")
		}
	}
	// a few patches of different ground make the battlefield easier to read
	for i := 0; i < 14 && len(g.style.floorPatches) > 0; i++ {
		material := g.pick(g.style.floorPatches)
		px, pz := g.rng.Int31n(genSizeX), g.rng.Int31n(genSizeZ)
		w, d := 3+g.rng.Int31n(7), 3+g.rng.Int31n(7)
		for x := px; x < px+w && x < genSizeX; x++ {
			for z := pz; z < pz+d && z < genSizeZ; z++ {
				g.set(x, 0, z, material)
			}
		}
	}
}

func (g *battlefieldGenerator) placeBuildings() {
	for attempt := 0; attempt < g.style.buildings; attempt++ {
		candidate := rect{w: 7 + g.rng.Int31n(7), d: 7 + g.rng.Int31n(6)}
		candidate.x = 1 + g.rng.Int31n(genSizeX-candidate.w-2)
		candidate.z = 10 + g.rng.Int31n(genSizeZ-candidate.d-20)
		free := true
		for _, other := range append(g.buildings, g.reserved...) {
			if candidate.overlaps(other, 3) { // keep streets at least 3 blocks wide
				free = false
				break
			}
		}
		if free {
			g.buildings = append(g.buildings, candidate)
			g.building(candidate)
		}
	}
}

func (g *battlefieldGenerator) building(r rect) {
	material := g.pick(g.style.wallMaterials)
	storeys := int32(1)
	if g.rng.Float64() < g.style.twoStoreys && r.w >= 8 && r.d >= 8 {
		storeys = 2
	}
	hasRoof := g.rng.Float64() < g.style.roofChance
	for storey := int32(0); storey < storeys; storey++ {
		base := storey * storeyHeight // floor level of this storey (the slab is at base, ground slab is y=0)
		isTop := storey == storeys-1
		for x := r.x; x < r.x+r.w; x++ {
			for z := r.z; z < r.z+r.d; z++ {
				onEdge := x == r.x || z == r.z || x == r.x+r.w-1 || z == r.z+r.d-1
				if storey > 0 {
					g.set(x, base, z, material) // floor slab of the upper storey
				}
				if onEdge {
					for y := base + 1; y < base+storeyHeight; y++ {
						if g.rng.Float64() >= g.style.wallDamage {
							g.set(x, y, z, material)
						}
					}
				}
				if isTop && hasRoof {
					g.set(x, base+storeyHeight, z, material)
				}
			}
		}
		g.windows(r, base)
		if hasRoof || !isTop { // a ceiling lamp lights up covered rooms
			g.lamps = append(g.lamps, voxel.Int3{X: r.x + r.w/2, Y: base + storeyHeight - 1, Z: r.z + r.d/2})
		}
	}
	g.doors(r)
	if storeys == 2 {
		g.stairs(r)
	}
}

// windows cut eye-level openings: the block below stays, so a window is also a firing position with cover.
func (g *battlefieldGenerator) windows(r rect, base int32) {
	eyeLevel := base + 2
	for x := r.x + 2; x < r.x+r.w-2; x += 3 {
		if g.rng.Float64() < 0.7 {
			g.clear(x, eyeLevel, r.z)
		}
		if g.rng.Float64() < 0.7 {
			g.clear(x, eyeLevel, r.z+r.d-1)
		}
	}
	for z := r.z + 2; z < r.z+r.d-2; z += 3 {
		if g.rng.Float64() < 0.7 {
			g.clear(r.x, eyeLevel, z)
		}
		if g.rng.Float64() < 0.7 {
			g.clear(r.x+r.w-1, eyeLevel, z)
		}
	}
}

// doors are two blocks wide, so units with a raised weapon fit through.
func (g *battlefieldGenerator) doors(r rect) {
	sides := g.rng.Perm(4)
	count := 1 + g.rng.Intn(2)
	for _, side := range sides[:count] {
		var x, z, dx, dz int32
		switch side {
		case 0:
			x, z, dx = r.x+1+g.rng.Int31n(r.w-3), r.z, 1
		case 1:
			x, z, dx = r.x+1+g.rng.Int31n(r.w-3), r.z+r.d-1, 1
		case 2:
			x, z, dz = r.x, r.z+1+g.rng.Int31n(r.d-3), 1
		default:
			x, z, dz = r.x+r.w-1, r.z+1+g.rng.Int31n(r.d-3), 1
		}
		for i := int32(0); i < 2; i++ {
			for y := int32(1); y <= 3; y++ {
				g.clear(x+dx*i, y, z+dz*i)
			}
		}
	}
}

// stairs climb one block per step along the inside of the west wall, two blocks wide.
func (g *battlefieldGenerator) stairs(r rect) {
	startZ := r.z + 2
	for step := int32(1); step <= 3; step++ {
		z := startZ + step - 1
		for x := r.x + 1; x <= r.x+2; x++ {
			for y := int32(1); y <= step; y++ {
				g.set(x, y, z, "stripped_oak_log")
			}
			// headroom: open the upper floor slab above the higher steps
			if step >= 2 {
				g.clear(x, storeyHeight, z)
			}
		}
	}
	// keep the landing in front of the stairs free on the upper floor
	for x := r.x + 1; x <= r.x+2; x++ {
		g.clear(x, storeyHeight+1, startZ+3)
		g.clear(x, storeyHeight+2, startZ+3)
	}
}

func (g *battlefieldGenerator) insideAnyBuilding(x, z int32, margin int32) bool {
	for _, b := range g.buildings {
		if (rect{x: b.x - margin, z: b.z - margin, w: b.w + margin*2, d: b.d + margin*2}).contains(x, z) {
			return true
		}
	}
	return false
}

func (g *battlefieldGenerator) insideReserved(x, z int32) bool {
	for _, r := range g.reserved {
		if r.contains(x, z) {
			return true
		}
	}
	return false
}

// scatterCover places crates, low and high walls and a few explosive blocks in the open.
func (g *battlefieldGenerator) scatterCover() {
	crates := []string{"barrel", "crafting_table", "furnace", "dispenser", "fletching_table", "smithing_table"}
	walls := []string{"bricks", "sandstone", "granite", "deepslate_tiles"}
	placed := 0
	for attempt := 0; attempt < g.style.coverObjects*8 && placed < g.style.coverObjects; attempt++ {
		x, z := 1+g.rng.Int31n(genSizeX-2), 1+g.rng.Int31n(genSizeZ-2)
		if g.insideAnyBuilding(x, z, 2) || g.solid(x, 1, z) {
			continue
		}
		inSpawn := g.insideReserved(x, z)
		switch kind := g.rng.Intn(10); {
		case kind < 4: // crate stack, 1 or 2 high
			height := 1 + g.rng.Int31n(2)
			if inSpawn {
				height = 1
			}
			for y := int32(1); y <= height; y++ {
				g.set(x, y, z, g.pick(crates))
			}
		case kind < 8: // low wall: half cover you can shoot over
			g.wallSegment(x, z, 1, g.pick(walls), 3+g.rng.Int31n(3))
		default: // high wall segment: full cover
			if inSpawn {
				continue
			}
			g.wallSegment(x, z, 2, g.pick(walls), 2+g.rng.Int31n(3))
		}
		placed++
	}
	for i := 0; i < g.style.explosives; i++ {
		for attempt := 0; attempt < 40; attempt++ {
			x, z := 2+g.rng.Int31n(genSizeX-4), 12+g.rng.Int31n(genSizeZ-24)
			if !g.insideAnyBuilding(x, z, 1) && !g.solid(x, 1, z) && !g.insideReserved(x, z) {
				g.set(x, 1, z, "tnt")
				break
			}
		}
	}
}

func (g *battlefieldGenerator) wallSegment(x, z, height int32, material string, length int32) {
	dx, dz := int32(1), int32(0)
	if g.rng.Intn(2) == 0 {
		dx, dz = 0, 1
	}
	for i := int32(0); i < length; i++ {
		px, pz := x+dx*i, z+dz*i
		if g.insideAnyBuilding(px, pz, 2) {
			return
		}
		for y := int32(1); y <= height; y++ {
			g.set(px, y, pz, material)
		}
	}
}

// placeObjectives puts two destructible targets for the Defend mission in front of the first spawn.
// The returned points of interest are one block above each target (see MissionDetails.SyncFromMap).
func (g *battlefieldGenerator) placeObjectives(area rect) []voxel.Int3 {
	var pois []voxel.Int3
	for _, x := range []int32{area.x + 3, area.x + area.w - 4} {
		z := area.z + area.d/2
		for dx := int32(-1); dx <= 1; dx++ { // a low wall around each target makes it defensible
			g.set(x+dx, 1, z+2, "sandstone")
		}
		g.set(x, 1, z, "target")
		pois = append(pois, voxel.Int3{X: x, Y: 2, Z: z})
	}
	return pois
}

// freePositions lists standing positions in an area with two free blocks of headroom.
func (g *battlefieldGenerator) freePositions(area rect) []voxel.Int3 {
	var positions []voxel.Int3
	for x := area.x; x < area.x+area.w; x++ {
		for z := area.z; z < area.z+area.d; z++ {
			if g.solid(x, 0, z) && !g.solid(x, 1, z) && !g.solid(x, 2, z) && !g.solid(x, 3, z) {
				positions = append(positions, voxel.Int3{X: x, Y: 1, Z: z})
			}
		}
	}
	return positions
}
