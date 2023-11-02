package util

import (
	"fmt"
	"github.com/go-gl/gl/v4.1-core/gl"
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

func (m *CompoundMesh) UploadVertexData(shader *glhf.Shader) {
	m.RootNode.ConvertVertexData(shader)
}

func (m *CompoundMesh) UpdateAnimations(deltaTime float64) bool {
	if m.holdAnimation {
		return false
	}
	duration := m.GetAnimationDuration()
	scaledDeltaTime := deltaTime * m.animationSpeed
	m.animationTimer += scaledDeltaTime
	animationFinished := m.animationTimer > duration
	if animationFinished {
		if m.loopAnimation {
			//loopDelta := math.Floor(m.animationTimer / duration)
			m.RootNode.UpdateAnimation(m.animationTimer)
			m.animationTimer -= duration //* loopDelta
			m.ResetAnimations()
			return animationFinished
		} else {
			m.holdAnimation = true
			return animationFinished
		}
	}
	// animationTimer will always be <= duration here
	// and also animationTimer > 0
	m.RootNode.UpdateAnimation(m.animationTimer)

	return animationFinished
}
func (m *CompoundMesh) getSamplerFrames(animationName string) [][]float32 {
	return m.SamplerFrames[animationName]
}
func (m *CompoundMesh) SetAnimationSpeed(newSpeed float64) {
	m.animationSpeed = newSpeed
}
func (m *CompoundMesh) Draw(shader *glhf.Shader, modelTransformUniformIndex int) {
	m.RootNode.Draw(shader, modelTransformUniformIndex, m.textures)
}

func (m *CompoundMesh) GetNodeByName(nodeName string) (*MeshNode, bool) {
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

	Duration float32
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

	// animation frames (trans, rot, scale)
	currentFrame [3]int

	// collision
	colliders              []Collider
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
	vertexFormatComponents := uint32(shader.VertexFormat().Size() / 4)
	if m.mesh != nil {
		for _, subMesh := range m.mesh.SubMeshes {
			m.drawPairs = append(m.drawPairs, &DrawPair{TextureIndex: subMesh.TextureIndex, VertexData: subMesh.ToVertexSlice(shader)})
			m.colliders = append(m.colliders, &MeshCollider{VertexData: subMesh.VertexData, VertexCount: subMesh.VertexCount, VertexIndices: subMesh.Indices, VertexFormatComponents: vertexFormatComponents, TransformFunc: m.GetTransformMatrix})
		}
		m.mesh = nil
	}
	for _, child := range m.children {
		child.ConvertVertexData(shader)
	}
}

func (m *MeshNode) CreateColliders() {
	vertexFormatComponents := uint32(11)
	if m.mesh != nil {
		for _, subMesh := range m.mesh.SubMeshes {
			m.colliders = append(m.colliders, &MeshCollider{VertexData: subMesh.VertexData, VertexCount: subMesh.VertexCount, VertexIndices: subMesh.Indices, VertexFormatComponents: vertexFormatComponents, TransformFunc: m.GetTransformMatrix})
		}
		m.mesh = nil
	}
	for _, child := range m.children {
		child.CreateColliders()
	}
}

func (m *MeshNode) Draw(shader *glhf.Shader, modelTransformUniformIndex int, textures []*glhf.Texture) {
	if m.hidden {
		return
	}
	shader.SetUniformAttr(modelTransformUniformIndex, m.GetTransformMatrix())
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
		child.Draw(shader, modelTransformUniformIndex, textures)
	}
}

func (m *MeshNode) GetTransformMatrix() mgl32.Mat4 {
	if m.temporaryParent != nil { // currently only used by the fps camera for attaching the arms
		externalParent := m.temporaryParent.GetTransformMatrix()
		offset := mgl32.Translate3D(0.2, -0.7, 0) // TODO: make this a parameter?
		externalParent = externalParent.Mul4(offset)
		return externalParent.Mul4(m.GetLocalMatrix())
	}
	if m.parent == nil {
		return m.GetLocalMatrix()
	}
	return m.parent.GetTransformMatrix().Mul4(m.GetLocalMatrix())
}
func (m *MeshNode) GetLocalMatrix() mgl32.Mat4 {
	translation := mgl32.Translate3D(m.translation[0], m.translation[1], m.translation[2])
	rotation := m.quatRotation.Mat4()
	scale := mgl32.Scale3D(m.scale[0], m.scale[1], m.scale[2])
	return translation.Mul4(rotation).Mul4(scale) // This actually represents S * R * T.. order is reversed because of how matrices work
}

func (m *CompoundMesh) ResetAnimations() {
	//println("[CompoundMesh] Resetting animations")
	//m.animationTimer = 0
	m.RootNode.ResetAnimations()
}
func (m *CompoundMesh) StopAnimations() {
	//println("[CompoundMesh] Stopping animations")
	m.holdAnimation = true
}
func (m *CompoundMesh) SetAnimationLoop(animationName string, speedFactor float64) {
	m.SetAnimationSpeed(speedFactor)
	if m.currentAnimation != animationName {
		m.animationTimer = 0
		LogAnimationDebug(fmt.Sprintf("[CompoundMesh] Changing animation loop %s", animationName))
	}
	m.currentAnimation = animationName
	m.RootNode.SetAnimation(animationName)
	m.loopAnimation = true
	m.holdAnimation = false
}
func (m *CompoundMesh) SetAnimation(animationName string, speedFactor float64) {
	m.SetAnimationSpeed(speedFactor)
	if m.currentAnimation != animationName {
		m.animationTimer = 0
		LogAnimationDebug(fmt.Sprintf("[CompoundMesh] Changing animation %s", animationName))
	}
	m.currentAnimation = animationName
	m.RootNode.SetAnimation(animationName)
	// need millisec accuracy
	//ts := time.Now().Format("15:04:05.000")
	//println(fmt.Sprintf("[CompoundMesh] Animation started at %s", ts))
	m.loopAnimation = false
	m.holdAnimation = false
}

func (m *CompoundMesh) SetAnimationPose(animation string) {
	LogAnimationDebug(fmt.Sprintf("[CompoundMesh] Setting animation pose to %s", animation))
	m.animationTimer = 0
	m.currentAnimation = animation
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

func (m *CompoundMesh) GetAnimationDuration() float64 {
	return m.RootNode.GetAnimationDuration()
}

func (m *MeshNode) SetAnimationPose(name string) {
	for _, child := range m.children {
		child.SetAnimationPose(name)
	}
	m.ResetToInitialTransform()
	if _, ok := m.animations[name]; ok {
		m.currentAnimation = name
		m.ResetAnimationFrames()
		m.InitAnimationPose()
	} else {
		m.currentAnimation = ""
		m.ResetAnimationFrames()
	}
}
func (m *MeshNode) SetAnimation(name string) {
	for _, child := range m.children {
		child.SetAnimation(name)
	}
	if m.currentAnimation == name {
		return
	}
	m.ResetToInitialTransform()
	if _, ok := m.animations[name]; ok {
		m.currentAnimation = name
		m.InitAnimationPose()
	} else {
		m.currentAnimation = ""
	}
	m.ResetAnimationFrames()

}

func (m *MeshNode) ResetAnimationFrames() {
	m.currentFrame = [3]int{0, 0, 0}
}
func (m *MeshNode) InitAnimationPose() {
	if m.IsCurrentAnimationValid() {
		animation := m.GetCurrentAnimation()
		if len(animation.TranslationFrames) > 0 {
			m.setTranslation(animation.TranslationFrames[0])
		}
		if len(animation.RotationFrames) > 0 {
			m.setRotation(animation.RotationFrames[0])
		}
		if len(animation.ScaleFrames) > 0 {
			m.setScale(animation.ScaleFrames[0])
		}
	}
	for _, child := range m.children {
		child.InitAnimationPose()
	}
}

func KeyframeAnimation[T any](currentTime float64, currentFrame *int, frames []T, frameTimes []float32, lerp func(T, T, float64) T, setter func(T)) {
	frameCount := len(frames)
	if frameCount == 0 {
		return
	}
	if frameCount == 1 {
		setter(frames[0])
		return
	}
	firstFrameTime := frameTimes[0]
	lastFrameTime := frameTimes[frameCount-1]

	if currentTime <= float64(firstFrameTime) { // before first keyframe
		*currentFrame = 0
		setter(frames[(*currentFrame)])
		return
	}

	if currentTime >= float64(lastFrameTime) { // after last keyframe
		*currentFrame = frameCount - 1
		setter(frames[(*currentFrame)])
		return
	}

	nextFrameIndex := ((*currentFrame) + 1) % frameCount
	nextFrameTime := frameTimes[nextFrameIndex]

	if currentTime >= float64(nextFrameTime) { // new keyframe
		*currentFrame = nextFrameIndex
		setter(frames[(*currentFrame)])
		return
	}
	// between keyframes
	if nextFrameIndex == 0 {
		println("nextFrameIndex == 0")
	}
	{
		currentFrameTime := frameTimes[(*currentFrame)]
		deltaTimeBetweenFrames := nextFrameTime - currentFrameTime
		timeSinceCurrentKeyframe := currentTime - float64(currentFrameTime)
		percentageBetweenFrames := mgl64.Clamp(timeSinceCurrentKeyframe/float64(deltaTimeBetweenFrames), 0, 1)

		currentKeyFrame := frames[(*currentFrame)]
		nextKeyFrame := frames[nextFrameIndex]

		lerpedValue := lerp(currentKeyFrame, nextKeyFrame, percentageBetweenFrames)
		//println(fmt.Sprintf("lerpedValue: %v", lerpedValue))
		setter(lerpedValue)
		return
	}
}

func (m *MeshNode) UpdateAnimation(animationTime float64) {
	if m.IsCurrentAnimationValid() {
		animation := m.GetCurrentAnimation()
		// translate the mesh
		KeyframeAnimation(
			animationTime,
			&m.currentFrame[0],
			animation.TranslationFrames,
			m.samplerSource(m.currentAnimation)[animation.TranslationSamplerIndex],
			Lerp3f,
			m.setTranslation,
		)
		// rotate the mesh
		KeyframeAnimation(
			animationTime,
			&m.currentFrame[1],
			animation.RotationFrames,
			m.samplerSource(m.currentAnimation)[animation.RotationSamplerIndex],
			LerpQuat,
			m.setRotation,
		)
		// scale the mesh
		KeyframeAnimation(
			animationTime,
			&m.currentFrame[2],
			animation.ScaleFrames,
			m.samplerSource(m.currentAnimation)[animation.ScaleSamplerIndex],
			Lerp3f,
			m.setScale,
		)
	}
	for _, child := range m.children {
		child.UpdateAnimation(animationTime)
	}
}

func (m *MeshNode) IsCurrentAnimationValid() bool {
	if _, ok := m.animations[m.currentAnimation]; ok {
		return true
	}
	return false
}

func (m *MeshNode) setRotation(keyFrame [4]float32) {
	m.quatRotation = mgl32.Quat{V: mgl32.Vec3{keyFrame[0], keyFrame[1], keyFrame[2]}, W: keyFrame[3]}
}

func (m *MeshNode) setScale(keyFrame [3]float32) {
	m.scale = keyFrame
}

func (m *MeshNode) setTranslation(keyFrame [3]float32) {
	m.translation = keyFrame
}

func (m *MeshNode) GetNodeByName(name string) (*MeshNode, bool) {
	if m.name == name {
		return m, true
	}
	for _, child := range m.children {
		node, exists := child.GetNodeByName(name)
		if exists {
			return node, true
		}
	}
	return nil, false
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

func (m *MeshNode) ResetAnimations() {
	m.ResetAnimationFrames()
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
		//println(fmt.Sprintf("[MeshNode] Hiding node %s", m.name))
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

func (m *MeshNode) setInitialTranslate(translate [3]float32) {
	m.initialTranslation = translate
	m.setTranslation(translate)
}

func (m *MeshNode) setInitialRotation(rotate [4]float32) {
	m.initialRotation = mgl32.Quat{V: mgl32.Vec3{rotate[0], rotate[1], rotate[2]}, W: rotate[3]}
	m.setRotation(rotate)

}

func (m *MeshNode) setInitialScale(scale [3]float32) {
	m.initialScale = scale
	m.setScale(scale)
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

func (m *MeshNode) SetUniformScale(f float64) {
	m.scale = [3]float32{float32(f), float32(f), float32(f)}
}

func (m *MeshNode) GetAnimationDuration() float64 {
	ownDuration := 0.0
	if m.IsCurrentAnimationValid() {
		animation := m.GetCurrentAnimation()
		ownDuration = float64(animation.Duration)
	}
	maxChildDuration := 0.0
	for _, child := range m.children {
		childDuration := child.GetAnimationDuration()
		maxChildDuration = max(maxChildDuration, childDuration)
	}
	return max(ownDuration, maxChildDuration)
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

	glErr := gl.GetError()
	if glErr != gl.NO_ERROR {
		panic(fmt.Sprintf("glhf.MakeVertexSlice failed with error %d", glErr))
	}


	slice.Begin()
	slice.SetVertexData(m.VertexData)
	slice.End()

	return slice
}
