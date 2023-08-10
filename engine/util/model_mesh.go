package util

import (
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/memmaker/battleground/engine/glhf"
)

type DrawPair struct {
	TextureIndex uint32
	VertexData   *glhf.VertexSlice[glhf.GlFloat]
}

type CompoundMesh struct {
	SamplerFrames    map[string][][]float32 // will map a sampler index to a list of keyframe times
	RootNode         *MeshNode
	textures         []*glhf.Texture // will map a texture index to a texture
	animationTimer   float64
	currentAnimation string
	animationSpeed   float64
	loopAnimation    bool
}

func (m *CompoundMesh) ConvertVertexData(shader *glhf.Shader) {
	m.RootNode.ConvertVertexData(shader)
}

func (m *CompoundMesh) UpdateAnimations(deltaTime float64) bool {
	scaledDeltaTime := deltaTime * m.animationSpeed
	animationFinished := m.RootNode.UpdateAnimation(m.SamplerFrames[m.currentAnimation], scaledDeltaTime)
	if animationFinished && !m.loopAnimation {
		m.StopAnimations()
	}
	return animationFinished
}
func (m *CompoundMesh) SetAnimationSpeed(newSpeed float64) {
	m.animationSpeed = newSpeed
}
func (m *CompoundMesh) Draw(shader *glhf.Shader) {
	m.RootNode.Draw(shader, m.textures)
}

func (m *CompoundMesh) SetProportionalScale(scaleFactor float64) {
	m.RootNode.Scale([3]float32{float32(scaleFactor), float32(scaleFactor), float32(scaleFactor)})
}

func (m *CompoundMesh) SetYRotationAngle(angle float32) {
	m.RootNode.SetYRotationAngle(angle)
}

func (m *CompoundMesh) GetNodeByName(nodeName string) *MeshNode {
	return m.RootNode.GetNodeByName(nodeName)
}

func (m *CompoundMesh) SetTexture(atIndex int, newTexture *glhf.Texture) {
	if atIndex >= len(m.textures) {
		// resize the texture array
		newTextures := make([]*glhf.Texture, atIndex+1)
		copy(newTextures, m.textures)
		m.textures = newTextures
	}
	m.textures[atIndex] = newTexture
}

func (m *CompoundMesh) SetPosition(pos mgl32.Vec3) {
	m.RootNode.Translate(pos)
}

func (m *CompoundMesh) GetPosition() mgl32.Vec3 {
	return m.RootNode.translation
}

type SimpleMesh struct {
	SubMeshes []*SubMesh
}
type SimpleAnimationData struct {
	TranslationSamplerIndex uint32
	TranslationFrames       [][3]float32 // will map a sampler index to a list translation keyframes

	RotationSamplerIndex uint32
	RotationFrames       [][4]float32

	ScaleSamplerIndex uint32
	ScaleFrames       [][3]float32
}
type MeshNode struct {
	Name string
	// Hierarchy
	children []*MeshNode
	parent   *MeshNode

	// rendering
	mesh      *SimpleMesh
	drawPairs []*DrawPair // will map a texture index to a list of vertex data

	// transformation
	translation  [3]float32
	quatRotation mgl32.Quat
	scale        [3]float32

	// animation
	animations       map[string]*SimpleAnimationData
	currentAnimation string
	animationTimer   float64
	holdAnimation    bool

	currentTranslationFrame int
	currentRotationFrame    int
	currentScaleFrame       int

	// collision
	colliders              []Collider
	outOfTranslationFrames bool
	outOfRotationFrames    bool
	outOfScaleFrames       bool
	hidden                 bool
}

func (m *MeshNode) HasMesh() bool {
	return m.mesh != nil || m.drawPairs != nil
}

func (m *MeshNode) GetCurrentAnimation() *SimpleAnimationData {
	return m.animations[m.currentAnimation]
}

func (m *MeshNode) ConvertVertexData(shader *glhf.Shader) {
	if m.mesh != nil {
		//pairs := make(map[uint32]*glhf.VertexSlice)
		for _, subMesh := range m.mesh.SubMeshes {
			m.drawPairs = append(m.drawPairs, &DrawPair{TextureIndex: subMesh.TextureIndex, VertexData: subMesh.ToVertexSlice(shader)})
			m.colliders = append(m.colliders, &MeshCollider{VertexData: subMesh.VertexData, VertexCount: subMesh.VertexCount, VertexIndices: subMesh.Indices, TransformFunc: m.GlobalMatrix})
		}
		m.mesh = nil
	}
	for _, child := range m.children {
		child.ConvertVertexData(shader)
	}
}

func (m *MeshNode) CreateColliders() {
	if m.mesh != nil {
		for _, subMesh := range m.mesh.SubMeshes {
			m.colliders = append(m.colliders, &MeshCollider{VertexData: subMesh.VertexData, VertexCount: subMesh.VertexCount, VertexIndices: subMesh.Indices, TransformFunc: m.GlobalMatrix})
		}
		m.mesh = nil
	}
	for _, child := range m.children {
		child.CreateColliders()
	}
}

func (m *MeshNode) Draw(shader *glhf.Shader, textures []*glhf.Texture) {
	if m.hidden {
		return
	}
	shader.SetUniformAttr(2, m.GlobalMatrix())

	for _, pair := range m.drawPairs {
		textureIndex := pair.TextureIndex
		meshGroup := pair.VertexData
		textures[textureIndex].Begin()
		meshGroup.Begin()
		meshGroup.Draw()
		meshGroup.End()
		textures[textureIndex].End()
	}
	for _, child := range m.children {
		child.Draw(shader, textures)
	}
}

func (m *MeshNode) SetYRotationAngle(angle float32) {
	m.quatRotation = mgl32.QuatRotate(angle, mgl32.Vec3{0, 1, 0})
}

func (m *MeshNode) SetXRotationAngle(angle float32) {
	m.quatRotation = mgl32.QuatRotate(angle, mgl32.Vec3{1, 0, 0})
}

func (m *MeshNode) SetZRotationAngle(angle float32) {
	m.quatRotation = mgl32.QuatRotate(angle, mgl32.Vec3{0, 0, 1})
}
func (m *MeshNode) GlobalMatrix() mgl32.Mat4 {
	if m.parent == nil {
		return m.LocalMatrix()
	}
	return m.parent.GlobalMatrix().Mul4(m.LocalMatrix())
}
func (m *MeshNode) LocalMatrix() mgl32.Mat4 {
	translation := mgl32.Translate3D(m.translation[0], m.translation[1], m.translation[2])
	quaternion := m.quatRotation.Mat4()
	scale := mgl32.Scale3D(m.scale[0], m.scale[1], m.scale[2])
	return translation.Mul4(quaternion).Mul4(scale)
}
func (m *CompoundMesh) SetAnimation(animationName string) {
	m.currentAnimation = animationName
	m.RootNode.ChangeAnimation(animationName)
}

func (m *CompoundMesh) GetFront() mgl32.Vec3 {
	return m.RootNode.GetFront()
}

func (m *CompoundMesh) SetXRotationAngle(angle float32) {
	m.RootNode.SetXRotationAngle(angle)
}

func (m *CompoundMesh) StopAnimations() {
	m.RootNode.StopAnimations()
}
func (m *CompoundMesh) StartAnimationLoop(animationName string, speedFactor float64) {
	m.SetAnimationSpeed(speedFactor)
	m.SetAnimation(animationName)
	m.loopAnimation = true
}

func (m *CompoundMesh) PlayAnimation(animationName string, speedFactor float64) {
	m.SetAnimationSpeed(speedFactor)
	m.SetAnimation(animationName)
	m.loopAnimation = false
}

func (m *CompoundMesh) GetAnimationDebugString() string {
	return m.RootNode.GetAnimationDebugString(0)
}

func (m *CompoundMesh) HideBone(name string) {
	m.RootNode.HideBone(name)
}

func (m *CompoundMesh) HideChildrenOfBoneExcept(parentName string, exception string) {
	m.RootNode.HideChildrenOfBoneExcept(false, parentName, exception)
}

func (m *CompoundMesh) SetAnimationPose(animation string) {
	m.RootNode.ChangeAnimation(animation)
	m.RootNode.holdAnimation = true
}
func (m *MeshNode) ChangeAnimation(name string) {
	for _, child := range m.children {
		child.ChangeAnimation(name)
	}
	if m.currentAnimation == name {
		return
	}
	if _, ok := m.animations[name]; ok {
		m.currentAnimation = name
		m.ResetAnimation()
		m.InitAnimationPose()
	} else {
		m.currentAnimation = ""
	}
}

func (m *MeshNode) ResetAnimation() {
	m.animationTimer = 0
	m.currentTranslationFrame = 0
	m.currentRotationFrame = 0
	m.currentScaleFrame = 0
	m.outOfTranslationFrames = false
	m.outOfRotationFrames = false
	m.outOfScaleFrames = false
	m.holdAnimation = false
}
func (m *MeshNode) InitAnimationPose() {
	if m.IsAnimated() {
		animation := m.GetCurrentAnimation()
		if len(animation.TranslationFrames) > 0 {
			m.Translate(animation.TranslationFrames[0])
		}
		if len(animation.RotationFrames) > 0 {
			m.Rotate(animation.RotationFrames[0])
		}
		if len(animation.ScaleFrames) > 0 {
			m.Scale(animation.ScaleFrames[0])
		}
	}
	for _, child := range m.children {
		child.InitAnimationPose()
	}
}
func (m *MeshNode) UpdateAnimation(samplerFrames [][]float32, deltaTime float64) bool {
	if m.holdAnimation {
		return false
	}
	animationFinished := false
	if m.IsAnimated() {
		m.animationTimer += deltaTime
		animation := m.GetCurrentAnimation()

		// translate the mesh
		if len(animation.TranslationFrames) > 0 {
			translationFrameTimes := samplerFrames[animation.TranslationSamplerIndex]
			nextTranslationFrameIndex := (m.currentTranslationFrame + 1) % len(translationFrameTimes)
			nextKeyFrameTime := translationFrameTimes[nextTranslationFrameIndex]
			if m.animationTimer >= float64(nextKeyFrameTime) {
				m.currentTranslationFrame = m.currentTranslationFrame + 1
				if m.currentTranslationFrame >= len(translationFrameTimes) {
					m.outOfTranslationFrames = true
					m.currentTranslationFrame = 0
				} else {
					translation := animation.TranslationFrames[m.currentTranslationFrame]
					m.Translate(translation)
				}
			} else if m.currentTranslationFrame != nextTranslationFrameIndex { // lerp between keyframes
				currentFrameTime := translationFrameTimes[m.currentTranslationFrame]
				nextTranslationFrameTime := translationFrameTimes[nextTranslationFrameIndex]
				deltaTimeBetweenFrames := nextTranslationFrameTime - currentFrameTime
				timeSinceCurrentKeyframe := m.animationTimer - float64(currentFrameTime)
				percentageBetweenFrames := mgl64.Clamp(timeSinceCurrentKeyframe/float64(deltaTimeBetweenFrames), 0, 1)

				currentTranslationKeyFrame := animation.TranslationFrames[m.currentTranslationFrame]
				nextTranslationKeyFrame := animation.TranslationFrames[nextTranslationFrameIndex]

				lerpedPos := Lerp3(currentTranslationKeyFrame, nextTranslationKeyFrame, percentageBetweenFrames)
				//println(fmt.Sprintf("lerpedPos: %v", lerpedPos))
				m.Translate(lerpedPos)
			} else {
				m.Translate(animation.TranslationFrames[m.currentTranslationFrame])
			}
		} else {
			m.outOfTranslationFrames = true
		}

		// rotate the mesh
		if len(animation.RotationFrames) > 0 {
			rotationFrameTimes := samplerFrames[animation.RotationSamplerIndex]
			nextRotationFrameIndex := (m.currentRotationFrame + 1) % len(rotationFrameTimes)
			nextKeyFrameTime := rotationFrameTimes[nextRotationFrameIndex]
			if m.animationTimer >= float64(nextKeyFrameTime) { // hit a keyframe
				m.currentRotationFrame = m.currentRotationFrame + 1
				if m.currentRotationFrame >= len(rotationFrameTimes) {
					m.outOfRotationFrames = true
					m.currentRotationFrame = 0
				} else {
					m.Rotate(animation.RotationFrames[m.currentRotationFrame])
				}
			} else if m.currentRotationFrame != nextRotationFrameIndex { // lerp between keyframes
				currentKeyFrameTime := rotationFrameTimes[m.currentRotationFrame]
				nextRotationFrameTime := rotationFrameTimes[nextRotationFrameIndex]
				deltaTimeBetweenFrames := nextRotationFrameTime - currentKeyFrameTime
				timeSinceCurrentKeyframe := m.animationTimer - float64(currentKeyFrameTime)
				percentageBetweenFrames := mgl64.Clamp(timeSinceCurrentKeyframe/float64(deltaTimeBetweenFrames), 0, 1)

				currentRotationKeyFrame := animation.RotationFrames[m.currentRotationFrame]
				nextRotationKeyFrame := animation.RotationFrames[nextRotationFrameIndex]
				lerpedRotation := LerpQuat(currentRotationKeyFrame, nextRotationKeyFrame, percentageBetweenFrames)
				m.Rotate(lerpedRotation)
			} else {
				m.Rotate(animation.RotationFrames[m.currentRotationFrame])
			}
		} else {
			m.outOfRotationFrames = true
		}
		// scale the mesh
		if len(animation.ScaleFrames) > 0 {
			scaleFrameTimes := samplerFrames[animation.ScaleSamplerIndex]
			nextScaleFrameIndex := (m.currentScaleFrame + 1) % len(scaleFrameTimes)
			nextKeyFrameTime := scaleFrameTimes[nextScaleFrameIndex]
			if m.animationTimer >= float64(nextKeyFrameTime) {
				m.currentScaleFrame = m.currentScaleFrame + 1
				if m.currentScaleFrame >= len(scaleFrameTimes) {
					m.outOfScaleFrames = true
					m.currentScaleFrame = 0
				} else {
					m.Scale(animation.ScaleFrames[m.currentScaleFrame])
				}
			} else if m.currentScaleFrame != nextScaleFrameIndex { // lerp between keyframes
				currentKeyFrameTime := scaleFrameTimes[m.currentScaleFrame]
				nextScaleFrameTime := scaleFrameTimes[nextScaleFrameIndex]
				deltaTimeBetweenFrames := nextScaleFrameTime - currentKeyFrameTime
				timeSinceCurrentKeyframe := m.animationTimer - float64(currentKeyFrameTime)
				percentageBetweenFrames := mgl64.Clamp(timeSinceCurrentKeyframe/float64(deltaTimeBetweenFrames), 0, 1)

				currentScaleKeyFrame := animation.ScaleFrames[m.currentScaleFrame]
				nextScaleKeyFrame := animation.ScaleFrames[nextScaleFrameIndex]

				lerpedScale := Lerp3(currentScaleKeyFrame, nextScaleKeyFrame, percentageBetweenFrames)

				m.Scale(lerpedScale)
			} else {
				m.Scale(animation.ScaleFrames[m.currentScaleFrame])
			}
		} else {
			m.outOfScaleFrames = true
		}

		if m.outOfTranslationFrames && m.outOfRotationFrames && m.outOfScaleFrames {
			m.ResetAnimation()
			animationFinished = true
		}
	}
	for _, child := range m.children {
		childAnimationFinished := child.UpdateAnimation(samplerFrames, deltaTime)
		if childAnimationFinished {
			animationFinished = true
		}
	}
	return animationFinished
}

func (m *MeshNode) IsAnimated() bool {
	if _, ok := m.animations[m.currentAnimation]; ok {
		return true
	}
	return false
}

func (m *MeshNode) Rotate(keyFrame [4]float32) {
	m.quatRotation = mgl32.Quat{V: mgl32.Vec3{keyFrame[0], keyFrame[1], keyFrame[2]}, W: keyFrame[3]}
}

func (m *MeshNode) Scale(keyFrame [3]float32) {
	m.scale = keyFrame
}

func (m *MeshNode) Translate(keyFrame [3]float32) {
	m.translation = keyFrame
}

func (m *MeshNode) GetNodeByName(name string) *MeshNode {
	if m.Name == name {
		return m
	}
	for _, child := range m.children {
		node := child.GetNodeByName(name)
		if node != nil {
			return node
		}
	}
	return nil
}

func (m *MeshNode) GetColliders() []Collider {
	var result []Collider

	if len(m.colliders) > 0 && !m.hidden {
		colliderName := m.parent.Name
		for _, collider := range m.colliders {
			collider.SetName(colliderName)
			result = append(result, collider)
		}
	}

	for _, child := range m.children {
		result = append(result, child.GetColliders()...)
	}

	return result
}

func (m *MeshNode) GetFront() mgl32.Vec3 {
	return m.quatRotation.Rotate(mgl32.Vec3{1, 0, 0})
}
func (m *MeshNode) StopAnimations() {
	m.currentAnimation = ""
	m.ResetAnimation()
	for _, child := range m.children {
		child.StopAnimations()
	}
}

func (m *MeshNode) GetAnimationDebugString(hierarchyLevel int) string {
	// add padding depending on the hierarchy level
	padding := ""
	for i := 0; i < hierarchyLevel; i++ {
		padding += "  "
	}
	result := ""
	result += padding + fmt.Sprintf("Node: %s\n", m.Name)
	if m.IsAnimated() {
		result += padding + fmt.Sprintf("Translation: %v\n", m.translation)
		result += padding + fmt.Sprintf("Rotation: %v\n", m.quatRotation)
		result += padding + fmt.Sprintf("Scale: %v\n", m.scale)
		result += padding + fmt.Sprintf("Animation: %s\n", m.currentAnimation)
	} else {
		result += padding + fmt.Sprintf("Animation: none\n")
	}
	for _, child := range m.children {
		result += child.GetAnimationDebugString(hierarchyLevel + 1)
	}
	return result
}

func (m *MeshNode) HideBone(name string) {
	if m.Name == name {
		m.hidden = true
	}
	for _, child := range m.children {
		child.HideBone(name)
	}
}

func (m *MeshNode) HideChildrenOfBoneExcept(isChild bool, name string, exception string) {
	if isChild && m.Name != exception {
		m.hidden = true
		return
	}
	for _, child := range m.children {
		child.HideChildrenOfBoneExcept(m.Name == name, name, exception)
	}
}

type SubMesh struct {
	TextureIndex uint32
	VertexData   []glhf.GlFloat
	VertexCount  int
	Indices      []uint32
}

func (m SubMesh) ToVertexSlice(shader *glhf.Shader) *glhf.VertexSlice[glhf.GlFloat] {

	var slice *glhf.VertexSlice[glhf.GlFloat]
	if len(m.Indices) > 0 {
		slice = glhf.MakeIndexedVertexSlice(shader, m.VertexCount, m.VertexCount, m.Indices)
	} else {
		slice = glhf.MakeVertexSlice(shader, m.VertexCount, m.VertexCount)
	}
	slice.Begin()
	slice.SetVertexData(m.VertexData)
	slice.End()

	return slice
}
