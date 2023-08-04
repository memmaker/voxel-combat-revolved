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
		Speed:  5,
		OccupiedBlockOffsets: []voxel.Int3{
			{0, 0, 0},
			{0, 1, 0},
		},
	}

	battleServer := server.NewBattleServer()

	battleServer.AddMap("Dev Map", "./assets/maps/map.bin")

	battleServer.AddFaction(server.FactionDefinition{
		Name:  "X-Com",
		Color: mgl32.Vec3{0, 0, 1},
		Units: []game.UnitDefinition{
			{
				ID:        0,
				CoreStats: defaultCoreStats,
				ModelFile: "./assets/models/Guard3.glb",
				ClientRepresentation: game.UnitClientDefinition{
					TextureFile: "./assets/textures/skins/steve.png",
				},
			},
			{
				ID:        1,
				CoreStats: defaultCoreStats,
				ModelFile: "./assets/models/Guard3.glb",
				ClientRepresentation: game.UnitClientDefinition{
					TextureFile: "./assets/textures/skins/soldier4.png",
				},
			},
		},
	})
	battleServer.AddFaction(server.FactionDefinition{
		Name:  "Deep Ones",
		Color: mgl32.Vec3{1, 0, 0},
		Units: []game.UnitDefinition{
			{
				ID:        2,
				CoreStats: defaultCoreStats,
				ModelFile: "./assets/models/Guard3.glb",
				ClientRepresentation: game.UnitClientDefinition{
					TextureFile: "./assets/textures/skins/deep_monster2.png",
				},
			},
		},
	})
	return battleServer
}
