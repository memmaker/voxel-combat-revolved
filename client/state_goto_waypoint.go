package client

import (
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/voxel"
	"github.com/memmaker/battleground/game"
	"github.com/solarlune/gocoro"
)

type UnitGotoWaypointBehavior struct {
	unit             *Unit
	yOffset          int32
	coroutine        gocoro.Coroutine
}

func should(err error) {
	if err != nil {
		println(fmt.Sprintf("[Script] Error: %v", err))
	}
}
func (a *UnitGotoWaypointBehavior) GetUnitMovementScript(exe *gocoro.Execution) {
	// we start by setting the new map position, so that the user can go on
	// with selecting the next unit and has the correct map state

	a.unit.ForceMapPosition(a.unit.GetLastWaypoint(), a.unit.GetLastDirection())

	for {
		// do we need to start some special animation and thus wait for its completion?
		if a.startAndWaitForAnimation() {
			should(exe.YieldFunc(a.unit.GetModel().IsHoldingAnimation)) // this can fail, becaus the unit might have switched to an idle loop, without this triggering
			// reposition after climb & drop animation
			wp := a.unit.GetWaypoint()
			fp := a.unit.GetBlockPosition()
			resolvedPosition := voxel.Int3{X: wp.X, Y: fp.Y + a.yOffset, Z: wp.Z}
			a.snapToPosition(resolvedPosition)
		}
		println(fmt.Sprintf("[UnitGotoWaypointBehavior] Start movement from %v to %v", a.unit.GetBlockPosition(), a.unit.GetWaypoint()))

		// move until we reach a waypoint
		should(exe.YieldFunc(func() bool {
			a.unit.MoveTowardsWaypoint()
			return a.unit.HasReachedWaypoint()
		}))

		println(fmt.Sprintf("[UnitGotoWaypointBehavior] Reached waypoint %v", a.unit.GetWaypoint()))
		// we reached a waypoint
		if a.unit.IsLastWaypoint() {
			a.snapToLastPosition(a.unit.GetWaypoint())
			a.unit.SetVelocity(mgl32.Vec3{0, 0, 0})
			break // end loop
		} else { // not last waypoint
			a.snapToPosition(a.unit.GetWaypoint())
			a.unit.NextWaypoint()
		}
	}
}

func (a *UnitGotoWaypointBehavior) Execute(deltaTime float64) TransitionEvent {
	if a.coroutine.Running() {
		a.coroutine.Update()
		return EventNone
	} else {
		return EventLastWaypointReached
	}
}

func (a *UnitGotoWaypointBehavior) snapToPosition(blockPosition voxel.Int3) {
	a.unit.GetModel().SetAnimationLoop(game.AnimationWeaponWalk.Str(), 1.0)
	a.unit.SetBlockPosition(blockPosition)
}

func (a *UnitGotoWaypointBehavior) snapToLastPosition(blockPosition voxel.Int3) {
	//println(fmt.Sprintf("[UnitGotoWaypointBehavior] Snapping to blockPosition: %v", blockPosition))
	a.unit.SetBlockPosition(blockPosition)
	a.unit.AutoSetStanceAndForwardAndUpdateMap()
	//println(fmt.Sprintf("[UnitGotoWaypointBehavior] New block position: %v, New FootPosition: %v", a.unit.GetBlockPosition(), a.unit.GetPosition()))
}

func (a *UnitGotoWaypointBehavior) startAndWaitForAnimation() bool {
	a.unit.TurnTowardsWaypoint()
	//println(fmt.Sprintf("[UnitGotoWaypointBehavior] Start waypoint animation for: %v -> %v", a.unit.GetBlockPosition(), a.unit.GetWaypoint()))
	if a.unit.IsCurrentWaypointAClimb() {
		a.unit.SetVelocity(mgl32.Vec3{0, 0, 0})
		a.unit.GetModel().SetAnimation(game.AnimationClimb.Str(), 1.0)
		a.yOffset = 1
		return true
	} else if a.unit.IsCurrentWaypointADrop() {
		a.unit.SetVelocity(mgl32.Vec3{0, 0, 0})
		a.unit.GetModel().SetAnimation(game.AnimationDrop.Str(), 1.0)
		a.yOffset = -1
		return true
	} else {
		a.unit.GetModel().SetAnimationLoop(game.AnimationWeaponWalk.Str(), 1.0)
	}
	return false
}

func (a *UnitGotoWaypointBehavior) GetName() ActorState {
	return UnitGotoWaypoint
}

func (a *UnitGotoWaypointBehavior) Init(unit *Unit) {
	a.unit = unit
	a.coroutine = gocoro.NewCoroutine()
	should(a.coroutine.Run(a.GetUnitMovementScript))
}
