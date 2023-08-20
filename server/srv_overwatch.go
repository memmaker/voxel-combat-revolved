package server

import (
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/voxel"
	"github.com/memmaker/battleground/game"
)

type ServerActionOverwatch struct {
	engine       *game.GameInstance
	unit         *game.UnitInstance
	createRay    func() (mgl32.Vec3, mgl32.Vec3)
	aimDirection mgl32.Vec3
	totalAPCost  int
	targets      []voxel.Int3
	gameAction   *game.ActionOverwatch
}

func (a *ServerActionOverwatch) SetAPCost(newCost int) {
	a.totalAPCost = newCost
}

func (a *ServerActionOverwatch) IsTurnEnding() bool {
	return true
}

func (a *ServerActionOverwatch) IsValid() (bool, string) {
	// check if weapon is ready
	if !a.unit.Weapon.IsReady() {
		return false, "Weapon is not ready"
	}

	if a.unit.GetIntegerAP() < a.totalAPCost {
		return false, fmt.Sprintf("Not enough AP for overwatch. Need %d, have %d", a.totalAPCost, a.unit.GetIntegerAP())
	}

	for _, target := range a.targets {
		if !a.gameAction.IsValidTarget(target) {
			return false, fmt.Sprintf("Invalid target for overwatch: %v", target)
		}
	}

	return true, ""
}

func NewServerActionOverwatch(g *game.GameInstance, unit *game.UnitInstance, targets []voxel.Int3) ServerAction {
	return &ServerActionOverwatch{
		targets:     targets,
		engine:      g,
		unit:        unit,
		totalAPCost: int(unit.GetWeapon().Definition.BaseAPForShot) + 1,
		gameAction:  game.NewActionOverwatch(g, unit),
	}
}

func (a *ServerActionOverwatch) Execute(mb *game.MessageBuffer) {
	a.engine.RegisterOverwatch(a.unit, a.targets)
	mb.AddMessageForAll(game.VisualBeginOverwatch{
		Watcher:          a.unit.UnitID(),
		WatchedLocations: a.targets,
	})
}
