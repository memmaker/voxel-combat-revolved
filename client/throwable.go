package client

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/glhf"
	"github.com/memmaker/battleground/engine/util"
)

type Throwable struct {
	*util.Transform

	shader *glhf.Shader

	path []mgl32.Vec3

	onArrival func()
	isDead    bool
	model     *util.CompoundMesh
	lerper    *util.WaypointLerper
}

func (t *Throwable) GetName() string {
	return "Throwable"
}
func (t *Throwable) IsDead() bool {
	return t.isDead
}

func NewThrowable(shader *glhf.Shader, model *util.CompoundMesh, path []mgl32.Vec3) *Throwable {
	p := &Throwable{
		Transform: util.NewTransform(path[0], mgl32.QuatIdent(), mgl32.Vec3{0.5, 0.5, 0.5}),
		path:      path,
		shader:    shader,
		model:     model,
	}
	p.lerper = util.NewWaypointLerper(p, path, 0.5)
	//p.Transform.SetLookAt(path[1].Add(velocity.Normalize().Mul(10)), up)
	return p
}
func (t *Throwable) Draw() {
	t.model.RootNode.SetParent(t)
	t.shader.SetUniformAttr(ShaderModelMatrix, t.GetTransformMatrix())
	t.model.Draw(t.shader, ShaderModelMatrix)
}
func (t *Throwable) Update(delta float64) {
	if !t.lerper.IsDone() {
		t.lerper.Update(delta)
		return
	} else if t.onArrival != nil && !t.isDead {
		t.isDead = true
		t.onArrival()
	}
}
func (t *Throwable) SetOnArrival(arrival func()) {
	t.onArrival = arrival
}
