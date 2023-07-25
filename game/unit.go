package game

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/glhf"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
	"math"
)

type Unit struct {
	extents         mgl32.Vec3
	model           *util.CompoundMesh
	velocity        mgl32.Vec3
	currentWaypoint mgl32.Vec3
	speed           float32
	waypointTimer   float64
	state           Behavior
	transition      *TransitionTable
	hitInfo         HitInfo
	removeActor     bool
	Height          uint8
	eventQueue      []TransitionEvent
}

func (p *Unit) GetOccupiedBlockOffsets() []voxel.Int3 {
	return []voxel.Int3{
		{0, 0, 0},
		{0, 1, 0},
	}
}

func (p *Unit) SetVelocity(newVelocity mgl32.Vec3) {
	if newVelocity.Len() > p.speed {
		newVelocity = newVelocity.Normalize().Mul(p.speed)
	}
	p.velocity = newVelocity
}

func (p *Unit) GetName() string {
	return "Unit"
}

func (p *Unit) IsProjectile() bool {
	return false
}

func (p *Unit) GetExtents() mgl32.Vec3 {
	return p.extents
}

func (p *Unit) GetPosition() mgl32.Vec3 {
	return p.model.GetPosition().Add(mgl32.Vec3{0, p.extents.Y() / 2, 0})
}

func (p *Unit) SetPosition(pos mgl32.Vec3) {
	p.model.SetPosition(pos.Sub(mgl32.Vec3{0, p.extents.Y() / 2, 0}))
}

func (p *Unit) GetVelocity() mgl32.Vec3 {
	return p.velocity
}

func (p *Unit) GetAABB() util.AABB {
	center := p.model.GetPosition().Add(mgl32.Vec3{0, p.extents.Y() / 2, 0})
	return util.NewAABB(center, p.extents)
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
	currentState := p.state.GetName()
	currentEvent := p.state.Execute(deltaTime)
	if p.transition.Exists(currentState, currentEvent) {
		nextState := p.transition.GetNextState(currentState, currentEvent)
		p.SetState(nextState)
	}
}

func (p *Unit) SetState(nextState ActorState) {
	p.state = BehaviorFactory(nextState)
	p.state.Init(p)
}

type HitInfo struct {
	ForceOfImpact mgl32.Vec3
	BodyPart      util.Collider
}

func (p *Unit) HitWithProjectile(projectile util.CollidingObject, bodyPart util.Collider) {
	event := EventHit
	// needs to be passed to the new state, we do indirectly via the actor
	forceOfImpact := projectile.GetVelocity()
	p.hitInfo = HitInfo{
		ForceOfImpact: forceOfImpact,
		BodyPart:      bodyPart,
	}

	if p.transition.Exists(p.state.GetName(), event) {
		nextState := p.transition.GetNextState(p.state.GetName(), event)
		p.SetState(nextState)
	}
}

func (p *Unit) Draw(shader *glhf.Shader, camPosition mgl32.Vec3) {
	p.model.Draw(shader, camPosition)
}

func (p *Unit) GetColliders() []util.Collider {
	return p.model.RootNode.GetColliders()
}
func (p *Unit) SetFootPosition(position mgl32.Vec3) {
	p.model.SetPosition(position)
}
func (p *Unit) GetFootPosition() mgl32.Vec3 {
	return p.model.GetPosition()
}
func (p *Unit) GetEyePosition() mgl32.Vec3 {
	return p.model.GetPosition().Add(mgl32.Vec3{0, p.extents.Y() * (7.0 / 8.0), 0})
}
func (p *Unit) GetTransformMatrix() mgl32.Mat4 {
	return p.model.RootNode.GlobalMatrix()
}
func (p *Unit) SetCurrentWaypoint(waypoint mgl32.Vec3) {
	p.currentWaypoint = waypoint
}
func (p *Unit) IsNearWaypoint() bool {
	return p.GetPosition().Sub(p.currentWaypoint).Len() < 0.5
}

func (p *Unit) SetWaypoint(targetPos mgl32.Vec3) {
	p.currentWaypoint = targetPos
	p.eventQueue = append(p.eventQueue, EventNewWaypoint)
}

func (p *Unit) MoveTowardsWaypoint() {
	newVelocity := p.currentWaypoint.Sub(p.GetPosition()).Normalize().Mul(p.speed)
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
	direction := p.currentWaypoint.Sub(p.GetPosition()).Normalize()
	p.turnToDirection(direction)
}

func (p *Unit) turnToDirection(direction mgl32.Vec3) {
	angle := float32(math.Atan2(float64(direction.X()), float64(direction.Z()))) + math.Pi
	p.model.SetYRotationAngle(angle)
}

func (p *Unit) GetFront() mgl32.Vec3 {
	return p.model.GetFront()
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


func NewUnit(model *util.CompoundMesh, pos mgl32.Vec3) *Unit {
	a := &Unit{
		model:           model,
		extents:         mgl32.Vec3{0.98, 1.98, 0.98},
		speed:           4,
		currentWaypoint: mgl32.Vec3{4, 1.75, 1},
		transition:      ActorTransitionTable, // one for all
	}
	a.SetState(ActorStateIdle)
	a.SetFootPosition(pos)
	return a
}
