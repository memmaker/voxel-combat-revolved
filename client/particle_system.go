package client

import (
    "github.com/go-gl/gl/v3.3-core/gl"
    "github.com/go-gl/mathgl/mgl32"
    "github.com/memmaker/battleground/engine/glhf"
    "github.com/memmaker/battleground/engine/util"
)

type TestParticleSystem struct {
    particleShader    *glhf.Shader
    transformFeedback *glhf.TransformFeedback
    frontIsSource     bool
}

func NewTestParticleSystem(transformFeedbackShader *glhf.Shader, particleShader *glhf.Shader) *TestParticleSystem {
    instanceData := []glhf.GlFloat{
        1, 2, 3,
        4, 5, 6,
    }
    /*
       primitives := 2           // 2 points
       verticesPerPrimitive := 1 // 1 vertex per point
       totalVertices := primitives * verticesPerPrimitive
       sizeOfOneVertex := transformFeedbackShader.VertexFormat().Size()
    */

    //sizeOfOnePrimitive := sizeOfOneVertex * verticesPerPrimitive
    //sizeOfAllVertices := sizeOfOneVertex * totalVertices

    transformFeedback := glhf.NewParticleArray(transformFeedbackShader, instanceData)

    // we apply the vertex shader and copy the results to the back buffer

    //transformFeedback.BackToFront()
    /*
       backBuffer := transformFeedback.DebugGetBackBufferData(0, 2)
       println(fmt.Sprintf("backBuffer: %v", backBuffer))
       frontBuffer := transformFeedback.DebugGetFrontBufferData(0, 2)
       println(fmt.Sprintf("frontBuffer: %v", frontBuffer))
    */
    return &TestParticleSystem{
        frontIsSource:     true,
        particleShader:    particleShader,
        transformFeedback: transformFeedback,
    }
}

func (p *TestParticleSystem) Draw(camera util.Camera) {

    glError := gl.GetError()
    if glError != gl.NO_ERROR {
        println("Gl Error:", glError)
    }

    modelMatrix := mgl32.Ident4()
    modelView := camera.GetViewMatrix().Mul4(modelMatrix)
    if p.frontIsSource {
        p.transformFeedback.FrontToBack()
        // use the particle shader to render the back buffer
        p.particleShader.Begin()
        p.particleShader.SetUniformAttr(0, camera.GetProjectionMatrix())
        p.particleShader.SetUniformAttr(1, modelView)
        p.particleShader.SetUniformAttr(2, camera.GetPosition())
        p.particleShader.SetUniformAttr(3, camera.GetUp())
        p.transformFeedback.DrawBackExplicitly()
        p.particleShader.End()
        p.frontIsSource = false
    } else {
        p.transformFeedback.BackToFront()
        // use the particle shader to render the front buffer
        p.particleShader.Begin()
        p.particleShader.SetUniformAttr(0, camera.GetProjectionMatrix())
        p.particleShader.SetUniformAttr(1, modelView)
        p.particleShader.SetUniformAttr(2, camera.GetPosition())
        p.particleShader.SetUniformAttr(3, camera.GetUp())

        p.transformFeedback.DrawFrontExplicitly()
        p.particleShader.End()
        p.frontIsSource = true
    }
    glError2 := gl.GetError()
    if glError2 != gl.NO_ERROR {
        println("Gl Error2:", glError)
    }
}
