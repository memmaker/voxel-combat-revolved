package client

import (
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/glhf"
	"github.com/memmaker/battleground/engine/util"
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
	println(fmt.Sprintf("\n>> Projectile spawned at %v", pos))
	p := &Projectile{
		Transform: util.NewTransform(pos, mgl32.QuatIdent(), mgl32.Vec3{0.5, 0.5, 0.5}),
		velocity:  velocity,
		startPos:  pos,
		shader:    shader,
		model:     model,
	}
	p.Transform.SetForward(velocity.Normalize())
	model.RootNode.SetParent(p)
	return p
}
func (p *Projectile) Draw() {
	p.shader.SetUniformAttr(2, p.GetTransformMatrix())
	p.model.DrawWithoutTransform(p.shader)
}
func (p *Projectile) Update(delta float64) {
	oldPos := p.GetPosition()
	newPos := oldPos.Add(p.velocity.Mul(float32(delta)))
	p.SetPosition(newPos)
	arrived := newPos.Sub(p.destination).Len() < PositionalTolerance
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
