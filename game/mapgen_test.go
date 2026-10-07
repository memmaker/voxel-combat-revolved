package game

import (
	"os"
	"testing"

	"github.com/memmaker/battleground/engine/voxel"
)

func TestMain(m *testing.M) {
	// assets are loaded relative to the repository root
	if err := os.Chdir(".."); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

// walkableFrom flood-fills the standing positions reachable by a humanoid.
func walkableFrom(voxelMap *voxel.Map, start voxel.Int3) map[voxel.Int3]bool {
	seen := map[voxel.Int3]bool{start: true}
	queue := []voxel.Int3{start}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, next := range voxelMap.GetNeighborsForGroundMovement(current, func(n voxel.Int3) bool {
			ok, _ := voxelMap.IsHumanoidPlaceable(n)
			return ok
		}) {
			if !seen[next] {
				seen[next] = true
				queue = append(queue, next)
			}
		}
	}
	return seen
}

func TestGeneratedMapsArePlayable(t *testing.T) {
	assets := NewAssets()
	for style := range mapStyles {
		upperFloorsReached := 0
		for seed := int64(1); seed <= 25; seed++ {
			library := assets.LoadBlockLibrary(generatedBlockSet)
			voxelMap, meta, err := GenerateBattlefield(style, seed, library)
			if err != nil {
				t.Fatal(err)
			}
			if len(meta.SpawnPositions) != 2 || len(meta.SpawnPositions[0]) < 20 || len(meta.SpawnPositions[1]) < 20 {
				t.Fatalf("%s/%d: spawn zones too small", style, seed)
			}
			if style != "empty" && len(meta.PoIPlacements) != 2 {
				t.Fatalf("%s/%d: expected 2 objectives", style, seed)
			}
			reachable := walkableFrom(voxelMap, meta.SpawnPositions[0][0])
			if !reachable[meta.SpawnPositions[1][0]] {
				t.Errorf("%s/%d: the spawn zones are not connected", style, seed)
			}
			for pos := range reachable {
				if pos.Y > storeyHeight {
					upperFloorsReached++
					break
				}
			}
		}
		t.Logf("%s: upper floors reachable in %d of 25 maps", style, upperFloorsReached)
	}
}

func TestGeneratedMapsAreDeterministic(t *testing.T) {
	assets := NewAssets()
	first, _, _ := GenerateBattlefield("urban", 42, assets.LoadBlockLibrary(generatedBlockSet))
	second, _, _ := GenerateBattlefield("urban", 42, assets.LoadBlockLibrary(generatedBlockSet))
	for x := int32(0); x < genSizeX; x++ {
		for y := int32(0); y < genChunksY*genChunkHeight; y++ {
			for z := int32(0); z < genSizeZ; z++ {
				if first.GetGlobalBlock(x, y, z).ID != second.GetGlobalBlock(x, y, z).ID {
					t.Fatalf("maps differ at %d,%d,%d", x, y, z)
				}
			}
		}
	}
}

func TestGeneratedMapLighting(t *testing.T) {
	assets := NewAssets()
	voxelMap, meta, _ := GenerateBattlefield("urban", 7, assets.LoadBlockLibrary(generatedBlockSet))
	voxelMap.EnableLighting(meta.Lamps)
	if light := voxelMap.LightAt(meta.SpawnPositions[0][0].Add(voxel.Int3{Y: 1})); light != 15 {
		t.Errorf("spawn zone should be in full sunlight, got %d", light)
	}
	if len(meta.Lamps) == 0 {
		t.Fatal("expected lamps inside buildings")
	}
	darker := 0
	for _, lamp := range meta.Lamps {
		floor := lamp.Sub(voxel.Int3{Y: 1})
		if light := voxelMap.LightAt(floor); light < 15 && light > 0 {
			darker++
		}
	}
	t.Logf("%d of %d rooms are dimmer than daylight but lit", darker, len(meta.Lamps))
	if darker == 0 {
		t.Error("no room is darker than outdoors")
	}
}

func TestSaveAndReloadEditedMap(t *testing.T) {
	assets := NewAssets()
	instance := NewGameInstanceWithDetails("editor-test", "gen:ruins:3", NewRandomDeathmatch())
	instance.voxelMap.SetBlock(5, 3, 5, instance.blockLibrary.NewBlockFromName("tnt"))
	if err := instance.SaveMapAs("zz_editor_test"); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(assets.GetMapPath("zz_editor_test"))
	defer os.Remove(assets.GetMapPath("zz_editor_test") + ".meta")

	reloaded := assets.LoadMapWithDetails("zz_editor_test", NewRandomDeathmatch())
	original := instance.voxelMap
	for x := int32(0); x < genSizeX; x++ {
		for y := int32(0); y < genChunksY*genChunkHeight; y++ {
			for z := int32(0); z < genSizeZ; z++ {
				if original.GetGlobalBlock(x, y, z).ID != reloaded.GetMap().GetGlobalBlock(x, y, z).ID {
					t.Fatalf("block %d,%d,%d differs after reload", x, y, z)
				}
			}
		}
	}
	meta := reloaded.GetMetaData()
	if len(meta.SpawnPositions[0]) == 0 || len(meta.Lamps) != len(instance.mapMeta.Lamps) || meta.Blocks != generatedBlockSet {
		t.Fatalf("metadata was not preserved: %+v", meta)
	}
}

func TestMeleeRange(t *testing.T) {
	origin := voxel.Int3{X: 5, Y: 1, Z: 5}
	for _, c := range []struct {
		to    voxel.Int3
		reach bool
	}{
		{voxel.Int3{X: 6, Y: 1, Z: 5}, true},
		{voxel.Int3{X: 6, Y: 2, Z: 6}, true},
		{voxel.Int3{X: 5, Y: 1, Z: 5}, false},
		{voxel.Int3{X: 7, Y: 1, Z: 5}, false},
		{voxel.Int3{X: 6, Y: 3, Z: 5}, false},
	} {
		if IsInMeleeRange(origin, c.to) != c.reach {
			t.Errorf("melee range to %v should be %v", c.to, c.reach)
		}
	}
}
