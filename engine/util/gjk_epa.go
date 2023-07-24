package util

import (
	"github.com/go-gl/mathgl/mgl32"
	"math"
)

type CollisionPoints struct {
	Normal           mgl32.Vec3
	PenetrationDepth float64
	HasCollision     bool
}

func EPA(simplex *Simplex, a, b Collider) CollisionPoints { // probably buggy :P
	// adapted from: https://blog.winter.dev/2020/epa-algorithm/
	polytope := []mgl32.Vec3{simplex.points[0], simplex.points[1], simplex.points[2], simplex.points[3]}
	faces := []uint{
		0, 1, 2,
		0, 3, 1,
		0, 2, 3,
		1, 3, 2,
	}
	normals, minFace := GetFaceNormals(polytope, faces)

	var minNormal mgl32.Vec3
	var minDistance = math.MaxFloat64

	for minDistance == math.MaxFloat64 {
		minNormal = normals[minFace].Vec3()
		minDistance = float64(normals[minFace].W())
		support := Support(a, b, minNormal)
		distance := minNormal.Dot(support)

		if math.Abs(float64(distance)-minDistance) < 0.001 {
			minDistance = math.MaxFloat64

			var uniqueEdges [][2]uint

			for i := 0; i < len(normals); i++ {
				if SameDirection(normals[i].Vec3(), support) {
					f := uint(i * 3)

					AddIfUniqueEdge(uniqueEdges, faces, f, f+1)
					AddIfUniqueEdge(uniqueEdges, faces, f+1, f+2)
					AddIfUniqueEdge(uniqueEdges, faces, f+2, f)

					faces[f+2] = faces[len(faces)-1]
					faces = faces[:len(faces)-1]
					faces[f+1] = faces[len(faces)-1]
					faces = faces[:len(faces)-1]
					faces[f] = faces[len(faces)-1]
					faces = faces[:len(faces)-1]

					normals[i] = normals[len(normals)-1]
					normals = normals[:len(normals)-1]

					i--
				}
			}

			var newFaces []uint

			for _, edge := range uniqueEdges {
				newFaces = append(newFaces, edge[0], edge[1], uint(len(polytope)))
			}

			polytope = append(polytope, support)

			newNormals, newMinFace := GetFaceNormals(polytope, newFaces)

			oldMinDistance := float32(math.MaxFloat32)

			for i := uint(0); i < uint(len(normals)); i++ {
				if normals[i].W() < oldMinDistance {
					oldMinDistance = normals[i].W()
					minFace = i
				}
			}

			if newNormals[newMinFace].W() < oldMinDistance {
				minFace = newMinFace + uint(len(normals))
			}

			normals = append(normals, newNormals...)
			faces = append(faces, newFaces...)
		}
	}

	var points CollisionPoints
	points.Normal = minNormal
	points.PenetrationDepth = minDistance + 0.001
	points.HasCollision = true

	return points
}

// AddIfUniqueEdge tests if the reverse of an edge already exists in the list and if so, removes it.
// If you look at how the winding works out, if a neighboring face shares an edge,
// it will be in reverse order.
// Remember, we only want to store the edges that we are going to save
// because every edge gets removed first, then we repair.
func AddIfUniqueEdge(edges [][2]uint, faces []uint, a uint, b uint) {
	for i := len(edges) - 1; i >= 0; i-- {
		// if the reverse edge exists, remove it
		if edges[i][0] == faces[b] && edges[i][1] == faces[a] {
			edges = append(edges[:i], edges[i+1:]...)
			return
		}
	}
	// else add this edge
	edges = append(edges, [2]uint{faces[a], faces[b]})
	return
}

func GetFaceNormals(polytope []mgl32.Vec3, faces []uint) ([]mgl32.Vec4, uint) {
	var normals []mgl32.Vec4
	minTriangle := uint(0)
	minDistance := math.MaxFloat64

	for i := 0; i < len(faces); i += 3 {
		a := polytope[faces[i]]
		b := polytope[faces[i+1]]
		c := polytope[faces[i+2]]
		BsubA := b.Sub(a)
		CsubA := c.Sub(a)
		cross := BsubA.Cross(CsubA)
		normal := cross.Normalize()
		distance := normal.Dot(a)

		if distance < 0 {
			normal = normal.Mul(-1)
			distance = distance * -1
		}

		normals = append(normals, mgl32.Vec4{normal.X(), normal.Y(), normal.Z(), distance})

		if float64(distance) < minDistance {
			minDistance = float64(distance)
			minTriangle = uint(i / 3)
		}
	}
	return normals, minTriangle
}
