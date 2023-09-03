package glhf

import (
    "github.com/go-gl/gl/v4.1-core/gl"
    "github.com/go-gl/mathgl/mgl32"
    "math/rand"
)

type ParticleProperties struct {
    Position, PositionVariation mgl32.Vec3
    Velocity, VelocityVariation       mgl32.Vec3
    //RotationVariation                 float32
    ColorBegin, ColorEnd              mgl32.Vec4
    SizeBegin, SizeEnd, SizeVariation float32
    Lifetime                          float32
}

func (p ParticleProperties) WithPosition(newPos mgl32.Vec3) ParticleProperties {
    p.Position = newPos
    return p
}

type Particle struct {
    id                     uint64
    Position, Velocity     mgl32.Vec3
    ColorBegin, ColorEnd   mgl32.Vec4
    //Rotation               float32
    SizeBegin, SizeEnd     float32
    Lifetime, LifetimeLeft float32
    IsActive               bool
}

func (p Particle) GetID() uint64 {
    return p.id
}

type ParticleSystem struct {
    frontBuffer *vertexArray[GlFloat]
    backBuffer  *vertexArray[GlFloat]

    frontIsSource bool

    currentOffset           int
    maxVertexCount          int
    transformFeedbackShader *Shader
    particleShader          *Shader
    primitiveType           uint32
    xfbo                    uint32

    getProjection func() mgl32.Mat4
    getView       func() mgl32.Mat4
}

// idea: use transform feedback, so we have two buffers and can swap them
// We'll write from the transform vertex transformFeedbackShader to the first buffer, then read from the second buffer in the fragment transformFeedbackShader
// Then we'll swap the buffers and repeat

func NewParticleSystem(particleCount int, tfShader, particleShader *Shader, getView, getProj func() mgl32.Mat4) *ParticleSystem {
    v := &ParticleSystem{ // TODO: specify DYNAMIC_READ instead of STATIC_DRAW during buffer creation
        maxVertexCount:          particleCount,
        transformFeedbackShader: tfShader,
        particleShader:          particleShader,
        getProjection:           getProj,
        getView:                 getView,
        frontIsSource:           true,
    }
    v.initializeBuffers(particleCount)
    return v
}

func (v *ParticleSystem) initializeBuffers(particleCount int) {
    var xfb uint32
    gl.GenTransformFeedbacks(1, &xfb)
    gl.BindTransformFeedback(gl.TRANSFORM_FEEDBACK, xfb)

    glError := gl.GetError()
    if glError != gl.NO_ERROR {
        println("BindTransformFeedback:", glError)
    }

    v.xfbo = xfb

    primitiveType := uint32(gl.POINTS)
    //vertexSize := v.transformFeedbackShader.VertexFormat().Size()
    //neededBufferSizeInBytes := vertexSize * particleCount
    //floatCount := neededBufferSizeInBytes / SizeOfFloat32

    v.backBuffer = newIndexedVertexArray[GlFloat](v.transformFeedbackShader, particleCount, nil)
    v.backBuffer.setPrimitiveType(primitiveType)

    // pre-fill the front buffer
    v.frontBuffer = newIndexedVertexArray[GlFloat](v.transformFeedbackShader, particleCount, nil)
    v.frontBuffer.setPrimitiveType(primitiveType)

    glError = gl.GetError()
    if glError != gl.NO_ERROR {
        println("failed to initializeBuffers:", glError)
    }
}
func (v *ParticleSystem) Draw(deltaTime float64) {
    v.doTransfer(v.currentBackBuffer(), v.currentFrontBuffer(), deltaTime)
    v.draw(v.currentFrontBuffer())
    v.frontIsSource = !v.frontIsSource
}

func (v *ParticleSystem) currentBackBuffer() *vertexArray[GlFloat] {
    if v.frontIsSource {
        return v.backBuffer
    }
    return v.frontBuffer
}

func (v *ParticleSystem) currentFrontBuffer() *vertexArray[GlFloat] {
    if v.frontIsSource {
        return v.frontBuffer
    }
    return v.backBuffer
}

func (v *ParticleSystem) doTransfer(src, dest *vertexArray[GlFloat], deltaTime float64) {
    gl.BindTransformFeedback(gl.TRANSFORM_FEEDBACK, v.xfbo)

    gl.Enable(gl.RASTERIZER_DISCARD) // no need for fragment transformFeedbackShader, this is just between the vertex transformFeedbackShader and these two buffers

    v.transformFeedbackShader.Begin()
    v.transformFeedbackShader.SetUniformAttr(0, float32(deltaTime))
    src.begin()

    gl.BindBufferBase(gl.TRANSFORM_FEEDBACK_BUFFER, 0, dest.vbo.obj)

    gl.BeginTransformFeedback(v.primitiveType)
    src.draw(0, v.maxVertexCount)
    gl.EndTransformFeedback()

    gl.BindBufferBase(gl.TRANSFORM_FEEDBACK_BUFFER, 0, 0)

    src.end()
    v.transformFeedbackShader.End()

    gl.Disable(gl.RASTERIZER_DISCARD)

    //gl.Flush()
}

func (v *ParticleSystem) draw(drawBuffer *vertexArray[GlFloat]) {
    view := v.getView()
    proj := v.getProjection()

    v.particleShader.Begin()
    v.particleShader.SetUniformAttr(0, proj)
    //v.particleShader.SetUniformAttr(2, float32(20)) // lifetime
    modelMatrix := mgl32.Ident4()
    modelView := view.Mul4(modelMatrix)

    v.particleShader.SetUniformAttr(1, modelView)

    drawBuffer.begin()

    gl.DrawTransformFeedback(gl.POINTS, v.xfbo)
    drawBuffer.end()

    v.particleShader.End()
}
func (v *ParticleSystem) Emit(props ParticleProperties, count int) {
    offset := v.currentOffset
    if count >= v.maxVertexCount {
        count = v.maxVertexCount
        offset = 0
    }
    buffer := v.currentBackBuffer()
    flatData := make([]GlFloat, 0)
    for index := 0; index < count; index++ {
        flatData = append(flatData, v.createParticle(props, index)...)
    }
    buffer.begin()
    if offset+count > v.maxVertexCount {
        buffer.setVertexDataWithOffset(offset, flatData[:v.maxVertexCount-offset])
        buffer.setVertexDataWithOffset(0, flatData[v.maxVertexCount-offset:])
    } else {
        buffer.setVertexDataWithOffset(offset, flatData)
    }
    buffer.end()

    v.particleShader.Begin()
    v.particleShader.SetUniformAttr(2, props.Lifetime)
    v.particleShader.SetUniformAttr(3, props.ColorBegin)
    v.particleShader.SetUniformAttr(4, props.ColorEnd)
    v.particleShader.SetUniformAttr(5, props.SizeEnd)
    v.particleShader.End()

    v.currentOffset = (offset + count) % v.maxVertexCount
}

func (v *ParticleSystem) createParticle(props ParticleProperties, index int) []GlFloat {
    return []GlFloat{
        // position x,y,z
        GlFloat(props.Position.X() + props.PositionVariation.X()*(rand.Float32()-0.5)),
        GlFloat(props.Position.Y() + props.PositionVariation.Y()*(rand.Float32()-0.5)),
        GlFloat(props.Position.Z() + props.PositionVariation.Z()*(rand.Float32()-0.5)),
        // lifetime left
        GlFloat(props.Lifetime),
        // velocity X
        GlFloat(props.Velocity.X() + props.VelocityVariation.X()*(rand.Float32()-0.5)),
        // velocity Y
        GlFloat(props.Velocity.Y() + props.VelocityVariation.Y()*(rand.Float32()-0.5)),
        // velocity Z
        GlFloat(props.Velocity.Z() + props.VelocityVariation.Z()*(rand.Float32()-0.5)),
        // size begin
        GlFloat(props.SizeBegin + props.SizeVariation*(rand.Float32()-0.5)),
    }
}

const SizeOfFloat32 = 4
