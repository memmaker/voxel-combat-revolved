package util

import (
	"github.com/go-gl/mathgl/mgl32"
	"testing"
)

// execute with: go test -bench=. -test.benchmem -test.benchtime=10s
func BenchmarkTriangleSegmentIntersection(b *testing.B) {
	triangle := [3]mgl32.Vec3{
		mgl32.Vec3{1, 0, 0},
		mgl32.Vec3{0, 0, 0},
		mgl32.Vec3{0, 1, 0},
	}
	rayStart := mgl32.Vec3{0.25, 0.25, 1}
	rayEnd := mgl32.Vec3{0.25, 0.25, -1}
	//var intersectsTriangle bool
	//var collision mgl32.Vec3
	for i := 0; i < b.N; i++ {
		//intersectsTriangle, collision = RayIntersectsTriangle(rayStart, rayEnd, triangle) // 41ns, no allocs, wrong point of intersection
		//intersectsTriangle, collision, _ = intersectTriangle(rayStart, rayEnd, triangle[0], triangle[1], triangle[2]) // slower and apparently wrong
		_, _ = intersectLineSegmentTriangle(rayStart, rayEnd, triangle[0], triangle[1], triangle[2])
		//_, _ = LineSegmentIntersectsTriangle(rayStart, rayEnd, triangle)
	}
	//println(fmt.Sprintf("%v %v", intersectsTriangle, collision))
}
