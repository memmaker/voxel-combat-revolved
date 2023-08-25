package util

import (
	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/glhf"
	"math"
)

type CollisionInfo struct {
	Result                            float32
	Normal                            mgl32.Vec3
	MinkowskiDifferenceContainsOrigin bool
}

// AABBVoxelMapIntersection returns the correction vector and whether or not a collision occurred.
// CANNOT BE CALLED WITH HIGH-VELOCITY OBJECTS
func AABBVoxelMapIntersection(previousPos, position, extents mgl32.Vec3, isSolid func(x int32, y int32, z int32) bool) (mgl32.Vec3, bool) {
	hitSomething := false
	// adapted from: https://luisreis.net/blog/aabb_collision_handling/
	for {
		velocity := position.Sub(previousPos)
		minX := int32(math.Floor(math.Min(float64(position.X()), float64(previousPos.X())) - float64(extents.X()/2.0)))
		maxX := int32(math.Floor(math.Max(float64(position.X()), float64(previousPos.X())) + float64(extents.X()/2.0)))

		minY := int32(math.Floor(math.Min(float64(position.Y()), float64(previousPos.Y())) - float64(extents.Y()/2.0)))
		maxY := int32(math.Floor(math.Max(float64(position.Y()), float64(previousPos.Y())) + float64(extents.Y()/2.0)))

		minZ := int32(math.Floor(math.Min(float64(position.Z()), float64(previousPos.Z())) - float64(extents.Z()/2.0)))
		maxZ := int32(math.Floor(math.Max(float64(position.Z()), float64(previousPos.Z())) + float64(extents.Z()/2.0)))

		r := CollisionInfo{
			Result: 1,
			Normal: mgl32.Vec3{0, 0, 0},
		}

		for yi := minY; yi <= maxY; yi++ { // this isn't suitable for high-velocity objects
			for zi := minZ; zi <= maxZ; zi++ {
				for xi := minX; xi <= maxX; xi++ {
					if !isSolid(xi, yi, zi) {
						continue
					}
					a := NewAABBFromMin(mgl32.Vec3{previousPos.X() - extents.X()*0.5, previousPos.Y() - extents.Y()*0.5, previousPos.Z() - extents.Z()*0.5}, extents)
					b := NewAABBFromMin(mgl32.Vec3{float32(xi), float32(yi), float32(zi)}, mgl32.Vec3{1, 1, 1})
					c := SweepAABB(a, b, velocity)
					if c.Result < r.Result {
						r = c
					}
				}
			}
		}

		epsilon := float32(0.001)
		position = previousPos.Add(velocity.Mul(r.Result).Add(r.Normal.Mul(epsilon)))
		if r.Result == 1 {
			break
		}
		hitSomething = true

		BdotB := r.Normal.Dot(r.Normal)

		if BdotB != 0 {
			previousPos = position
			AdotB := (1 - r.Result) * velocity.Dot(r.Normal)
			//translation = translation.AddFlat(velocity.Mul(1 - r.Result)).Sub(r.Normal.Mul(AdotB / BdotB))
			position = position.Add(velocity.Mul(1 - r.Result).Sub(r.Normal.Mul(AdotB / BdotB)))
		}
	}

	return position, hitSomething
}

type AABB struct {
	center               mgl32.Vec3
	extents              mgl32.Vec3 // size in respective axis, they extend from the center to the max and min
	vertexesForDebugDraw *glhf.VertexSlice[glhf.GlFloat]
	id                   int
}

func NewAABB(center, extents mgl32.Vec3) AABB {
	return AABB{
		center:  center,
		extents: extents,
	}
}

func NewAABBFromMin(min, extents mgl32.Vec3) AABB {
	return AABB{
		center:  min.Add(extents.Mul(0.5)),
		extents: extents,
	}
}
func (a AABB) Min() mgl32.Vec3 {
	return a.center.Sub(a.extents.Mul(0.5))
}

func (a AABB) Max() mgl32.Vec3 {
	return a.center.Add(a.extents.Mul(0.5))
}

func (a AABB) MinkowskiDifference(other AABB) AABB {
	minM := other.Min().Sub(a.Max())
	extM := a.extents.Add(other.extents)
	return NewAABBFromMin(minM, extM)
}

// an AABB is basically [2]mgl32.Vec3

func AABBMinkowskiDifference(aMinPos, aExtents, bMinPos, bExtents mgl32.Vec3) (mgl32.Vec3, mgl32.Vec3) {
	minM := bMinPos.Sub(aMinPos.Add(aExtents))
	extM := aExtents.Add(bExtents)
	return minM, extM
}
func (a AABB) Draw(shader *glhf.Shader) {
	if a.vertexesForDebugDraw == nil {
		a.vertexesForDebugDraw = glhf.MakeVertexSlice(shader, 3*2*4, 3*2*4) // 2*4 verts for top and bottom, 2*4 verts for sides
		a.vertexesForDebugDraw.SetPrimitiveType(gl.LINES)
		a.vertexesForDebugDraw.Begin()
		a.vertexesForDebugDraw.SetVertexData([]glhf.GlFloat{
			// top
			glhf.GlFloat(a.Min().X()), glhf.GlFloat(a.Max().Y()), glhf.GlFloat(a.Min().Z()),
			glhf.GlFloat(a.Max().X()), glhf.GlFloat(a.Max().Y()), glhf.GlFloat(a.Min().Z()),

			glhf.GlFloat(a.Max().X()), glhf.GlFloat(a.Max().Y()), glhf.GlFloat(a.Min().Z()),
			glhf.GlFloat(a.Max().X()), glhf.GlFloat(a.Max().Y()), glhf.GlFloat(a.Max().Z()),

			glhf.GlFloat(a.Max().X()), glhf.GlFloat(a.Max().Y()), glhf.GlFloat(a.Max().Z()),
			glhf.GlFloat(a.Min().X()), glhf.GlFloat(a.Max().Y()), glhf.GlFloat(a.Max().Z()),

			glhf.GlFloat(a.Min().X()), glhf.GlFloat(a.Max().Y()), glhf.GlFloat(a.Max().Z()),
			glhf.GlFloat(a.Min().X()), glhf.GlFloat(a.Max().Y()), glhf.GlFloat(a.Min().Z()),

			// bottom
			glhf.GlFloat(a.Min().X()), glhf.GlFloat(a.Min().Y()), glhf.GlFloat(a.Min().Z()),
			glhf.GlFloat(a.Max().X()), glhf.GlFloat(a.Min().Y()), glhf.GlFloat(a.Min().Z()),

			glhf.GlFloat(a.Max().X()), glhf.GlFloat(a.Min().Y()), glhf.GlFloat(a.Min().Z()),
			glhf.GlFloat(a.Max().X()), glhf.GlFloat(a.Min().Y()), glhf.GlFloat(a.Max().Z()),

			glhf.GlFloat(a.Max().X()), glhf.GlFloat(a.Min().Y()), glhf.GlFloat(a.Max().Z()),
			glhf.GlFloat(a.Min().X()), glhf.GlFloat(a.Min().Y()), glhf.GlFloat(a.Max().Z()),

			glhf.GlFloat(a.Min().X()), glhf.GlFloat(a.Min().Y()), glhf.GlFloat(a.Max().Z()),
			glhf.GlFloat(a.Min().X()), glhf.GlFloat(a.Min().Y()), glhf.GlFloat(a.Min().Z()),

			// sides
			glhf.GlFloat(a.Min().X()), glhf.GlFloat(a.Min().Y()), glhf.GlFloat(a.Min().Z()),
			glhf.GlFloat(a.Min().X()), glhf.GlFloat(a.Max().Y()), glhf.GlFloat(a.Min().Z()),

			glhf.GlFloat(a.Max().X()), glhf.GlFloat(a.Min().Y()), glhf.GlFloat(a.Min().Z()),
			glhf.GlFloat(a.Max().X()), glhf.GlFloat(a.Max().Y()), glhf.GlFloat(a.Min().Z()),

			glhf.GlFloat(a.Max().X()), glhf.GlFloat(a.Min().Y()), glhf.GlFloat(a.Max().Z()),
			glhf.GlFloat(a.Max().X()), glhf.GlFloat(a.Max().Y()), glhf.GlFloat(a.Max().Z()),

			glhf.GlFloat(a.Min().X()), glhf.GlFloat(a.Min().Y()), glhf.GlFloat(a.Max().Z()),
			glhf.GlFloat(a.Min().X()), glhf.GlFloat(a.Max().Y()), glhf.GlFloat(a.Max().Z()),
		})
		a.vertexesForDebugDraw.End()
	}
	a.vertexesForDebugDraw.Begin()
	a.vertexesForDebugDraw.Draw()
	a.vertexesForDebugDraw.End()
}

func (a AABB) Contains(vec3 mgl32.Vec3) bool {
	minVal := a.Min()
	maxVal := a.Max()
	return vec3.X() >= minVal.X() && vec3.X() <= maxVal.X() &&
		vec3.Y() >= minVal.Y() && vec3.Y() <= maxVal.Y() &&
		vec3.Z() >= minVal.Z() && vec3.Z() <= maxVal.Z()
}

func (a AABB) Center() mgl32.Vec3 {
	return a.center
}

// SweepAABB will return a value between 0 and 1, where 0.5 means "a collision occurred halfway of the AABB's path" and 1 means "there was no collision".
// Notably, if one of the AABB's is completely inside the other, it will return 1, meaning our AABBs will be "hollow".
// It also returns the Normal to the surface that was hit. This will be useful when we implement wall sliding.
func SweepAABB(a, b AABB, vel mgl32.Vec3) CollisionInfo {
	// adapted from: https://luisreis.net/blog/aabb_collision_handling/
	m := a.MinkowskiDifference(b)
	return SweepAABBFromMinkowski(m, vel)
}

func SweepAABBFromMinkowski(m AABB, vel mgl32.Vec3) CollisionInfo {
	// adapted from: https://luisreis.net/blog/aabb_collision_handling/
	containsOrigin := m.Contains(mgl32.Vec3{})
	h := float32(1.0)
	nx := 0
	ny := 0
	nz := 0
	var s float32
	nullVec := mgl32.Vec3{}

	// X Min
	s = LineToPlaneIntersection(nullVec, vel, m.Min(), mgl32.Vec3{-1, 0, 0})
	if s >= 0 && vel.X() > 0 && s < h && InRange(s*vel.Y(), m.Min().Y(), m.Max().Y()) && InRange(s*vel.Z(), m.Min().Z(), m.Max().Z()) {
		nx = -1
		h = s
		ny = 0
		nz = 0
	}

	// X Max
	s = LineToPlaneIntersection(nullVec, vel, mgl32.Vec3{m.Max().X(), m.Min().Y(), m.Min().Z()}, mgl32.Vec3{1, 0, 0})
	if s >= 0 && vel.X() < 0 && s < h && InRange(s*vel.Y(), m.Min().Y(), m.Max().Y()) && InRange(s*vel.Z(), m.Min().Z(), m.Max().Z()) {
		nx = 1
		h = s
		ny = 0
		nz = 0
	}

	// Y Min
	s = LineToPlaneIntersection(nullVec, vel, m.Min(), mgl32.Vec3{0, -1, 0})
	if s >= 0 && vel.Y() > 0 && s < h && InRange(s*vel.X(), m.Min().X(), m.Max().X()) && InRange(s*vel.Z(), m.Min().Z(), m.Max().Z()) {
		nx = 0
		h = s
		ny = -1
		nz = 0
	}

	// Y Max
	s = LineToPlaneIntersection(nullVec, vel, mgl32.Vec3{m.Min().X(), m.Max().Y(), m.Min().Z()}, mgl32.Vec3{0, 1, 0})
	if s >= 0 && vel.Y() < 0 && s < h && InRange(s*vel.X(), m.Min().X(), m.Max().X()) && InRange(s*vel.Z(), m.Min().Z(), m.Max().Z()) {
		nx = 0
		h = s
		ny = 1
		nz = 0
	}

	// Z Min
	s = LineToPlaneIntersection(nullVec, vel, m.Min(), mgl32.Vec3{0, 0, -1})
	if s >= 0 && vel.Z() > 0 && s < h && InRange(s*vel.X(), m.Min().X(), m.Max().X()) && InRange(s*vel.Y(), m.Min().Y(), m.Max().Y()) {
		nx = 0
		h = s
		ny = 0
		nz = -1
	}

	// Z Max
	s = LineToPlaneIntersection(nullVec, vel, mgl32.Vec3{m.Min().X(), m.Min().Y(), m.Max().Z()}, mgl32.Vec3{0, 0, 1})
	if s >= 0 && vel.Z() < 0 && s < h && InRange(s*vel.X(), m.Min().X(), m.Max().X()) && InRange(s*vel.Y(), m.Min().Y(), m.Max().Y()) {
		nx = 0
		h = s
		ny = 0
		nz = 1
	}

	return CollisionInfo{
		Result:                            h,
		Normal:                            mgl32.Vec3{float32(nx), float32(ny), float32(nz)},
		MinkowskiDifferenceContainsOrigin: containsOrigin,
	}
}

func InRange(x, min, max float32) bool {
	return x >= min && x <= max
}

func LineToPlaneIntersection(p, u, v, n mgl32.Vec3) float32 {
	NdotU := n.Dot(u)
	if NdotU == 0 {
		return math.MaxFloat32
	}
	return n.Dot(v.Sub(p)) / NdotU
}
