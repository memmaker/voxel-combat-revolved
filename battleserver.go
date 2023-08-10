package main

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/voxel"
	"github.com/memmaker/battleground/game"
	"github.com/memmaker/battleground/server"
)

func NewBattleServer() *server.BattleServer {
	defaultCoreStats := game.UnitCoreStats{
		Health: 10,
		Speed:  15,
	}

	battleServer := server.NewBattleServer()

	battleServer.AddMap("Dev Map", "./assets/maps/map.bin")

	humanoid := []voxel.Int3{
		{0, 0, 0},
		{0, 1, 0},
	}
	threeByThree := []voxel.Int3{
		{0, 0, 0},
		{0, 1, 0},
		{0, 2, 0},

		{1, 0, 0},
		{1, 1, 0},
		{1, 2, 0},

		{-1, 0, 0},
		{-1, 1, 0},
		{-1, 2, 0},

		{0, 0, 1},
		{0, 1, 1},
		{0, 2, 1},

		{1, 0, 1},
		{1, 1, 1},
		{1, 2, 1},

		{-1, 0, 1},
		{-1, 1, 1},
		{-1, 2, 1},

		{0, 0, -1},
		{0, 1, -1},
		{0, 2, -1},

		{1, 0, -1},
		{1, 1, -1},
		{1, 2, -1},

		{-1, 0, -1},
		{-1, 1, -1},
		{-1, 2, -1},
	}
	battleServer.AddFaction(server.FactionDefinition{
		Name:  "X-Com",
		Color: mgl32.Vec3{0, 0, 1},
		Units: []game.UnitDefinition{
			{
				ID:                   0,
				CoreStats:            defaultCoreStats,
				ModelFile:            "./assets/models/human.glb",
				OccupiedBlockOffsets: humanoid,
				ClientRepresentation: game.UnitClientDefinition{
					TextureFile: "./assets/textures/skins/steve.png",
				},
			},
			{
				ID:                   1,
				CoreStats:            defaultCoreStats,
				ModelFile:            "./assets/models/walker_3x3.glb",
				OccupiedBlockOffsets: threeByThree,
				ClientRepresentation: game.UnitClientDefinition{
					TextureFile: "",
				},
			},
		},
	})
	battleServer.AddFaction(server.FactionDefinition{
		Name:  "Deep Ones",
		Color: mgl32.Vec3{1, 0, 0},
		Units: []game.UnitDefinition{
			{
				ID:                   2,
				CoreStats:            defaultCoreStats,
				ModelFile:            "./assets/models/human.glb",
				OccupiedBlockOffsets: humanoid,
				ClientRepresentation: game.UnitClientDefinition{
					TextureFile: "./assets/textures/skins/deep_monster2.png",
				},
			},
			{
				ID:                   3,
				CoreStats:            defaultCoreStats,
				ModelFile:            "./assets/models/deep_monster_3x3.glb",
				OccupiedBlockOffsets: threeByThree,
				ClientRepresentation: game.UnitClientDefinition{
					TextureFile: "",
				},
			},
		},
	})
	return battleServer
}
