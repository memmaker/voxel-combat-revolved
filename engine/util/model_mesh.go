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
	holdAnimation    bool
}

func (m *CompoundMesh) ConvertVertexData(shader *glhf.Shader) {
	m.RootNode.ConvertVertexData(shader)
}

func (m *CompoundMesh) UpdateAnimations(deltaTime float64) bool {
	if m.holdAnimation {
		return false
	}
	scaledDeltaTime := deltaTime * m.animationSpeed
	animationFinished := m.RootNode.UpdateAnimation(scaledDeltaTime)
	if animationFinished && !m.loopAnimation {
		//ts := time.Now().Format("15:04:05.000")
		//println(fmt.Sprintf("[CompoundMesh] Animation %s finished at %s -> holding", m.currentAnimation, ts))
		m.holdAnimation = true
	}
	return animationFinished
}
func (m *CompoundMesh) getSamplerFrames(animationName string) [][]float32 {
	return m.SamplerFrames[animationName]
}
func (m *CompoundMesh) SetAnimationSpeed(newSpeed float64) {
	m.animationSpeed = newSpeed
}
func (m *CompoundMesh) Draw(shader *glhf.Shader) {
	m.RootNode.Draw(shader, m.textures)
}

func (m *CompoundMesh) DrawWithoutTransform(shader *glhf.Shader) {
	m.RootNode.DrawWithoutTransform(shader, m.textures)
}

func (m *CompoundMesh) SetProportionalScale(scaleFactor float64) {
	m.RootNode.Scale([3]float32{float32(scaleFactor), float32(scaleFactor), float32(scaleFactor)})
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
	name string
	// Hierarchy
	children        []*MeshNode
	parent          Transformer
	temporaryParent Transformer

	// rendering
	mesh      *SimpleMesh
	drawPairs []*DrawPair // will map a texture index to a list of vertex data

	// transformation
	translation  [3]float32
	quatRotation mgl32.Quat
	scale        [3]float32

	// t-pose (initial transform)
	initialTranslation [3]float32
	initialRotation    mgl32.Quat
	initialScale       [3]float32

	// animation
	animations       map[string]*SimpleAnimationData
	currentAnimation string
	animationTimer   float64

	currentTranslationFrame int
	currentRotationFrame    int
	currentScaleFrame       int

	// collision
	colliders              []Collider
	outOfTranslationFrames bool
	outOfRotationFrames    bool
	outOfScaleFrames       bool
	hidden                 bool

	samplerSource func(animationName string) [][]float32
}

func (m *MeshNode) GetName() string {
	return m.name
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
			m.colliders = append(m.colliders, &MeshCollider{VertexData: subMesh.VertexData, VertexCount: subMesh.VertexCount, VertexIndices: subMesh.Indices, TransformFunc: m.GetTransformMatrix})
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
			m.colliders = append(m.colliders, &MeshCollider{VertexData: subMesh.VertexData, VertexCount: subMesh.VertexCount, VertexIndices: subMesh.Indices, TransformFunc: m.GetTransformMatrix})
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
	shader.SetUniformAttr(2, m.GetTransformMatrix())
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

func (m *MeshNode) DrawWithoutTransform(shader *glhf.Shader, textures []*glhf.Texture) {
	if m.hidden {
		return
	}
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
		child.DrawWithoutTransform(shader, textures)
	}
}
func (m *MeshNode) GetTransformMatrix() mgl32.Mat4 {
	if m.temporaryParent != nil {
		externalParent := m.temporaryParent.GetTransformMatrix()
		// for the weapon, we needed this..this obviously won't work for parenting anything else..
		// also, we did attach part of the model to the cam
		camInversed := externalParent.Inv()
		offset := mgl32.Translate3D(0.2, -0.7, 0) // TODO: make this a parameter?
		camInversed = camInversed.Mul4(offset)
		return camInversed.Mul4(m.GetLocalMatrix())
	}
	if m.parent == nil {
		return m.GetLocalMatrix()
	}
	return m.parent.GetTransformMatrix().Mul4(m.GetLocalMatrix())
}
func (m *MeshNode) GetLocalMatrix() mgl32.Mat4 {
	translation := mgl32.Translate3D(m.translation[0], m.translation[1], m.translation[2])
	quaternion := m.quatRotation.Mat4()
	scale := mgl32.Scale3D(m.scale[0], m.scale[1], m.scale[2])
	return translation.Mul4(quaternion).Mul4(scale) // This actually represents S * R * T.. order is reversed because of how matrices work
}

func (m *CompoundMesh) ResetAnimations() {
	//println("[CompoundMesh] Resetting animations")
	m.RootNode.ResetAnimations()
}
func (m *CompoundMesh) StopAnimations() {
	//println("[CompoundMesh] Stopping animations")
	m.holdAnimation = true
}
func (m *CompoundMesh) SetAnimationLoop(animationName string, speedFactor float64) {
	//println(fmt.Sprintf("[CompoundMesh] Setting animation loop %s", animationName))
	m.SetAnimationSpeed(speedFactor)
	m.currentAnimation = animationName
	m.RootNode.SetAnimation(animationName)
	m.loopAnimation = true
	m.holdAnimation = false
}
func (m *CompoundMesh) SetAnimation(animationName string, speedFactor float64) {
	//println(fmt.Sprintf("[CompoundMesh] Setting animation %s", animationName))
	m.SetAnimationSpeed(speedFactor)
	m.currentAnimation = animationName
	m.RootNode.SetAnimation(animationName)
	// need millisec accuracy
	//ts := time.Now().Format("15:04:05.000")
	//println(fmt.Sprintf("[CompoundMesh] Animation started at %s", ts))
	m.loopAnimation = false
	m.holdAnimation = false
}

func (m *CompoundMesh) SetAnimationPose(animation string) {
	//println(fmt.Sprintf("[CompoundMesh] Setting animation pose to %s", animation))
	m.RootNode.SetAnimationPose(animation)
	m.holdAnimation = true
}
func (m *CompoundMesh) GetAnimationDebugString() string {
	return m.RootNode.GetAnimationDebugString(0)
}

func (m *CompoundMesh) HideBone(name string) {
	m.RootNode.HideBone(name)
}

func (m *CompoundMesh) IsHoldingAnimation() bool {
	return m.holdAnimation
}

func (m *CompoundMesh) HideChildrenOfBoneExcept(parentName string, exception string) {
	m.RootNode.HideChildrenOfBoneExcept(false, parentName, exception)
}

func (m *CompoundMesh) GetAnimationName() string {
	return m.currentAnimation
}

func (m *CompoundMesh) HasBone(name string) bool {
	return m.RootNode.HasBone(name)
}

func (m *MeshNode) SetAnimationPose(name string) {
	for _, child := range m.children {
		child.SetAnimationPose(name)
	}
	if _, ok := m.animations[name]; ok {
		m.currentAnimation = name
		m.ResetAnimation()
		m.InitAnimationPose()
	} else {
		m.currentAnimation = ""
		m.ResetToInitialTransform()
	}
}
func (m *MeshNode) SetAnimation(name string) {
	for _, child := range m.children {
		child.SetAnimation(name)
	}
	if m.currentAnimation == name {
		return
	}
	if _, ok := m.animations[name]; ok {
		m.currentAnimation = name
		m.InitAnimationPose()
	} else {
		m.currentAnimation = ""
		m.ResetToInitialTransform()
	}
	m.ResetAnimation()

}

func (m *MeshNode) ResetAnimation() {
	m.animationTimer = 0
	m.currentTranslationFrame = 0
	m.currentRotationFrame = 0
	m.currentScaleFrame = 0
	m.outOfTranslationFrames = false
	m.outOfRotationFrames = false
	m.outOfScaleFrames = false
}
func (m *MeshNode) InitAnimationPose() {
	if m.IsCurrentAnimationValid() {
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
func (m *MeshNode) UpdateAnimation(deltaTime float64) bool {
	animationFinished := false
	if m.IsCurrentAnimationValid() {
		m.animationTimer += deltaTime
		animation := m.GetCurrentAnimation()

		// translate the mesh
		if len(animation.TranslationFrames) > 0 {
			translationFrameTimes := m.samplerSource(m.currentAnimation)[animation.TranslationSamplerIndex]
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
			rotationFrameTimes := m.samplerSource(m.currentAnimation)[animation.RotationSamplerIndex]
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
			scaleFrameTimes := m.samplerSource(m.currentAnimation)[animation.ScaleSamplerIndex]
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
		childAnimationFinished := child.UpdateAnimation(deltaTime)
		if childAnimationFinished {
			animationFinished = true
		}
	}

	return animationFinished
}

func (m *MeshNode) IsCurrentAnimationValid() bool {
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
	if m.name == name {
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
		colliderName := m.parent.GetName()
		for _, collider := range m.colliders {
			collider.SetName(colliderName)
			result = append(result, collider)
		}
	}

	if !m.hidden {
		for _, child := range m.children {
			result = append(result, child.GetColliders()...)
		}
	}
	return result
}

func (m *MeshNode) GetFront() mgl32.Vec3 {
	return m.quatRotation.Rotate(mgl32.Vec3{1, 0, 0})
}
func (m *MeshNode) ResetAnimations() {
	m.currentAnimation = ""
	m.ResetAnimation()
	for _, child := range m.children {
		child.ResetAnimations()
	}
}

func (m *MeshNode) GetAnimationDebugString(hierarchyLevel int) string {
	padding := ""
	for i := 0; i < hierarchyLevel; i++ {
		padding += "  "
	}
	result := ""
	result += padding + fmt.Sprintf("Node: %s\n", m.name)
	if m.IsCurrentAnimationValid() {
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
	if m.name == name {
		m.hidden = true
	}
	for _, child := range m.children {
		child.HideBone(name)
	}
}

func (m *MeshNode) HideChildrenOfBoneExcept(isChild bool, name string, exception string) {
	if isChild && m.name != exception {
		m.hidden = true
		println(fmt.Sprintf("[MeshNode] Hiding node %s", m.name))
		return
	}
	for _, child := range m.children {
		child.HideChildrenOfBoneExcept(m.name == name, name, exception)
	}
}

type Transformer interface {
	GetTransformMatrix() mgl32.Mat4
	GetName() string
}

type TransRotator interface {
	GetPosition() mgl32.Vec3
	GetRotation() mgl32.Quat
}

func (m *MeshNode) SetTempParent(transform Transformer) {
	m.temporaryParent = transform
}

func (m *MeshNode) SetParent(transform Transformer) {
	m.parent = transform
}

func (m *MeshNode) GetAnimationName() string {
	return m.currentAnimation
}

func (m *MeshNode) SetSamplerSource(source func(animationName string) [][]float32) {
	m.samplerSource = source
}

func (m *MeshNode) SetInitialTranslate(translate [3]float32) {
	m.initialTranslation = translate
	m.Translate(translate)
}

func (m *MeshNode) SetInitialRotate(rotate [4]float32) {
	m.initialRotation = mgl32.Quat{V: mgl32.Vec3{rotate[0], rotate[1], rotate[2]}, W: rotate[3]}
	m.Rotate(rotate)

}

func (m *MeshNode) SetInitialScale(scale [3]float32) {
	m.initialScale = scale
	m.Scale(scale)
}

func (m *MeshNode) ResetToInitialTransform() {
	m.translation = m.initialTranslation
	m.quatRotation = m.initialRotation
	m.scale = m.initialScale
}

func (m *MeshNode) GetWorldPosition() mgl32.Vec3 {
	localPos := mgl32.Vec3{m.translation[0], m.translation[1], m.translation[2]}
	worldPos := m.GetTransformMatrix().Mul4x1(localPos.Vec4(1.0)).Vec3()
	return worldPos
}

func (m *MeshNode) HasBone(name string) bool {
	if m.name == name {
		return true
	}
	for _, child := range m.children {
		if child.HasBone(name) {
			return true
		}
	}
	return false
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
