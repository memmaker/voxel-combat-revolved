package main

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/game"
)

func runGame() {
	width := 800
	height := 600
	battleGame := game.NewBattleGame("BattleGrounds", width, height)
	battleGame.LoadEmptyWorld()
	battleGame.SpawnUnit(mgl32.Vec3{4, 1, 9})
	battleGame.Run()
}
