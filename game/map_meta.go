package game

import (
	"encoding/json"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
	"os"
)

type MapMetadata struct {
	Name                string
	FloorCeilingHeights [][2]int
	SpawnPositions      [][]voxel.Int3
	PoIPlacements       []voxel.Int3
}

func (m *MapMetadata) SaveToDisk(mapfilename string) {
	metaFilename := mapfilename + ".meta"
	file, err := os.Create(metaFilename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(m)
	if err != nil {
		panic(err)
	}
}

func NewMapMetadataFromFile(filename string) *MapMetadata {
	if util.DoesFileExist(filename) {
		var metadata MapMetadata
		if util.FromJson(filename, &metadata) {
			return &metadata
		}
	}
	return &MapMetadata{
		Name:                "Unnamed Map",
		FloorCeilingHeights: [][2]int{{1, 4}},
		SpawnPositions:      [][]voxel.Int3{{{0, 0, 0}}},
	}
}
