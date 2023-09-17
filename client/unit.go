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
	velocity           mgl32.Vec3
	currentWaypoint    int
	animationSpeed     float32
	waypointTimer      float64
	state              AnimationState
	transition         *TransitionTable
	hitInfo            HitInfo
	removeActor        bool
	eventQueue         []TransitionEvent
	currentPath        []voxel.Int3
	controlledByUser   bool
	clientOnlyRotation mgl32.Quat
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

func (p *Unit) GetNextEvent(currentState AnimationStateName) TransitionEvent {
	if len(p.eventQueue) > 0 {
		nextEvent := p.eventQueue[0]
		p.eventQueue = p.eventQueue[1:]
		if !p.transition.Exists(currentState, nextEvent) {
			// only advance the state machine from the current state, if the transition is defined
			// otherwise, we just re-emit the event
			p.eventQueue = append(p.eventQueue, nextEvent)
			return EventNone
		}
		util.LogGreen(fmt.Sprintf("[Unit] %s(%d) Event retrieved from Queue %s", p.GetName(), p.UnitID(), nextEvent.ToString()))
		return nextEvent
	}
	return EventNone
}
func (p *Unit) EmitEvent(stateEvent TransitionEvent) {
	if stateEvent != EventNone {
		p.eventQueue = append(p.eventQueue, stateEvent)
	}
}

func (p *Unit) Update(deltaTime float64) {
	if p.IsDead() {
		return
	}

	p.applyVelocity(deltaTime)

	p.GetModel().UpdateAnimations(deltaTime)

	currentState := p.state.GetName()

	stateEvent := p.state.Execute(deltaTime)

	p.EmitEvent(stateEvent)

	currentEvent := p.GetNextEvent(currentState)
	// HMM2 now the problem is the movement animation not finishing, because we switched to idle before it could register

	if p.transition.Exists(currentState, currentEvent) {
		nextState := p.transition.GetNextState(currentState, currentEvent)
		util.LogGlobalUnitDebug(fmt.Sprintf("[%s] Received %s -> Transition from %s to %s", p.GetName(), currentEvent.ToString(), currentState.ToString(), nextState.ToString()))
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

func (p *Unit) SetState(nextState AnimationStateName) {
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
	p.hitInfo = HitInfo{
		ForceOfImpact: forceOfImpact,
		BodyPart:      bodyPart,
	}
	p.eventQueue = append(p.eventQueue, EventHit)
}

func (p *Unit) Draw(shader *glhf.Shader) {
	p.UnitInstance.GetModel().Draw(shader, ShaderModelMatrix)
}

func (p *Unit) GetTransformMatrix() mgl32.Mat4 {
	unitTransform := p.UnitInstance.Transform
	trans := unitTransform.GetTranslationMatrix()
	transInv := trans.Inv()
	// undo translation, apply rotation, reapply translation
	return trans.Mul4(p.clientOnlyRotation.Mat4()).Mul4(transInv)
}

func (p *Unit) SetPath(path []voxel.Int3) {
	p.currentPath = path
	p.currentWaypoint = 0
	p.eventQueue = append(p.eventQueue, EventNewPath)
	util.LogGreen(fmt.Sprintf("[Unit] %s(%d) SetPath %v", p.GetName(), p.UnitID(), path))
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
	//println(fmt.Sprintf("[Unit] %s(%d) TurnTowardsWaypoint %v", p.GetName(), p.Attacker(), direction))
	p.turnToDiagonalDirectionForAnimation(direction)
}
func (p *Unit) turnToDirectionForAnimation(direction mgl32.Vec3) {
	currentForwards := p.GetForward()
	direction = mgl32.Vec3{direction.X(), 0, direction.Z()}
	p.clientOnlyRotation = mgl32.QuatBetweenVectors(currentForwards, direction)
}

func (p *Unit) setClientOnlyRotation(direction mgl32.Quat) {
	p.clientOnlyRotation = direction
}
func (p *Unit) resetClientOnlyRotation() {
	p.clientOnlyRotation = mgl32.QuatIdent()
}
func (p *Unit) turnToDiagonalDirectionForAnimation(direction voxel.Int3) {
	p.UnitInstance.Transform.SetForward2DDiagonal(direction) // ok, because this is temporary
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
	//println(fmt.Sprintf("[Unit] %s(%d) NextWaypoint, now: %s", p.GetName(), p.Attacker(), p.GetWaypoint().ToString()))
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

func (p *Unit) GetLastDirection() voxel.Int3 {
	if len(p.currentPath) == 0 {
		return p.GetForward2DCardinal()
	}
	last := p.currentPath[len(p.currentPath)-1]
	prev := p.GetBlockPosition()
	if len(p.currentPath) > 1 {
		prev = p.currentPath[len(p.currentPath)-2]
	}

	return voxel.Int3{X: last.X - prev.X, Y: last.Y - prev.Y, Z: last.Z - prev.Z}
}

func (p *Unit) IsIdle() bool {
	return p.state.GetName() == ActorStateIdle
}

func (p *Unit) FreezeStanceAnimation() {
	p.UnitInstance.GetModel().SetAnimationPose(p.GetStance().GetAnimation().Str())
}

func (p *Unit) IsInTheAir() bool {
	posBelow := p.GetPosition().Sub(mgl32.Vec3{0, 1, 0})
	currentBlock := p.GetBlockPosition()
	blockPosBelow := voxel.PositionToGridInt3(posBelow)
	footHeight := p.GetPosition().Y()

	if !p.GetVoxelMap().IsSolidBlockAt(blockPosBelow.X, blockPosBelow.Y, blockPosBelow.Z) {
		return true
	}
	return footHeight > (float32(currentBlock.Y) + game.PositionalTolerance)
}

func (p *Unit) GetForward() mgl32.Vec3 {
	return p.UnitInstance.Transform.GetForward()
}

func (p *Unit) IsAtLocation(destination voxel.Int3) bool {
	targetPos := destination.ToBlockCenterVec3()
	currentPos := p.GetPosition()
	dist := targetPos.Sub(currentPos).Len()
	return dist < game.PositionalTolerance
}

func (p *Unit) HasReachedWaypoint() bool {
	return p.IsAtLocation(p.GetWaypoint())
}

func (p *Unit) GetClientOnlyRotation() mgl32.Quat {
	return p.clientOnlyRotation
}
func NewClientUnit(instance *game.UnitInstance) *Unit {
	// load model of unit
	a := &Unit{
		animationSpeed:  4,
		currentWaypoint: -1,
		transition:      ActorTransitionTable, // one for all
		clientOnlyRotation: mgl32.QuatIdent(),
	}
	a.SetServerInstance(instance)
	a.SetState(ActorStateIdle)
	return a
}
func (p *Unit) UpdateFromServerInstance(unit *game.UnitInstance) {
	oldModel := p.UnitInstance.GetModel()
	oldVoxelMap := p.GetVoxelMap()

	unit.SetModel(oldModel)
	unit.SetVoxelMap(oldVoxelMap)

	p.SetServerInstance(unit)
}

func (p *Unit) SetServerInstance(unit *game.UnitInstance) {
	p.UnitInstance = unit
	p.UnitInstance.Transform.SetParent(p)
	p.AutoSetStanceAndForwardAndUpdateMap()
	p.StartStanceAnimation()
}

