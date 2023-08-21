package util

import (
	"encoding/json"
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/voxel"
)

type Transform struct {
	position    mgl32.Vec3
	rotation    mgl32.Quat
	scale       mgl32.Vec3
	nameOfOwner string
}

func (t *Transform) GetName() string {
	return t.nameOfOwner
}
func (t *Transform) SetName(name string) {
	t.nameOfOwner = name
}
func NewDefaultTransform(name string) *Transform {
	return &Transform{
		position:    mgl32.Vec3{0, 0, 0},
		rotation:    mgl32.QuatIdent(),
		scale:       mgl32.Vec3{1, 1, 1},
		nameOfOwner: name,
	}
}

func NewTransform(position mgl32.Vec3, rotation mgl32.Quat, scale mgl32.Vec3) *Transform {
	return &Transform{
		position: position,
		rotation: rotation,
		scale:    scale,
	}
}
func NewTransformFromTopDown(position mgl32.Vec3, viewingAngle, rotationAngle float32) *Transform {
	t := &Transform{
		position: position,
		rotation: mgl32.QuatIdent(),
		scale:    mgl32.Vec3{1, 1, 1},
	}
	t.SetTopdownRotation(viewingAngle, rotationAngle)
	return t
}
func NewTransformFromForward(position mgl32.Vec3, forward mgl32.Vec3) *Transform {
	t := &Transform{
		position: position,
		rotation: mgl32.QuatIdent(),
		scale:    mgl32.Vec3{1, 1, 1},
	}
	t.SetForward(forward)
	return t
}

func NewTransformFromLookAt(position, target mgl32.Vec3) *Transform {
	t := &Transform{
		position: position,
		rotation: mgl32.QuatIdent(),
		scale:    mgl32.Vec3{1, 1, 1},
	}
	t.SetLookAt(target)
	return t
}

func (t *Transform) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Position mgl32.Vec3 `json:"position"`
		Rotation mgl32.Quat `json:"rotation"`
		Scale    mgl32.Vec3 `json:"scale"`
	}{
		Position: t.position,
		Rotation: t.rotation,
		Scale:    t.scale,
	})
}

func (t *Transform) UnmarshalJSON(data []byte) error {
	var tmp struct {
		Position mgl32.Vec3 `json:"position"`
		Rotation mgl32.Quat `json:"rotation"`
		Scale    mgl32.Vec3 `json:"scale"`
	}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	t.position = tmp.Position
	t.rotation = tmp.Rotation
	t.scale = tmp.Scale
	return nil
}
func (t *Transform) GetTransformMatrix() mgl32.Mat4 {
	translation := t.GetTranslationMatrix()
	rotation := t.GetRotationMatrix()
	scale := t.GetScaleMatrix()
	return translation.Mul4(rotation).Mul4(scale)
}

func (t *Transform) GetScaleMatrix() mgl32.Mat4 {
	return mgl32.Scale3D(t.scale.X(), t.scale.Y(), t.scale.Z())
}

func (t *Transform) GetRotationMatrix() mgl32.Mat4 {
	return t.rotation.Mat4()
}

func (t *Transform) GetTranslationMatrix() mgl32.Mat4 {
	return mgl32.Translate3D(t.position.X(), t.position.Y(), t.position.Z())
}
func (t *Transform) GetPosition() mgl32.Vec3 {
	return t.position
}

func (t *Transform) GetBlockPosition() voxel.Int3 {
	return voxel.PositionToGridInt3(t.GetPosition())
}

func (t *Transform) GetRotation() mgl32.Quat {
	return t.rotation
}
func (t *Transform) SetRotation(rotation mgl32.Quat) {
	t.rotation = rotation
}
func (t *Transform) GetForward() mgl32.Vec3 {
	return t.rotation.Rotate(mgl32.Vec3{0, 0, -1})
}

func (t *Transform) GetForward2DCardinal() voxel.Int3 {
	forward := t.GetForward()
	gridForward := voxel.DirectionToGridInt3(forward)
	cardinalForward := gridForward.ToCardinalDirection()
	return cardinalForward
}
func (t *Transform) GetScale() mgl32.Vec3 {
	return t.scale
}

func (t *Transform) setYRotationAngle(angle float32) {
	t.rotation = mgl32.QuatRotate(angle, mgl32.Vec3{0, 1, 0})
	println(fmt.Sprintf("[Transform] setYRotationAngle for %s: %v", t.GetName(), angle))
}

func (t *Transform) SetForward2D(forward mgl32.Vec3) {
	t.setYRotationAngle(DirectionToAngleVec(forward))
}

func (t *Transform) SetForward(direction mgl32.Vec3) {
	t.rotation = mgl32.QuatBetweenVectors(mgl32.Vec3{0, 0, -1}, direction)
}

func (t *Transform) SetForward2DCardinal(forward voxel.Int3) {
	t.setYRotationAngle(DirectionToAngle(forward))
}

func (t *Transform) SetBlockPosition(position voxel.Int3) {
	t.SetPosition(position.ToBlockCenterVec3())
}
func (t *Transform) SetPosition(position mgl32.Vec3) {
	t.position = position
}

func (t *Transform) SetTopdownRotation(viewingAngle, rotationAngle float32) {
	t.rotation = mgl32.AnglesToQuat(mgl32.DegToRad(viewingAngle), mgl32.DegToRad(rotationAngle), 0, mgl32.XYZ)
}
func (t *Transform) SetLookAt(target mgl32.Vec3) {
	t.rotation = t.getLookAt(target)
}

func (t *Transform) getLookAt(target mgl32.Vec3) mgl32.Quat {
	origin := t.position
	upAxis := mgl32.Vec3{0, 1, 0}
	lookDirection := target.Sub(origin).Normalize()
	right := lookDirection.Cross(upAxis)
	up := right.Cross(lookDirection)
	lookAtMatrix := mgl32.QuatLookAtV(origin, target, up)
	return lookAtMatrix
}

func (t *Transform) SetOrbit(target mgl32.Vec3, angle float32) {
	lookAt := t.getLookAt(target)
	rotation := mgl32.QuatRotate(angle, mgl32.Vec3{0, 1, 0})
	t.rotation = rotation.Mul(lookAt)
}
