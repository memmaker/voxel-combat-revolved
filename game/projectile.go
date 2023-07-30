package game

import (
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/glhf"
	"github.com/memmaker/battleground/engine/util"
)

type Projectile struct {
	extents    mgl32.Vec3
	velocity   mgl32.Vec3
	vertexData *glhf.VertexSlice[glhf.GlFloat]
	shader     *glhf.Shader
	texture    *glhf.Texture

	// transformation
	position     mgl32.Vec3
	quatRotation mgl32.Quat
	scale        mgl32.Vec3
	meshCollider *util.MeshCollider
}

func (p *Projectile) HitWithProjectile(projectile util.CollidingObject, part util.Collider) {

}

func (p *Projectile) GetColliders() []util.Collider {
	return []util.Collider{p.meshCollider}
}

func (p *Projectile) GetName() string {
	return "Projectile"
}

func (p *Projectile) IsProjectile() bool {
	return true
}

func (p *Projectile) GetExtents() mgl32.Vec3 {
	return p.extents
}

func (p *Projectile) GetCollider() util.Collider {
	return p.meshCollider.SetVelocityForSweep(p.velocity)
}

func (p *Projectile) GetPosition() mgl32.Vec3 {
	return p.position
}

func (p *Projectile) GetVelocity() mgl32.Vec3 {
	return p.velocity
}

func (p *Projectile) IsDead() bool {
	return false
}

func (p *Projectile) SetPosition(pos mgl32.Vec3) {
	p.position = pos
}

func NewProjectile(shader *glhf.Shader, texture *glhf.Texture, pos mgl32.Vec3) *Projectile {
	vd := glhf.MakeVertexSlice(shader, 36, 36)
	// we want to create small lengthy rectangle
	//pos(3), norm(3), uv(2)
	length := glhf.GlFloat(0.05) // 5 cm long
	height := glhf.GlFloat(0.02) // 2 cm tall & wide
	halfLength := length / 2
	halfHeight := height / 2
	vd.Begin()
	// we'll use halfLength for the depth along the Z axis
	// and halfHeight for the width along the X and Y axes
	rawVertexData := []glhf.GlFloat{
		// front, first triangle
		-halfHeight, -halfHeight, -halfLength, 0, 0, 1, 0, 0,
		halfHeight, -halfHeight, -halfLength, 0, 0, 1, 1, 0,
		halfHeight, halfHeight, -halfLength, 0, 0, 1, 1, 1,
		// front, second triangle
		-halfHeight, -halfHeight, -halfLength, 0, 0, 1, 0, 0,
		halfHeight, halfHeight, -halfLength, 0, 0, 1, 1, 1,
		-halfHeight, halfHeight, -halfLength, 0, 0, 1, 0, 1,

		// back, first triangle
		-halfHeight, -halfHeight, halfLength, 0, 0, -1, 0, 0,
		halfHeight, -halfHeight, halfLength, 0, 0, -1, 1, 0,
		halfHeight, halfHeight, halfLength, 0, 0, -1, 1, 1,
		// back, second triangle
		-halfHeight, -halfHeight, halfLength, 0, 0, -1, 0, 0,
		halfHeight, halfHeight, halfLength, 0, 0, -1, 1, 1,
		-halfHeight, halfHeight, halfLength, 0, 0, -1, 0, 1,

		// left, first triangle
		-halfHeight, -halfHeight, -halfLength, -1, 0, 0, 0, 0,
		-halfHeight, -halfHeight, halfLength, -1, 0, 0, 1, 0,
		-halfHeight, halfHeight, halfLength, -1, 0, 0, 1, 1,
		// left, second triangle
		-halfHeight, -halfHeight, -halfLength, -1, 0, 0, 0, 0,
		-halfHeight, halfHeight, halfLength, -1, 0, 0, 1, 1,
		-halfHeight, halfHeight, -halfLength, -1, 0, 0, 0, 1,

		// right, first triangle
		halfHeight, -halfHeight, -halfLength, 1, 0, 0, 0, 0,
		halfHeight, -halfHeight, halfLength, 1, 0, 0, 1, 0,
		halfHeight, halfHeight, halfLength, 1, 0, 0, 1, 1,

		// right, second triangle
		halfHeight, -halfHeight, -halfLength, 1, 0, 0, 0, 0,
		halfHeight, halfHeight, halfLength, 1, 0, 0, 1, 1,
		halfHeight, halfHeight, -halfLength, 1, 0, 0, 0, 1,

		// top, first triangle
		-halfHeight, halfHeight, -halfLength, 0, 1, 0, 0, 0,
		halfHeight, halfHeight, -halfLength, 0, 1, 0, 1, 0,
		halfHeight, halfHeight, halfLength, 0, 1, 0, 1, 1,

		// top, second triangle
		-halfHeight, halfHeight, -halfLength, 0, 1, 0, 0, 0,
		halfHeight, halfHeight, halfLength, 0, 1, 0, 1, 1,
		-halfHeight, halfHeight, halfLength, 0, 1, 0, 0, 1,

		// bottom, first triangle
		-halfHeight, -halfHeight, -halfLength, 0, -1, 0, 0, 0,
		halfHeight, -halfHeight, -halfLength, 0, -1, 0, 1, 0,
		halfHeight, -halfHeight, halfLength, 0, -1, 0, 1, 1,

		// bottom, second triangle
		-halfHeight, -halfHeight, -halfLength, 0, -1, 0, 0, 0,
		halfHeight, -halfHeight, halfLength, 0, -1, 0, 1, 1,
		-halfHeight, -halfHeight, halfLength, 0, -1, 0, 0, 1,
	}
	vd.SetVertexData(rawVertexData)
	vd.End()
	println(fmt.Sprintf("\n>> Projectile spawned at %v", pos))
	p := &Projectile{
		position:     pos,
		extents:      mgl32.Vec3{float32(length), float32(length), float32(length)},
		shader:       shader,
		vertexData:   vd,
		texture:      texture,
		quatRotation: mgl32.QuatIdent(),
		scale:        mgl32.Vec3{1, 1, 1},
	}
	collider := &util.MeshCollider{VertexData: rawVertexData, TransformFunc: p.GetTransformationMatrix}
	p.meshCollider = collider
	return p
}

func (p *Projectile) GetAABB() util.AABB {
	return util.NewAABB(p.position, p.extents)
}

func (p *Projectile) GetTransformationMatrix() mgl32.Mat4 {
	return mgl32.Translate3D(p.position.X(), p.position.Y(), p.position.Z()).Mul4(p.quatRotation.Mat4()).Mul4(mgl32.Scale3D(p.scale.X(), p.scale.Y(), p.scale.Z()))
}
func (p *Projectile) Draw() {
	p.shader.SetUniformAttr(2, p.GetTransformationMatrix())
	p.texture.Begin()
	p.vertexData.Begin()
	p.vertexData.Draw()
	p.vertexData.End()
	p.texture.End()
}

func (p *Projectile) SetVelocity(velocity mgl32.Vec3) {
	p.velocity = velocity
}
