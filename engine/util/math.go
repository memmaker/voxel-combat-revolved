package util

import (
	"github.com/memmaker/battleground/engine/voxel"
	"math"

	"github.com/go-gl/mathgl/mgl32"
)

func abs(x float32) float32 {
	return float32(math.Abs(float64(x)))
}

func Round(x float32) float32 {
	return float32(math.Round(float64(x)))
}

func Floor(x float32) float32 {
	return float32(math.Floor(float64(x)))
}
func Sin(x float32) float32 {
	return float32(math.Sin(float64(x)))
}

func Cos(x float32) float32 {
	return float32(math.Cos(float64(x)))
}

func ToRadian(angle float32) float32 {
	return mgl32.DegToRad(angle)
}

func max(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func Min(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func Mix(a, b, factor float32) float32 {
	return a*(1-factor) + factor*b
}

func Mix64(a, b float32, factor float64) float32 {
	return float32(float64(a)*(1.0-factor) + factor*float64(b))
}
func EucledianDistance3D(one, two mgl32.Vec3) float32 {
	return float32(math.Sqrt(float64((one.X()-two.X())*(one.X()-two.X()) + (one.Y()-two.Y())*(one.Y()-two.Y()) + (one.Z()-two.Z())*(one.Z()-two.Z()))))
}

func Clamp(value, min, max float64) float64 {
	return math.Min(math.Max(value, min), max)
}

func IntMax3(i int, i2 int, i3 int) int {
	max := i
	if i2 > max {
		max = i2
	}
	if i3 > max {
		max = i3
	}
	return max
}
func ToGrid(position mgl32.Vec3) mgl32.Vec3 {
	return mgl32.Vec3{float32(math.Floor(float64(position.X()))), float32(math.Floor(float64(position.Y()))), float32(math.Floor(float64(position.Z())))}
}

func Lerp3(one, two mgl32.Vec3, factor float64) mgl32.Vec3 {
	return mgl32.Vec3{Mix64(one.X(), two.X(), factor), Mix64(one.Y(), two.Y(), factor), Mix64(one.Z(), two.Z(), factor)}
}

func LerpQuat(one, two [4]float32, factor float64) [4]float32 {
	if one[0] == two[0] && one[1] == two[1] && one[2] == two[2] && one[3] == two[3] {
		return one
	}
	dotProduct := float64(one[0]*two[0] + one[1]*two[1] + one[2]*two[2] + one[3]*two[3])
	a := math.Acos(math.Abs(dotProduct))
	s := dotProduct / math.Abs(dotProduct)
	result := [4]float32{}
	result[0] = one[0]*float32(math.Sin(a*(1.0-factor))/math.Sin(a)) + two[0]*float32(s*math.Sin(a*factor)/math.Sin(a))
	result[1] = one[1]*float32(math.Sin(a*(1.0-factor))/math.Sin(a)) + two[1]*float32(s*math.Sin(a*factor)/math.Sin(a))
	result[2] = one[2]*float32(math.Sin(a*(1.0-factor))/math.Sin(a)) + two[2]*float32(s*math.Sin(a*factor)/math.Sin(a))
	result[3] = one[3]*float32(math.Sin(a*(1.0-factor))/math.Sin(a)) + two[3]*float32(s*math.Sin(a*factor)/math.Sin(a))
	return result
}

func LerpQuatMgl(one, two mgl32.Quat, factor float64) mgl32.Quat {
	dotProduct := float64(one.Dot(two))
	a := math.Acos(math.Abs(dotProduct))
	s := dotProduct / math.Abs(dotProduct)
	return one.Scale(float32(math.Sin(a*(1-factor)) / math.Sin(a))).Add(two.Scale(float32(s * math.Sin(a*factor) / math.Sin(a))))
}

func DirectionToAngle(direction voxel.Int3) float32 {
	angle := float32(math.Atan2(float64(direction.X), float64(direction.Z))) + math.Pi
	return angle
}
func DirectionToAngleVec(direction mgl32.Vec3) float32 {
	angle := float32(math.Atan2(float64(direction.X()), float64(direction.Z()))) + math.Pi
	return angle
}

func DirectionToCardinalAim(direction mgl32.Vec3) voxel.Int3 {
	angle := DirectionToAngleVec(direction)
	if angle < ToRadian(45) || angle > ToRadian(315) {
		return voxel.Int3{Z: -1}
	} else if angle < ToRadian(135) {
		return voxel.Int3{X: -1}
	} else if angle < ToRadian(225) {
		return voxel.Int3{Z: 1}
	} else {
		return voxel.Int3{X: 1}
	}
}

func ExtractRotation(viewMatrix mgl32.Mat4) mgl32.Quat {
	return mgl32.Mat4ToQuat(viewMatrix.Mat3().Mat4())
}

func ExtractPosition(viewMatrix mgl32.Mat4) mgl32.Vec3 {
	return viewMatrix.Col(3).Vec3()
}

// ExtractUniformScale only works if the matrix is uniformly scaled. Does not work with non-uniform scaling and shearing.
func ExtractUniformScale(viewMatrix mgl32.Mat4) mgl32.Vec3 {
	return mgl32.Vec3{viewMatrix.Col(0).Len(), viewMatrix.Col(1).Len(), viewMatrix.Col(2).Len()}
}

/* TS
function easeInOutQuad(x: number): number {
return x < 0.5 ? 2 * x * x : 1 - Math.pow(-2 * x + 2, 2) / 2;
}

*/

func EaseInOutQuad(x float64) float64 {
	if x < 0.5 {
		return 2 * x * x
	} else {
		return 1 - math.Pow(-2*x+2, 2)/2
	}
}

/* TS
function easeOutQuart(x: number): number {
return 1 - Math.pow(1 - x, 4);
}
*/

func EaseOutQuart(x float64) float64 {
	return 1 - math.Pow(1-x, 4)
}

/*
function easeOutSine(x: number): number {
  return Math.sin((x * Math.PI) / 2);
}
*/

func EaseOutSine(x float64) float64 {
	return math.Sin((x * math.Pi) / 2)
}
