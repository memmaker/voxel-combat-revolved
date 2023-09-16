package server

import (
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/voxel"
	"github.com/memmaker/battleground/game"
)

type ServerActionThrow struct {
	engine           *game.GameInstance
	unit             *game.UnitInstance
	lastAimDirection mgl32.Vec3
	totalAPCost      int
	action           *game.ActionThrow
	accuracyModifier float64
	targets          []mgl32.Vec3
	itemName         string
}

func (a *ServerActionThrow) GetUnit() *game.UnitInstance {
	return a.unit
}

func (a *ServerActionThrow) GetAccuracyModifier() float64 {
	return a.accuracyModifier
}

func (a *ServerActionThrow) SetAPCost(newCost int) {
	a.totalAPCost = newCost
}

func (a *ServerActionThrow) IsTurnEnding() bool {
	return a.engine.GetRules().IsThrowTurnEnding
}

func (a *ServerActionThrow) IsValid() (bool, string) {
	// check if grenade is ready
	if !a.unit.HasItem(a.itemName) {
		return false, fmt.Sprintf("Unit does not have item %s", a.itemName)
	}

	if a.unit.GetIntegerAP() < a.totalAPCost {
		return false, fmt.Sprintf("Not enough AP for throw. Need %d, have %d", a.totalAPCost, a.unit.GetIntegerAP())
	}
	return true, ""
}
func NewServerActionThrow(g *game.GameInstance, unit *game.UnitInstance, targets []mgl32.Vec3, itemName string) *ServerActionThrow {
	// todo: add anti-cheat validation
	s := &ServerActionThrow{
		engine:           g,
		unit:             unit,
		totalAPCost:      int(unit.Definition.CoreStats.BaseAPForThrow),
		accuracyModifier: 1.0,
		targets:          targets,
		action:           game.NewActionThrow(g, unit, nil),
		itemName:         itemName,
	}
	return s
}

func (a *ServerActionThrow) Execute(mb *game.MessageBuffer) {
	currentPos := a.unit.GetBlockPosition()
	println(fmt.Sprintf("[ServerActionThrow] %s(%d) throws from %s.", a.unit.GetName(), a.unit.UnitID(), currentPos.ToString()))

	var validTrajectories [][]mgl32.Vec3
	for _, targetPos := range a.targets {
		trajectory := a.action.GetTrajectory(targetPos)
		if len(trajectory) == 0 {
			mb.AddMessageFor(a.unit.ControlledBy(), game.ActionResponse{Success: false, Message: "Invalid target: No trajectory found"})
			return
		}
		validTrajectories = append(validTrajectories, trajectory)
	}
	var flyers []game.VisualFlightWithImpact
	for _, trajectory := range validTrajectories {

		visitedBlocks, finalWorldPos := a.simulateTrajectory(trajectory)
		finalBlockPos := voxel.PositionToGridInt3(finalWorldPos)
		a.lastAimDirection = finalWorldPos.Sub(a.unit.GetPosition()).Normalize()
		flyers = append(flyers, game.VisualFlightWithImpact{
			Trajectory:    trajectory,
			VisitedBlocks: visitedBlocks,
			FinalWorldPos: finalWorldPos,
			Consequence: game.MessageTargetedEffect{
				Position:    finalBlockPos,
				TurnsToLive: 3, // TODO: effect is hardcoded for now
				Radius:      5,
				Effect:      game.TargetedEffectSmokeCloud,
			},
		})
	}

	ammoCost := uint(len(validTrajectories))
	costOfAPForShot := a.totalAPCost
	a.unit.ConsumeAP(costOfAPForShot)
	a.unit.RemoveItem(a.itemName)
	//a.unit.ConsumeGrenade(ammoCost) // TODO: implement grenade ammo

	lastAimDir := mgl32.Vec3{a.lastAimDirection.X(), 0, a.lastAimDirection.Z()}
	newForward := voxel.DirectionToGridInt3(lastAimDir)
	a.unit.SetForward(newForward)
	a.unit.UpdateMapPosition()

	mb.AddMessageForAll(game.VisualThrow{
		Flyers: flyers,
		//WeaponType:        a.unit.Weapon.Definition.WeaponType,
		AmmoCost:          ammoCost,
		Attacker:          a.unit.UnitID(),
		APCostForAttacker: costOfAPForShot,
		AimDirection:      newForward,
		IsTurnEnding:      a.IsTurnEnding(),
		ItemUsed:          a.itemName,
	})
}

func (a *ServerActionThrow) simulateTrajectory(trajectory []mgl32.Vec3) ([]voxel.Int3, mgl32.Vec3) {

	pathTaken := make(map[voxel.Int3]bool)
	finalPos := trajectory[len(trajectory)-1]

	for i, pos := range trajectory {
		if i == 0 {
			continue
		}
		prev := trajectory[i-1]
		curr := pos

		raycastHitInfo := a.engine.RayCastFreeAim(prev, curr, a.unit)
		for _, onPath := range raycastHitInfo.VisitedBlocks {
			pathTaken[onPath] = true
		}
		if raycastHitInfo.Hit || raycastHitInfo.HitUnit() {
			finalPos = raycastHitInfo.CollisionWorldPosition
			break
		}
	}

	visitedBlocks := make([]voxel.Int3, len(pathTaken))
	i := 0
	for pos := range pathTaken {
		visitedBlocks[i] = pos
		i++
	}

	return visitedBlocks, finalPos
}
