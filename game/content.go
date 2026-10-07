package game

import (
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/memmaker/battleground/engine/afs"
)

// MapEntry is a playable map as listed in the content files.
type MapEntry struct {
	Name string // display name
	File string // map file name (without extension) in assets/maps
}

// Content is all data-driven game content: maps, factions with their unit types, weapons and items.
// It lives as JSON in assets/content so it can be modded without recompiling.
type Content struct {
	Maps     []MapEntry
	Factions []FactionDefinition
	Weapons  []WeaponDefinition
	Items    []ItemDefinition
}

const contentDirectory = "assets/content"

func LoadContent() (Content, error) {
	var content Content
	files := map[string]any{
		"maps.json":     &content.Maps,
		"factions.json": &content.Factions,
		"weapons.json":  &content.Weapons,
		"items.json":    &content.Items,
	}
	for filename, target := range files {
		data, err := afs.ReadFile(path.Join(contentDirectory, filename))
		if err != nil {
			return content, fmt.Errorf("reading %s: %w", filename, err)
		}
		if err = json.Unmarshal(data, target); err != nil {
			return content, fmt.Errorf("parsing %s: %w", filename, err)
		}
	}
	return content, content.Validate()
}

// Validate checks the references between content entries, so modding mistakes fail loudly at startup.
func (c Content) Validate() error {
	nextUnitID := uint64(0)
	for _, faction := range c.Factions {
		for _, unit := range faction.Units {
			if unit.ID != nextUnitID {
				return fmt.Errorf("unit type IDs must be consecutive starting at 0 (faction %s has ID %d, expected %d)", faction.Name, unit.ID, nextUnitID)
			}
			nextUnitID++
		}
	}
	if len(c.Maps) == 0 {
		return fmt.Errorf("no maps defined")
	}
	return nil
}

// RegisterMapInContent adds a saved map to assets/content/maps.json if it is not listed yet.
func RegisterMapInContent(mapFile string) error {
	filename := path.Join(contentDirectory, "maps.json")
	var maps []MapEntry
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(data, &maps); err != nil {
		return err
	}
	for _, entry := range maps {
		if entry.File == mapFile {
			return nil
		}
	}
	maps = append(maps, MapEntry{Name: mapFile, File: mapFile})
	data, err = json.MarshalIndent(maps, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, append(data, '\n'), 0o644)
}
