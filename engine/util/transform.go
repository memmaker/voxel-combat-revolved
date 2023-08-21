package util

import (
	"encoding/json"
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/voxel"
)

type Transform struct {
	translation mgl32.Vec3
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
		translation: mgl32.Vec3{0, 0, 0},
		rotation:    mgl32.QuatIdent(),
		scale:       mgl32.Vec3{1, 1, 1},
		nameOfOwner: name,
	}
}

func NewTransform(position mgl32.Vec3, rotation mgl32.Quat, scale mgl32.Vec3) *Transform {
	return &Transform{
		translation: position,
		rotation:    rotation,
		scale:       scale,
	}
}
func NewTransformFromTopDown(position mgl32.Vec3, viewingAngle, rotationAngle float32) *Transform {
	t := &Transform{
		translation: position,
		rotation:    mgl32.QuatIdent(),
		scale:       mgl32.Vec3{1, 1, 1},
	}
	t.SetTopdownRotation(viewingAngle, rotationAngle)
	return t
}
func NewTransformFromForward(position mgl32.Vec3, forward mgl32.Vec3) *Transform {
	t := &Transform{
		translation: position,
		rotation:    mgl32.QuatIdent(),
		scale:       mgl32.Vec3{1, 1, 1},
	}
	t.SetForward(forward)
	return t
}

func NewTransformFromLookAt(position, target, up mgl32.Vec3) *Transform {
	t := &Transform{
		translation: position,
		rotation:    mgl32.QuatIdent(),
		scale:       mgl32.Vec3{1, 1, 1},
	}
	t.SetLookAt(target, up)
	return t
}

func (t *Transform) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Position mgl32.Vec3 `json:"translation"`
		Rotation mgl32.Quat `json:"rotation"`
		Scale    mgl32.Vec3 `json:"scale"`
	}{
		Position: t.translation,
		Rotation: t.rotation,
		Scale:    t.scale,
	})
}

func (t *Transform) UnmarshalJSON(data []byte) error {
	var tmp struct {
		Position mgl32.Vec3 `json:"translation"`
		Rotation mgl32.Quat `json:"rotation"`
		Scale    mgl32.Vec3 `json:"scale"`
	}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	t.translation = tmp.Position
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
	return mgl32.Translate3D(t.translation.X(), t.translation.Y(), t.translation.Z())
}
func (t *Transform) GetPosition() mgl32.Vec3 {
	return t.translation
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
	t.translation = position
}

func (t *Transform) SetTopdownRotation(viewingAngle, rotationAngle float32) {
	t.rotation = mgl32.AnglesToQuat(mgl32.DegToRad(viewingAngle), mgl32.DegToRad(rotationAngle), 0, mgl32.XYZ)
}
func (t *Transform) SetLookAt(target, up mgl32.Vec3) {
	t.rotation = t.getLookAt(target, up)
}

func (t *Transform) SetLookAt2D(target mgl32.Vec3) {
	t.rotation = t.getLookAt(target, mgl32.Vec3{0, 1, 0})
}

func (t *Transform) getLookAt(target, up mgl32.Vec3) mgl32.Quat {
	lookAtMatrix := mgl32.QuatLookAtV(t.translation, target, up)
	return lookAtMatrix
}
