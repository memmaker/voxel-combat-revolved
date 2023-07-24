package util

import "github.com/go-gl/mathgl/mgl32"

// bench: no allocs, 65ns/op
func intersectLineSegmentTriangle(rayStart, rayEnd mgl32.Vec3, v0, v1, v2 mgl32.Vec3) (bool, mgl32.Vec3) {
	const EPSILON = 0.000001

	direction := rayEnd.Sub(rayStart)
	edge1 := v1.Sub(v0)
	edge2 := v2.Sub(v0)

	h := direction.Cross(edge2)
	a := edge1.Dot(h)

	if a > -EPSILON && a < EPSILON {
		return false, mgl32.Vec3{} // This ray is parallel to this triangle.
	}

	f := 1.0 / a
	s := rayStart.Sub(v0)
	u := f * s.Dot(h)

	if u < 0.0 || u > 1.0 {
		return false, mgl32.Vec3{} // Intersection is outside the triangle
	}

	q := s.Cross(edge1)
	v := f * direction.Dot(q)

	if v < 0.0 || u+v > 1.0 {
		return false, mgl32.Vec3{} // Intersection is outside the triangle
	}

	t := f * edge2.Dot(q)

	if t > EPSILON {
		return true, rayStart.Add(direction.Mul(t)) // Intersection
	}

	return false, mgl32.Vec3{} // This means that there is a line intersection but not a ray intersection.
}
