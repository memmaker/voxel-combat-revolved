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
	previousPos := p.GetFootPosition()
	rawVelocity := p.GetVelocity()
	appliedVelocity := rawVelocity.Add(gravity).Mul(float32(deltaTime))
	if appliedVelocity.Len() > 1.0 {
		appliedVelocity = appliedVelocity.Normalize()
	}
	newPos := previousPos.Add(appliedVelocity)

	prevGrid := voxel.ToGridInt3(previousPos)
	newGrid := voxel.ToGridInt3(newPos)

	if prevGrid != newGrid {
		if p.GetVoxelMap().IsSolidBlockAt(newGrid.X, newGrid.Y, newGrid.Z) {
			newPos = previousPos.Add(mgl32.Vec3{appliedVelocity.X(), 0, appliedVelocity.Z()})
			newGrid = voxel.ToGridInt3(newPos)
		}
	}
	p.SetFootPosition(newPos)
}

func (p *Unit) SetState(nextState ActorState) {
	p.state = BehaviorFactory(nextState)
	p.state.Init(p)
}

type HitInfo struct {
	ForceOfImpact mgl32.Vec3
	BodyPart      util.PartName
}

func (p *Unit) HitWithProjectile(forceOfImpact mgl32.Vec3, bodyPart util.PartName, damage int, lethal bool) {
	// needs to be passed to the new state, we do indirectly via the unit
	p.hitInfo = HitInfo{
		ForceOfImpact: forceOfImpact,
		BodyPart:      bodyPart,
	}

	p.ApplyDamage(damage, bodyPart)

	if lethal {
		p.eventQueue = append(p.eventQueue, EventLethalHit)
	}
}

func (p *Unit) Draw(shader *glhf.Shader) {
	p.UnitInstance.GetModel().Draw(shader)
}

func (p *Unit) SetFootPosition(position mgl32.Vec3) {
	p.UnitInstance.GetModel().SetPosition(position)
}
func (p *Unit) GetFootPosition() mgl32.Vec3 {
	return p.UnitInstance.GetModel().GetPosition()
}

func (p *Unit) GetTransformMatrix() mgl32.Mat4 {
	return p.UnitInstance.GetModel().RootNode.GlobalMatrix()
}
func (p *Unit) HasReachedWaypoint() bool {
	return p.GetFootPosition().Sub(p.GetWaypoint().ToBlockCenterVec3()).Len() < 0.05
}

func (p *Unit) SetPath(path []voxel.Int3) {
	p.currentPath = path
	p.currentWaypoint = 0
	p.eventQueue = append(p.eventQueue, EventNewPath)
	println(fmt.Sprintf("[Unit] %s(%d) SetPath %v", p.GetName(), p.UnitID(), path))
}
func (p *Unit) GetWaypoint() voxel.Int3 {
	return p.currentPath[p.currentWaypoint]
}
func (p *Unit) MoveTowardsWaypoint() {
	newVelocity := p.GetWaypoint().ToBlockCenterVec3().Sub(p.GetFootPosition()).Normalize().Mul(p.animationSpeed)
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
	direction := p.GetWaypoint().Sub(voxel.ToGridInt3(p.GetFootPosition()))
	p.SetForward(direction)
}
func (p *Unit) turnToDirectionForDeathAnimation(direction mgl32.Vec3) {
	angle := util.DirectionToAngleVec(direction)
	p.UnitInstance.GetModel().SetYRotationAngle(angle)
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
	return p.currentPath[p.currentWaypoint].Y == voxel.ToGridInt3(p.GetFootPosition()).Y+1
}

func (p *Unit) IsCurrentWaypointADrop() bool {
	return p.currentPath[p.currentWaypoint].Y < voxel.ToGridInt3(p.GetFootPosition()).Y
}

func (p *Unit) NextWaypoint() {
	p.currentWaypoint++
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

func (p *Unit) StartIdleAnimationLoop() {
	ownPos := p.GetBlockPosition()
	animation, front := game.GetIdleAnimationAndForwardVector(p.GetVoxelMap(), ownPos, p.GetForward())
	println(fmt.Sprintf("[Unit] %s(%d) StartIdleAnimationLoop %s -> %v", p.GetName(), p.UnitID(), animation.Str(), front))
	p.UnitInstance.GetModel().SetAnimationLoop(animation.Str(), 1.0)
	p.SetForward(front)
	//println(p.model.GetAnimationDebugString())
}

func (p *Unit) GetLastWaypoint() voxel.Int3 {
	return p.currentPath[len(p.currentPath)-1]
}

func (p *Unit) IsIdle() bool {
	return p.state.GetName() == ActorStateIdle
}

func (p *Unit) FreezeIdleAnimation() {
	p.StartIdleAnimationLoop()
	p.UnitInstance.GetModel().ResetAnimations()
}

func (p *Unit) SetServerInstance(unit *game.UnitInstance) {
	oldModel := p.UnitInstance.GetModel()
	oldVoxelMap := p.GetVoxelMap()

	unit.SetModel(oldModel)
	unit.SetVoxelMap(oldVoxelMap)

	p.UnitInstance = unit
	p.UpdateMapAndModelPosition()
}

func NewClientUnit(instance *game.UnitInstance) *Unit {
	// load model of unit
	a := &Unit{
		UnitInstance:    instance,
		animationSpeed:  4,
		currentWaypoint: -1,
		transition:      ActorTransitionTable, // one for all
	}
	a.UpdateMapAndModelPosition()
	a.SetState(ActorStateIdle)
	return a
}
