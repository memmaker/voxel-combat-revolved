package glhf

import (
    "github.com/go-gl/gl/v3.3-core/gl"
)

type TransformFeedback struct {
    frontBuffer   *vertexArray[GlFloat]
    backBuffer    *vertexArray[GlFloat]
    vertexCount   int
    shader        *Shader
    primitiveType uint32
}

// idea: use transform feedback, so we have two buffers and can swap them
// We'll write from the transform vertex shader to the first buffer, then read from the second buffer in the fragment shader
// Then we'll swap the buffers and repeat

func NewParticleArray(shader *Shader, intialBufferData []GlFloat) *TransformFeedback {
    //vertexFormat := shader.VertexFormat()
    // we would probably get the shader as a parameter
    primitiveType := uint32(gl.POINTS)
    vertexSize := shader.VertexFormat().Size()
    neededBufferSizeInBytes := SizeOfFloat32 * len(intialBufferData)
    vertexCount := neededBufferSizeInBytes / vertexSize

    backBufferArray := newIndexedVertexArray[GlFloat](shader, neededBufferSizeInBytes, nil)
    backBufferArray.setPrimitiveType(primitiveType)

    // pre-fill the front buffer
    frontBufferArray := newIndexedVertexArray[GlFloat](shader, neededBufferSizeInBytes, nil)
    frontBufferArray.setPrimitiveType(primitiveType)
    frontBufferArray.begin()
    frontBufferArray.setVertexData(intialBufferData)
    frontBufferArray.end()
    v := &TransformFeedback{ // TODO: specify DYNAMIC_READ instead of STATIC_DRAW during buffer creation
        frontBuffer:   frontBufferArray,
        backBuffer:    backBufferArray,
        vertexCount:   vertexCount,
        shader:        shader,
        primitiveType: primitiveType,
    }

    return v
}
func (v *TransformFeedback) FrontToBack() {
    v.doTransfer(v.frontBuffer, v.backBuffer.vbo.obj)
}
func (v *TransformFeedback) BackToFront() {
    v.doTransfer(v.backBuffer, v.frontBuffer.vbo.obj)
}
func (v *TransformFeedback) doTransfer(src *vertexArray[GlFloat], dstBuffer uint32) {
    gl.Enable(gl.RASTERIZER_DISCARD) // no need for fragment shader, this is just between the vertex shader and these two buffers

    v.shader.Begin()
    src.begin()

    gl.BindBufferBase(gl.TRANSFORM_FEEDBACK_BUFFER, 0, dstBuffer)
    gl.BeginTransformFeedback(v.primitiveType)
    src.draw(0, v.vertexCount)
    gl.EndTransformFeedback()
    gl.BindBufferBase(gl.TRANSFORM_FEEDBACK_BUFFER, 0, 0)

    src.end()
    v.shader.End()

    gl.Disable(gl.RASTERIZER_DISCARD)
}

const SizeOfFloat32 = 4
const SizeOfMat4 = 16 * SizeOfFloat32

func (v *TransformFeedback) createMeshBuffer() uint32 {
    quadVertices := []float32{
        // positions
        // first triangle (CCW)
        0.5, 0.5, 0.0, // top right
        -0.5, 0.5, 0.0, // top left
        -0.5, -0.5, 0.0, // bottom left

        // second triangle (CCW)
        -0.5, -0.5, 0.0, // bottom left
        0.5, -0.5, 0.0, // bottom right
        0.5, 0.5, 0.0, // top right
    }

    var quadBufferID uint32

    gl.GenBuffers(1, &quadBufferID)

    gl.BindBuffer(gl.ARRAY_BUFFER, quadBufferID)
    gl.BufferData(gl.ARRAY_BUFFER, len(quadVertices)*SizeOfFloat32, gl.Ptr(quadVertices), gl.DYNAMIC_READ)

    gl.BindBuffer(gl.ARRAY_BUFFER, 0)

    return quadBufferID
}

func (v *TransformFeedback) DebugGetFrontBufferData(start, end int) []GlFloat {
    v.frontBuffer.begin()
    data := v.frontBuffer.vertexData(start, end)
    v.frontBuffer.end()
    return data
}

func (v *TransformFeedback) DebugGetBackBufferData(start, end int) []GlFloat {
    v.backBuffer.begin()
    data := v.backBuffer.vertexData(start, end)
    v.backBuffer.end()
    return data
}

// TODO: IMPORTANT -> Don't forget using glDrawTransformFeedback()

func (v *TransformFeedback) DrawBackExplicitly() {
    v.backBuffer.begin()
    v.backBuffer.draw(0, v.vertexCount)
    v.backBuffer.end()
}

func (v *TransformFeedback) DrawFrontExplicitly() {
    v.frontBuffer.begin()
    v.frontBuffer.draw(0, v.vertexCount)
    //gl.DrawArrays(gl.TRIANGLE_STRIP, int32(startIndex), int32(endIndex-startIndex))
    v.frontBuffer.end()
}

func (v *TransformFeedback) DrawBackFromFeedback() {
    v.backBuffer.begin()
    v.backBuffer.drawFromFeedback()
    v.backBuffer.end()
}

func (v *TransformFeedback) DrawFrontFromFeedback() {
    v.frontBuffer.begin()
    v.frontBuffer.drawFromFeedback()
    v.frontBuffer.end()
}
