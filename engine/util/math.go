package util

import (
	"github.com/memmaker/battleground/engine/voxel"
	"math"
	"math/rand"

	"github.com/go-gl/mathgl/mgl32"
)

func Sin(x float32) float32 {
	return float32(math.Sin(float64(x)))
}

func Cos(x float32) float32 {
	return float32(math.Cos(float64(x)))
}

func ToRadian(angle float32) float32 {
	return mgl32.DegToRad(angle)
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

func ToGrid(position mgl32.Vec3) mgl32.Vec3 {
	return mgl32.Vec3{float32(math.Floor(float64(position.X()))), float32(math.Floor(float64(position.Y()))), float32(math.Floor(float64(position.Z())))}
}

func Lerp3(one, two mgl32.Vec3, factor float64) mgl32.Vec3 {
	return mgl32.Vec3{Mix64(one.X(), two.X(), factor), Mix64(one.Y(), two.Y(), factor), Mix64(one.Z(), two.Z(), factor)}
}

func Lerp3f(one, two [3]float32, factor float64) [3]float32 {
	return [3]float32{Mix64(one[0], two[0], factor), Mix64(one[1], two[1], factor), Mix64(one[2], two[2], factor)}
}

func LerpQuat(one, two [4]float32, factor float64) [4]float32 {
	if one[0] == two[0] && one[1] == two[1] && one[2] == two[2] && one[3] == two[3] {
		return one
	}
	dotProduct := float64(one[0]*two[0] + one[1]*two[1] + one[2]*two[2] + one[3]*two[3])
	if dotProduct == 1 {
		return one
	}
	a := math.Acos(math.Abs(dotProduct))
	s := dotProduct / math.Abs(dotProduct)
	result := [4]float32{}
	result[0] = one[0]*float32(math.Sin(a*(1.0-factor))/math.Sin(a)) + two[0]*float32(s*math.Sin(a*factor)/math.Sin(a))
	result[1] = one[1]*float32(math.Sin(a*(1.0-factor))/math.Sin(a)) + two[1]*float32(s*math.Sin(a*factor)/math.Sin(a))
	result[2] = one[2]*float32(math.Sin(a*(1.0-factor))/math.Sin(a)) + two[2]*float32(s*math.Sin(a*factor)/math.Sin(a))
	result[3] = one[3]*float32(math.Sin(a*(1.0-factor))/math.Sin(a)) + two[3]*float32(s*math.Sin(a*factor)/math.Sin(a))
	return result
}

/*
	float phi = (1 + Mathf.Sqrt(5)) / 2;//golden ratio
    float angle_stride = 360 * phi;
    float radius(float k, float n, float b)
    {
        return k > n - b ? 1 : Mathf.Sqrt(k - 0.5f) / Mathf.Sqrt(n - (b + 1) / 2);
    }

    int b = (int)(alpha * Mathf.Sqrt(n));  //# number of boundary points

    List<Vector2>points = new List<Vector2>();
    for (int k = 0; k < n; k++)
    {
        float r = radius(k, n, b);
        float theta = geodesic ? k * 360 * phi : k * angle_stride;
        float x = !float.IsNaN(r * Mathf.Cos(theta)) ? r * Mathf.Cos(theta) : 0;
        float y = !float.IsNaN(r * Mathf.Sin(theta)) ? r * Mathf.Sin(theta) : 0;
        points.Add(new Vector2(x, y));
    }
*/
func SunflowerSeeds(n int, alpha float64) []mgl32.Vec2 {
	phi := (1 + math.Sqrt(5)) / 2
	angleStride := 360 * phi
	radius := func(k, n, b int) float64 {
		if k > n-b {
			return 1
		}
		return math.Sqrt(float64(k-1)) / math.Sqrt(float64(n-(b+1)/2))
	}
	b := int(alpha * math.Sqrt(float64(n)))
	points := make([]mgl32.Vec2, 0)
	for k := 0; k < n; k++ {
		r := radius(k, n, b)
		theta := float64(k) * angleStride
		x := r * math.Cos(theta)
		y := r * math.Sin(theta)
		points = append(points, mgl32.Vec2{float32(x), float32(y)})
	}
	return points
}

func LerpQuatMgl(one, two mgl32.Quat, factor float64) mgl32.Quat {
	dotProduct := float64(one.Dot(two))
	a := math.Acos(math.Abs(dotProduct))
	s := dotProduct / math.Abs(dotProduct)
	return one.Scale(float32(math.Sin(a*(1-factor)) / math.Sin(a))).Add(two.Scale(float32(s * math.Sin(a*factor) / math.Sin(a))))
}

func AngleToVector(angle float64) mgl32.Vec2 {
	return mgl32.Vec2{float32(math.Cos(angle)), float32(math.Sin(angle))}
}

func DirectionToAngle(direction voxel.Int3) float32 {
	angle := float32(math.Atan2(float64(direction.X), float64(direction.Z))) + math.Pi
	return angle
}
func DirectionTo2D(direction mgl32.Vec3) mgl32.Vec3 {
	return mgl32.Vec3{direction.X(), 0, direction.Z()}
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

func RandomChoice(choices []voxel.Int3) voxel.Int3 {
	return choices[rand.Intn(len(choices))]
}

/*
function easeInOutExpo(x: number): number {
return x === 0
  ? 0
  : x === 1
  ? 1
  : x < 0.5 ? Math.pow(2, 20 * x - 10) / 2
  : (2 - Math.pow(2, -20 * x + 10)) / 2;
}
*/

func EaseInOutExpo(x float64) float64 {
	if x == 0 {
		return 0
	} else if x == 1 {
		return 1
	} else if x < 0.5 {
		return math.Pow(2, 20*x-10) / 2
	} else {
		return (2 - math.Pow(2, -20*x+10)) / 2
	}
}

func EaseSlowEnd(x float64) float64 {
	if x < 0.5 {
		return float64(2) * (x * x)
	} else {
		return 1.0 - (1.0 / (5.0*((2.0*x)+0.8) + 1.0))
	}
}

func SmallestAngleBetween(vecOne, vecTwo mgl32.Vec3) float64 {
	dot := vecOne.Dot(vecTwo)
	cross := vecOne.Cross(vecTwo)
	det := cross.Len()
	return math.Atan2(float64(det), float64(dot))
}

func AngleBetween(vecOne, vecTwo mgl32.Vec2) float32 {
	return float32(math.Atan2(float64(vecOne.X()*vecTwo.Y()-vecOne.Y()*vecTwo.X()), float64(vecOne.X()*vecTwo.X()+vecOne.Y()*vecTwo.Y())))
}

func TrajectoryXY(time float64, angle float64, velocity float64, gravity float64) (float64, float64) {
	x := velocity * math.Cos(angle) * time
	y := (velocity * math.Sin(angle) * time) - (0.5 * gravity * math.Pow(time, 2))
	return x, y
}

func TimeOfFlight(xPosition float64, angle float64, velocity float64) float64 {
	return xPosition / (velocity * math.Cos(angle))
}
func MinLaunchVelocity(targetPos mgl32.Vec2, gravity float64) float64 {
	return math.Sqrt((float64(targetPos.Y()) + math.Sqrt(math.Pow(float64(targetPos.Y()), 2)+math.Pow(float64(targetPos.X()), 2))) * gravity)
}
func MinLaunchAngle(targetPos mgl32.Vec2) float64 {
	return math.Atan((float64(targetPos.Y()) / float64(targetPos.X())) + math.Sqrt((math.Pow(float64(targetPos.Y()), 2)/math.Pow(float64(targetPos.X()), 2))+1))
}
func CalculateTrajectory(sourcePos, dest mgl32.Vec3, maxVelocity, gravity float64) []mgl32.Vec3 {
	destRelatedToOrigin := dest.Sub(sourcePos) // translate by sourcePos

	dest2D := mgl32.Vec2{destRelatedToOrigin.X(), destRelatedToOrigin.Z()}
	xAxis := mgl32.Vec2{1.0, 0.0}
	angle := AngleBetween(xAxis, dest2D)

	rotationForAxisAlign := mgl32.QuatRotate(angle, mgl32.Vec3{0, 1, 0}).Mat4()
	rotatedDest := rotationForAxisAlign.Mul4x1(destRelatedToOrigin.Vec4(1)).Vec3()

	// reduced to 2d problem by rotating around y axis to align with x axis
	// source is now the origin
	twoDeeX := rotatedDest.X()
	twoDeeY := rotatedDest.Y()

	velocity := MinLaunchVelocity(mgl32.Vec2{twoDeeX, twoDeeY}, gravity)
	aimAngle := MinLaunchAngle(mgl32.Vec2{twoDeeX, twoDeeY})

	if math.IsNaN(aimAngle) || velocity > maxVelocity {
		return []mgl32.Vec3{}
	}

	timeToTarget := TimeOfFlight(float64(twoDeeX), aimAngle, velocity)
	inverseRotation := rotationForAxisAlign.Inv()
	var trajectory3D []mgl32.Vec3
	for i := 0; i <= 10; i++ {
		percent := float64(i) / 10.0
		time := percent * timeToTarget
		currentX, currentY := TrajectoryXY(time, aimAngle, velocity, gravity)
		rotatedPoint := inverseRotation.Mul4x1(mgl32.Vec4{float32(currentX), float32(currentY), 0.0, 1.0}).Vec3()
		translatedPoint := rotatedPoint.Add(sourcePos)
		trajectory3D = append(trajectory3D, translatedPoint)
	}
	return trajectory3D
}
