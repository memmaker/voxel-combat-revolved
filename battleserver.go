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
		ThrowVelocity:  12,
		BaseAPForThrow: 2,
	}

	battleServer := server.NewBattleServer()

	battleServer.AddMap("Dev Map", "map")
	battleServer.AddFaction(game.FactionDefinition{
		Name:  "X-Com",
		Color: mgl32.Vec3{0, 0, 1},
		Units: []game.UnitDefinition{
			{
				ID:        0,
				CoreStats: defaultCoreStats,
				ModelFile: "human",
				/*
					AnimationMap: map[string]string{
						"idle":      game.AnimationIdle.Str(),
						"idle2":     game.AnimationWeaponIdle.Str(),
						"hit":       game.AnimationHit.Str(),
						"run":       game.AnimationWeaponWalk.Str(),
						"death":     game.AnimationDeath.Str(),
						"climb":     game.AnimationClimb.Str(),
						"drop":      game.AnimationDrop.Str(),
						"wall_idle": game.AnimationWallIdle.Str(),
					},
				*/
				ClientRepresentation: game.UnitClientDefinition{
					TextureFile: "steve",
				},
			},
			{
				ID:        1,
				CoreStats: defaultCoreStats,
				ModelFile: "walker_3x3",
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
				ModelFile: "human",
				/*
					AnimationMap: map[string]string{
						"idle":      game.AnimationIdle.Str(),
						"idle2":     game.AnimationWeaponIdle.Str(),
						"hit":       game.AnimationHit.Str(),
						"run":       game.AnimationWeaponWalk.Str(),
						"death":     game.AnimationDeath.Str(),
						"climb":     game.AnimationClimb.Str(),
						"drop":      game.AnimationDrop.Str(),
						"wall_idle": game.AnimationWallIdle.Str(),
					},
				*/
				ClientRepresentation: game.UnitClientDefinition{
					//TextureFile: "deep_monster2",
				},
			},
			{
				ID:        3,
				CoreStats: defaultCoreStats,
				ModelFile: "deep_monster_3x3",
				//				OccupiedBlockOffsets: threeByThree,
				ClientRepresentation: game.UnitClientDefinition{
					TextureFile: "",
				},
			},
		},
	})
	battleServer.AddWeapon(game.WeaponDefinition{
		UniqueName:          "M1911 Pistol",
		Model:               "Rifle",
		WeaponType:          game.WeaponPistol,
		AccuracyModifier:    0.95,
		BulletsPerShot:      2,
		EffectiveRange:      12,
		MaxRange:            30,
		MagazineSize:        4,
		BaseDamagePerBullet: 2,
		MinFOVForZoom:       45,
		BaseAPForShot:       2,
		BaseAPForReload:     2,
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

	battleServer.AddWeapon(game.WeaponDefinition{
		UniqueName:          "LAW Rocket",
		Model:               "Sniper",
		WeaponType:          game.WeaponRocketLauncher,
		AccuracyModifier:    0.9,
		BulletsPerShot:      1,
		EffectiveRange:      20,
		MaxRange:            60,
		MagazineSize:        1,
		BaseDamagePerBullet: 0,
		MinFOVForZoom:       40,
		BaseAPForShot:       3,
		BaseAPForReload:     3,
		InsteadOfDamage:     game.TargetedEffectExplosion,
		Radius:              3,
	})

	battleServer.AddItem(game.ItemDefinition{
		UniqueName:  "Smoke Grenade",
		Model:       "SmokeGrenade",
		ItemType:    game.ItemTypeGrenade,
		Radius:      5.0,
		TurnsToLive: 3,
		Effect: game.TargetedEffectSmokeCloud,
	})

	battleServer.AddItem(game.ItemDefinition{
		UniqueName:  "Poison Grenade",
		Model:       "PoisonGrenade",
		ItemType:    game.ItemTypeGrenade,
		Radius:      5.0,
		TurnsToLive: 3,
		Effect:      game.TargetedEffectPoisonCloud,
	})

	battleServer.AddItem(game.ItemDefinition{
		UniqueName:  "Frag Grenade",
		Model:       "FragGrenade",
		ItemType:    game.ItemTypeGrenade,
		Radius:      3.0,
		TurnsToLive: 1,
		Effect:      game.TargetedEffectExplosion,
	})
	return battleServer
}
