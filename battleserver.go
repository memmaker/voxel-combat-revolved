package main

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/game"
	"github.com/memmaker/battleground/server"
)

func NewBattleServer() *server.BattleServer {
	defaultCoreStats := game.UnitCoreStats{
		Health:          10,
		MovementPerAP:   3,
		Accuracy:        0.9,
		MaxActionPoints: 4,
	}

	battleServer := server.NewBattleServer()

	battleServer.AddMap("Dev Map", "./assets/maps/map.bin")
	/*
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
	*/
	battleServer.AddFaction(game.FactionDefinition{
		Name:  "X-Com",
		Color: mgl32.Vec3{0, 0, 1},
		Units: []game.UnitDefinition{
			{
				ID:        0,
				CoreStats: defaultCoreStats,
				ModelFile: "./assets/models/human.glb",
				ClientRepresentation: game.UnitClientDefinition{
					TextureFile: "./assets/textures/skins/steve.png",
				},
			},
			{
				ID:        1,
				CoreStats: defaultCoreStats,
				ModelFile: "./assets/models/walker_3x3.glb",
				//				OccupiedBlockOffsets: threeByThree,
				ClientRepresentation: game.UnitClientDefinition{
					TextureFile: "",
				},
			},
		},
	})
	battleServer.AddFaction(game.FactionDefinition{
		Name:  "Deep Ones",
		Color: mgl32.Vec3{1, 0, 0},
		Units: []game.UnitDefinition{
			{
				ID:        2,
				CoreStats: defaultCoreStats,
				ModelFile: "./assets/models/human.glb",
				ClientRepresentation: game.UnitClientDefinition{
					TextureFile: "./assets/textures/skins/deep_monster2.png",
				},
			},
			{
				ID:        3,
				CoreStats: defaultCoreStats,
				ModelFile: "./assets/models/deep_monster_3x3.glb",
				//				OccupiedBlockOffsets: threeByThree,
				ClientRepresentation: game.UnitClientDefinition{
					TextureFile: "",
				},
			},
		},
	})
	battleServer.AddWeapon(game.WeaponDefinition{
		UniqueName:          "M16 Rifle",
		Model:               "Rifle",
		WeaponType:          game.WeaponAutomatic,
		AccuracyModifier:    0.75,
		BulletsPerShot:      3,
		EffectiveRange:      14,
		MaxRange:            50,
		MagazineSize:        5,
		BaseDamagePerBullet: 3,
		MinFOVForZoom:       40,
		BaseAPForShot:       2,
		BaseAPForReload:     2,
	})

	battleServer.AddWeapon(game.WeaponDefinition{
		UniqueName:          "Mossberg 500",
		Model:               "Mossberg",
		WeaponType:          game.WeaponShotgun,
		AccuracyModifier:    0.5,
		BulletsPerShot:      5,
		EffectiveRange:      7,
		MaxRange:            14,
		MagazineSize:        3,
		BaseDamagePerBullet: 2,
		MinFOVForZoom:       45,
		BaseAPForShot:       2,
		BaseAPForReload:     2,
	})

	battleServer.AddWeapon(game.WeaponDefinition{
		UniqueName:          "Steyr SSG 69",
		Model:               "Sniper",
		WeaponType:          game.WeaponSniper,
		AccuracyModifier:    1.0,
		BulletsPerShot:      1,
		EffectiveRange:      20,
		MaxRange:            100,
		MagazineSize:        3,
		BaseDamagePerBullet: 5,
		MinFOVForZoom:       20,
		BaseAPForShot:       3,
		BaseAPForReload:     3,
	})
	return battleServer
}
