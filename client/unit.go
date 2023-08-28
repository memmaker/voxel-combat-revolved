package client

import (
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/glhf"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
	"github.com/memmaker/battleground/game"
)

type Unit struct {
	*game.UnitInstance
	velocity         mgl32.Vec3
	currentWaypoint  int
	animationSpeed   float32
	waypointTimer    float64
	state            AnimationState
	transition       *TransitionTable
	hitInfo          HitInfo
	removeActor      bool
	eventQueue       []TransitionEvent
	currentPath      []voxel.Int3
	controlledByUser bool
}

func (p *Unit) SetVelocity(newVelocity mgl32.Vec3) {
	if newVelocity.Len() > p.animationSpeed {
		newVelocity = newVelocity.Normalize().Mul(p.animationSpeed)
	}
	p.velocity = newVelocity
}
func (p *Unit) IsProjectile() bool {
	return false
}

func (p *Unit) GetVelocity() mgl32.Vec3 {
	return p.velocity
}

func (p *Unit) GetIdleEvents() TransitionEvent {
	//p.SetWaypoint()
	if len(p.eventQueue) > 0 {
		nextEvent := p.eventQueue[0]
		p.eventQueue = p.eventQueue[1:]
		return nextEvent
	}
	return EventNone
}
func (p *Unit) Update(deltaTime float64) {
	if p.IsDead() {
		return
	}

	p.applyVelocity(deltaTime)

	p.GetModel().UpdateAnimations(deltaTime)

	currentState := p.state.GetName()
	currentEvent := p.state.Execute(deltaTime)

	if p.transition.Exists(currentState, currentEvent) {
		nextState := p.transition.GetNextState(currentState, currentEvent)
		//println(fmt.Sprintf("[%s] Transition from %s to %s", p.GetName(), currentState.ToString(), nextState.ToString()))
		p.SetState(nextState)
	}
}

func (p *Unit) applyVelocity(deltaTime float64) {
	gravity := mgl32.Vec3{0, -9.8, 0}
	previousPos := p.GetPosition()
	rawVelocity := p.GetVelocity()
	appliedVelocity := rawVelocity.Add(gravity).Mul(float32(deltaTime))
	if appliedVelocity.Len() > 1.0 {
		appliedVelocity = appliedVelocity.Normalize()
	}
	newPos := previousPos.Add(appliedVelocity)

	prevGrid := voxel.PositionToGridInt3(previousPos)
	newGrid := voxel.PositionToGridInt3(newPos)

	if newGrid.IsBelow(prevGrid) && p.GetVoxelMap().IsSolidBlockAt(newGrid.X, newGrid.Y, newGrid.Z) {
		appliedVelocity = mgl32.Vec3{appliedVelocity.X(), 0, appliedVelocity.Z()}
		newPos = previousPos.Add(appliedVelocity)
		newPos = mgl32.Vec3{newPos.X(), float32(prevGrid.Y), newPos.Z()}
	}

	if previousPos != newPos {
		p.SetPosition(newPos)
	}
}

func (p *Unit) SetState(nextState ActorState) {
	p.state = BehaviorFactory(nextState)
	p.state.Init(p)
}

type HitInfo struct {
	ForceOfImpact mgl32.Vec3
	BodyPart      util.DamageZone
}

func (p *Unit) PlayDeathAnimation(forceOfImpact mgl32.Vec3, bodyPart util.DamageZone) {
	// needs to be passed to the new state, we do indirectly via the unit
	p.hitInfo = HitInfo{
		ForceOfImpact: forceOfImpact,
		BodyPart:      bodyPart,
	}
	p.eventQueue = append(p.eventQueue, EventLethalHit)
}

func (p *Unit) PlayHitAnimation(forceOfImpact mgl32.Vec3, bodyPart util.DamageZone) {
	// needs to be passed to the new state, we do indirectly via the unit
	p.UnitInstance.Transform.SetForward2D(forceOfImpact.Mul(-1.0).Normalize()) // ok, because this is temporary
	p.GetModel().SetAnimation(game.AnimationHit.Str(), 1.0)
	// TODO: add the actual hit animation
}

func (p *Unit) Draw(shader *glhf.Shader) {
	p.UnitInstance.GetModel().Draw(shader, ShaderModelMatrix)
}

func (p *Unit) GetTransformMatrix() mgl32.Mat4 {
	return p.UnitInstance.GetModel().RootNode.GetTransformMatrix()
}
func (p *Unit) HasReachedWaypoint() bool {
	footPosition := p.GetPosition()
	waypoint := p.GetWaypoint().ToBlockCenterVec3()
	dist := footPosition.Sub(waypoint).Len()

	reached := dist < PositionalTolerance
	return reached
}

func (p *Unit) SetPath(path []voxel.Int3) {
	p.currentPath = path
	p.currentWaypoint = 0
	p.eventQueue = append(p.eventQueue, EventNewPath)
	//println(fmt.Sprintf("[Unit] %s(%d) SetPath %v", p.GetName(), p.UnitID(), path))
}
func (p *Unit) GetWaypoint() voxel.Int3 {
	return p.currentPath[p.currentWaypoint]
}
func (p *Unit) MoveTowardsWaypoint() {
	newVelocity := p.GetWaypoint().ToBlockCenterVec3().Sub(p.GetPosition()).Normalize().Mul(p.animationSpeed)
	p.SetVelocity(newVelocity)
}

func (p *Unit) shouldContinue(deltaTime float64) bool {
	if p.waypointTimer < 6.0 {
		p.waypointTimer += deltaTime
		return false
	}
	p.waypointTimer = 0
	return true
}

func (p *Unit) TurnTowardsWaypoint() {
	d := p.GetWaypoint().Sub(p.GetBlockPosition())
	direction := voxel.Int3{X: d.X, Y: 0, Z: d.Z}
	println(fmt.Sprintf("[Unit] %s(%d) TurnTowardsWaypoint %v", p.GetName(), p.UnitID(), direction))
	p.SetForward(direction)
}
func (p *Unit) turnToDirectionForDeathAnimation(direction mgl32.Vec3) {
	p.UnitInstance.Transform.SetForward2D(direction) // ok, because this is temporary
}

func (p *Unit) IsDead() bool {
	return p.state.GetName() == ActorStateDead
}

func (p *Unit) IsDying() bool {
	return p.state.GetName() == ActorStateDying
}

func (p *Unit) ShouldBeRemoved() bool {
	return p.removeActor
}

func (p *Unit) IsLastWaypoint() bool {
	return p.currentWaypoint == len(p.currentPath)-1
}

func (p *Unit) IsCurrentWaypointAClimb() bool {
	return p.currentPath[p.currentWaypoint].Y == p.GetBlockPosition().Y+1
}

func (p *Unit) IsCurrentWaypointADrop() bool {
	return p.currentPath[p.currentWaypoint].Y < p.GetBlockPosition().Y
}

func (p *Unit) NextWaypoint() {
	p.currentWaypoint++
	//println(fmt.Sprintf("[Unit] %s(%d) NextWaypoint, now: %s", p.GetName(), p.UnitID(), p.GetWaypoint().ToString()))
}
func (p *Unit) IsUserControlled() bool {
	return p.controlledByUser
}

func (p *Unit) SetUserControlled() {
	p.controlledByUser = true
}

func (p *Unit) Description() string {
	return fmt.Sprintf("Unit: %s", p.GetName())
}

func (p *Unit) GetLastWaypoint() voxel.Int3 {
	return p.currentPath[len(p.currentPath)-1]
}

func (p *Unit) IsIdle() bool {
	return p.state.GetName() == ActorStateIdle
}

func (p *Unit) FreezeStanceAnimation() {
	p.UnitInstance.GetModel().SetAnimationPose(p.GetStance().GetAnimation().Str())
}

func (p *Unit) SetServerInstance(unit *game.UnitInstance) {
	oldModel := p.UnitInstance.GetModel()
	oldVoxelMap := p.GetVoxelMap()

	unit.SetModel(oldModel)
	unit.SetVoxelMap(oldVoxelMap)

	p.UnitInstance = unit
	p.AutoSetStanceAndForward()
	p.StartStanceAnimation()
}

func (p *Unit) IsInTheAir() bool {
	posBelow := p.GetPosition().Sub(mgl32.Vec3{0, 1, 0})
	currentBlock := p.GetBlockPosition()
	blockPosBelow := voxel.PositionToGridInt3(posBelow)
	footHeight := p.GetPosition().Y()

	if !p.GetVoxelMap().IsSolidBlockAt(blockPosBelow.X, blockPosBelow.Y, blockPosBelow.Z) {
		return true
	}
	return footHeight > (float32(currentBlock.Y) + PositionalTolerance)
}

func (p *Unit) GetForward() mgl32.Vec3 {
	return p.UnitInstance.Transform.GetForward()
}

func (p *Unit) IsAtLocation(destination voxel.Int3) bool {
	targetPos := destination.ToBlockCenterVec3()
	currentPos := p.GetPosition()
	dist := targetPos.Sub(currentPos).Len()
	return dist < PositionalTolerance
}

func NewClientUnit(instance *game.UnitInstance) *Unit {
	// load model of unit
	a := &Unit{
		UnitInstance:    instance,
		animationSpeed:  4,
		currentWaypoint: -1,
		transition:      ActorTransitionTable, // one for all
	}
	a.SetState(ActorStateIdle)
	return a
}
