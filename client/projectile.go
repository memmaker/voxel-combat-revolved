package client

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/glhf"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/game"
)

type Projectile struct {
	*util.Transform

	velocity mgl32.Vec3
	shader   *glhf.Shader

	destination mgl32.Vec3

	onArrival func()
	isDead    bool
	startPos  mgl32.Vec3
	model     *util.CompoundMesh
}

func (p *Projectile) GetParticleProps() glhf.ParticleProperties {
	return glhf.ParticleProperties{
		Origin:               p.GetPosition().Add(p.GetForward().Normalize().Mul(-0.1)),
		PositionVariation:    mgl32.Vec3{0.04, 0.04, 0.04},
		VelocityFromPosition: func(origin, pos mgl32.Vec3) mgl32.Vec3 { return mgl32.Vec3{} },
		ColorBegin:           mgl32.Vec3{0.9, 0.9, 0.9},
		ColorEnd:             mgl32.Vec3{0.8, 0.8, 0.8},
		ColorVariation:       0,
		SizeBegin:            0.03,
		SizeEnd:              0.01,
		SizeVariation:        0,
		Lifetime:             2,
		MaxDistance:          0,
		SpreadLifetime:       0,
	}
}

func (p *Projectile) GetName() string {
	return "Projectile"
}

func (p *Projectile) IsProjectile() bool {
	return true
}

func (p *Projectile) GetVelocity() mgl32.Vec3 {
	return p.velocity
}

func (p *Projectile) IsDead() bool {
	return p.isDead
}

func NewProjectile(shader *glhf.Shader, model *util.CompoundMesh, pos, velocity mgl32.Vec3) *Projectile {
	p := &Projectile{
		Transform: util.NewTransform(pos, mgl32.QuatIdent(), mgl32.Vec3{0.5, 0.5, 0.5}),
		velocity:  velocity,
		startPos:  pos,
		shader:    shader,
		model:     model,
	}
	forward := velocity.Normalize()
	right := forward.Cross(mgl32.Vec3{0, 1, 0})
	up := right.Cross(forward)
	p.Transform.SetLookAt(pos.Add(velocity.Normalize().Mul(10)), up)
	return p
}
func (p *Projectile) Draw() {
	p.model.RootNode.SetParent(p)
	p.shader.SetUniformAttr(ShaderModelMatrix, p.GetTransformMatrix())
	p.model.Draw(p.shader, ShaderModelMatrix)
}
func (p *Projectile) Update(delta float64) {
	oldPos := p.GetPosition()
	newPos := oldPos.Add(p.velocity.Mul(float32(delta)))
	p.SetPosition(newPos)
	arrived := newPos.Sub(p.destination).Len() < game.PositionalTolerance
	traveled := newPos.Sub(p.startPos).Len()
	distance := p.startPos.Sub(p.destination).Len()
	tooFar := traveled > distance
	if (arrived || tooFar) && !p.isDead {
		p.isDead = true
		if p.onArrival != nil {
			p.onArrival()
		}
	}
}
func (p *Projectile) SetDestination(destination mgl32.Vec3) {
	p.destination = destination
}

func (p *Projectile) SetOnArrival(arrival func()) {
	p.onArrival = arrival
}
