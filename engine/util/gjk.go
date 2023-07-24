package util

import (
	"github.com/go-gl/mathgl/mgl32"
)

type Simplex struct {
	points [4]mgl32.Vec3
	size   uint8
}

func NewSimplex() *Simplex {
	return &Simplex{}
}

func (s *Simplex) PushFront(v mgl32.Vec3) {
	s.points = [4]mgl32.Vec3{v, s.points[0], s.points[1], s.points[2]}
	s.size = s.size + 1
	if s.size > 4 {
		s.size = 4
	}
}

func (s *Simplex) Size() uint8 {
	return s.size
}

func (s *Simplex) SetPoints(vec3s []mgl32.Vec3) *Simplex {
	for i, v := range vec3s {
		s.points[i] = v
	}
	s.size = uint8(len(vec3s))
	return s
}
func GJK(a, b Collider) (bool, *Simplex) {
	// adapted from: https://blog.winter.dev/2020/gjk-algorithm/
	//println(fmt.Sprintf("\n\n[GJK-START] a: %s, b: %s", a.ToString(), b.ToString()))
	support := Support(a, b, mgl32.Vec3{1, 0, 0})
	simplex := NewSimplex()
	simplex.PushFront(support)

	startDir := support.Mul(-1)
	direction := &startDir

	for {
		support = Support(a, b, *direction)
		//println(fmt.Sprintf("direction: %v -> support: %v", *direction, support))

		if support.Dot(*direction) <= 0 {
			//println("[GJK-END] no collision")
			return false, nil // no collision
		}
		simplex.PushFront(support)
		if NextSimplex(simplex, direction) {
			//println("[GJK-END] COLLISION HIT!")
			return true, simplex
		}
	}
}
func SameDirection(a, b mgl32.Vec3) bool {
	return a.Dot(b) > 0
}
func NextSimplex(simplex *Simplex, direction *mgl32.Vec3) bool {
	switch simplex.Size() {
	case 2:
		line := Line(simplex, direction)
		//println(fmt.Sprintf("[line] direction: %v", direction))
		return line
	case 3:
		triangle := Triangle(simplex, direction)
		//println(fmt.Sprintf("[triangle] direction: %v", direction))
		return triangle
	case 4:
		tetrahedron := Tetrahedron(simplex, direction)
		//println(fmt.Sprintf("[tetrahedron] direction: %v", direction))
		return tetrahedron
	}
	return false
}

func Tetrahedron(simplex *Simplex, direction *mgl32.Vec3) bool {
	a := simplex.points[0]
	b := simplex.points[1]
	c := simplex.points[2]
	d := simplex.points[3]

	ab := b.Sub(a)
	ac := c.Sub(a)
	ad := d.Sub(a)
	ao := a.Mul(-1)

	abc := ab.Cross(ac)
	acd := ac.Cross(ad)
	adb := ad.Cross(ab)

	if SameDirection(abc, ao) {
		return Triangle(simplex.SetPoints([]mgl32.Vec3{a, b, c}), direction)
	} else if SameDirection(acd, ao) {
		return Triangle(simplex.SetPoints([]mgl32.Vec3{a, c, d}), direction)
	} else if SameDirection(adb, ao) {
		return Triangle(simplex.SetPoints([]mgl32.Vec3{a, d, b}), direction)
	}

	return true
}

func Triangle(simplex *Simplex, direction *mgl32.Vec3) bool {
	a := simplex.points[0]
	b := simplex.points[1]
	c := simplex.points[2]

	ab := b.Sub(a)
	ac := c.Sub(a)
	ao := a.Mul(-1)

	abc := ab.Cross(ac)

	if SameDirection(abc.Cross(ac), ao) {
		if SameDirection(ac, ao) {
			simplex.SetPoints([]mgl32.Vec3{a, c})
			newDir := ac.Cross(ao).Cross(ac)
			*direction = newDir
		} else {
			return Line(simplex.SetPoints([]mgl32.Vec3{a, b}), direction)
		}
	} else {
		if SameDirection(ab.Cross(abc), ao) {
			return Line(simplex.SetPoints([]mgl32.Vec3{a, b}), direction)
		} else {
			if SameDirection(abc, ao) {
				*direction = abc
			} else {
				simplex.SetPoints([]mgl32.Vec3{a, c, b})
				newDir := abc.Mul(-1)
				*direction = newDir
			}
		}
	}
	return false
}

func Line(simplex *Simplex, direction *mgl32.Vec3) bool {
	a := simplex.points[0]
	b := simplex.points[1]
	ab := b.Sub(a)
	ao := a.Mul(-1)
	if SameDirection(ab, ao) {
		newDir := ab.Cross(ao).Cross(ab)
		*direction = newDir
	} else {
		simplex.SetPoints([]mgl32.Vec3{a})
		*direction = ao
	}
	return false
}
