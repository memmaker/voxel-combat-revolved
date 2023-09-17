package game

import (
	"encoding/json"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
	"os"
)

type MapMetadata struct {
	Name           string
	SpawnPositions [][]voxel.Int3
	PoIPlacements  []voxel.Int3
}

func (m *MapMetadata) SaveToDisk(mapfilename string) error {
	metaFilename := mapfilename + ".meta"
	file, err := os.Create(metaFilename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(m)
}

func NewMapMetadataFromFile(filename string) MapMetadata {
	if util.DoesFileExist(filename) {
		var metadata MapMetadata
		data, err := os.ReadFile(filename)
		if err == nil {
			if util.FromJson(string(data), &metadata) {
				return metadata
			}
		}
	}
	return MapMetadata{
		Name:           "Unnamed Map",
		SpawnPositions: [][]voxel.Int3{{}},
	}
}
