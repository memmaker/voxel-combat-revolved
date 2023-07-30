package main

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/voxel"
	"github.com/memmaker/battleground/game"
)

func runGame() {
	width := 800
	height := 600
	battleGame := game.NewBattleGame("BattleGrounds", width, height)
	battleGame.LoadMap("./assets/maps/map.bin")
	battleGame.AddFaction(game.FactionDefinition{
		Name:  "X-Com",
		Color: mgl32.Vec3{0, 0, 1},
		Units: []game.UnitDefinition{
			{
				Name:        "Soldier #1",
				SpawnPos:    voxel.Int3{X: 2, Y: 1, Z: 2},
				TextureFile: "./assets/textures/skins/steve.png",
			},
			{
				Name:        "Soldier #2",
				SpawnPos:    voxel.Int3{X: 4, Y: 1, Z: 2},
				TextureFile: "./assets/textures/skins/soldier4.png",
			},
		},
	})
	battleGame.AddFaction(game.FactionDefinition{
		Name:  "Deep Ones",
		Color: mgl32.Vec3{1, 0, 0},
		Units: []game.UnitDefinition{
			{
				Name:        "Deep Monster",
				SpawnPos:    voxel.Int3{X: 5, Y: 1, Z: 12},
				TextureFile: "./assets/textures/skins/deep_monster2.png",
			},
		},
	})
	battleGame.FirstTurn("X-Com")
	battleGame.Run()
}
