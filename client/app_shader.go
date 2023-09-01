package client

import (
	_ "embed"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/glhf"
	"github.com/memmaker/battleground/engine/util"
)

var (
	//go:embed shader/chunk.vert
	chunkVertexShaderSource string

	//go:embed shader/chunk.frag
	chunkFragmentShaderSource string

	//go:embed shader/line.vert
	lineVertexShaderSource string

	//go:embed shader/line.frag
	lineFragmentShaderSource string

	//go:embed shader/gui.vert
	guiVertexShaderSource string

	//go:embed shader/gui.frag
	guiFragmentShaderSource string

	//go:embed shader/default.vert
	defaultVertexShaderSource string

	//go:embed shader/default.frag
	defaultFragmentShaderSource string

	//go:embed shader/quad_particle.geom
	quadParticleGeometryShaderSource string

	//go:embed shader/quad_particle.vert
	quadParticleVertexShaderSource string

	//go:embed shader/quad_particle.frag
	quadParticleFragmentShaderSource string

	//go:embed shader/transform_feedback.vert
	transformFeedbackVertexShaderSource string
)

func loadTransformFeedbackShader() *glhf.Shader {
	vertexFormat := glhf.AttrFormat{
		{Name: "inputPosition", Type: glhf.Vec3},
	}
	uniformFormat := glhf.AttrFormat{}

	tfShader, shaderErr := glhf.NewShader(
		vertexFormat,
		uniformFormat,
		transformFeedbackVertexShaderSource,
		"",
		"",
		[]string{"outputPosition"},
	)
	if shaderErr != nil {
		panic(shaderErr)
	}

	return tfShader
}

func loadParticleShader() *glhf.Shader {
	vertexFormat := glhf.AttrFormat{
		{Name: "inputPosition", Type: glhf.Vec3},
	}
	uniformFormat := glhf.AttrFormat{
		glhf.Attr{Name: "projection", Type: glhf.Mat4},
		glhf.Attr{Name: "modelView", Type: glhf.Mat4},
		glhf.Attr{Name: "camPos", Type: glhf.Vec3},
		glhf.Attr{Name: "camUp", Type: glhf.Vec3},
	}

	particleShader, shaderErr := glhf.NewShader(
		vertexFormat,
		uniformFormat,
		quadParticleVertexShaderSource,
		quadParticleGeometryShaderSource,
		quadParticleFragmentShaderSource,
		nil,
	)
	if shaderErr != nil {
		panic(shaderErr)
	}

	return particleShader
}

func (a *BattleClient) loadGuiShader() *glhf.Shader {
	var (
		vertexFormat = glhf.AttrFormat{
			{Name: "position", Type: glhf.Vec2},
			{Name: "texCoord", Type: glhf.Vec2},
		}
		uniformFormat = glhf.AttrFormat{
			glhf.Attr{Name: "projection", Type: glhf.Mat4},
			glhf.Attr{Name: "model", Type: glhf.Mat4},
		}
		shader *glhf.Shader
	)
	var err error
	shader, err = glhf.NewBasicShader(vertexFormat, uniformFormat, guiVertexShaderSource, guiFragmentShaderSource)

	if err != nil {
		panic(err)
	}

	shader.Begin()
	shader.SetUniformAttr(0, util.Get2DPixelCoordOrthographicProjectionMatrix(a.WindowWidth, a.WindowHeight))
	shader.End()
	return shader
}

const (
	ShaderDrawTexturedQuads      = int32(0)
	ShaderDrawColoredQuads       = int32(1)
	ShaderDrawColoredFadingQuads = int32(2)
	ShaderDrawCircle             = int32(3)
	ShaderDrawLine               = int32(4)
)
const (
	ShaderProjectionViewMatrix = 0
	ShaderModelMatrix          = 1
	ShaderDrawMode             = 2
	ShaderDrawColor            = 3
	ShaderThickness            = 4
	ShaderViewport             = 5
	ShaderGlobalLightDirection = 6
	ShaderGlobalLightColor     = 7
	ShaderLightPosition        = 8
	ShaderLightColor           = 9
	ShaderMultiPurpose         = 10
)

func (a *BattleClient) loadDefaultShader() *glhf.Shader {
	var (
		vertexFormat = glhf.AttrFormat{
			{Name: "position", Type: glhf.Vec3},
			{Name: "texCoord", Type: glhf.Vec2},
			{Name: "vertexColor", Type: glhf.Vec3},
			{Name: "normal", Type: glhf.Vec3},
		}
		uniformFormat = glhf.AttrFormat{
			glhf.Attr{Name: "camProjectionView", Type: glhf.Mat4}, // 0
			glhf.Attr{Name: "modelTransform", Type: glhf.Mat4},    // 1

			glhf.Attr{Name: "drawMode", Type: glhf.Int}, // 2
			glhf.Attr{Name: "color", Type: glhf.Vec4},   // 3

			glhf.Attr{Name: "thickness", Type: glhf.Float}, // 4

			glhf.Attr{Name: "viewport", Type: glhf.Vec2}, // 5

			glhf.Attr{Name: "global_light_direction", Type: glhf.Vec3}, // 6
			glhf.Attr{Name: "global_light_color", Type: glhf.Vec3},     // 7

			glhf.Attr{Name: "light_position", Type: glhf.Vec3}, // 8
			glhf.Attr{Name: "light_color", Type: glhf.Vec3},    // 9
			glhf.Attr{Name: "multi", Type: glhf.Float},         // 10
		}
		shader *glhf.Shader
	)
	var err error
	shader, err = glhf.NewBasicShader(vertexFormat, uniformFormat, defaultVertexShaderSource, defaultFragmentShaderSource)

	if err != nil {
		panic(err)
	}

	shader.Begin()
	shader.SetUniformAttr(ShaderProjectionViewMatrix, a.isoCamera.GetProjectionViewMatrix())

	lightPos := mgl32.Vec3{1, 5, 0}
	shader.SetUniformAttr(ShaderLightPosition, lightPos)

	lightColor := mgl32.Vec3{0.4, 0.4, 0.4}
	shader.SetUniformAttr(ShaderLightColor, lightColor)

	shader.End()
	return shader
}

func (a *BattleClient) loadLineShader() *glhf.Shader {
	var (
		vertexFormat = glhf.AttrFormat{
			{Name: "position", Type: glhf.Vec3},
		}
		uniformFormat = glhf.AttrFormat{
			glhf.Attr{Name: "projection", Type: glhf.Mat4},
			glhf.Attr{Name: "camera", Type: glhf.Mat4},
			glhf.Attr{Name: "model", Type: glhf.Mat4},
			glhf.Attr{Name: "drawColor", Type: glhf.Vec3},
		}
		shader *glhf.Shader
	)
	var err error
	shader, err = glhf.NewBasicShader(vertexFormat, uniformFormat, lineVertexShaderSource, lineFragmentShaderSource)

	if err != nil {
		panic(err)
	}

	shader.Begin()
	shader.SetUniformAttr(0, a.isoCamera.GetProjectionMatrix())
	shader.SetUniformAttr(1, a.isoCamera.GetViewMatrix())
	model := mgl32.Ident4()
	shader.SetUniformAttr(2, model)
	shader.End()
	return shader
}

func (a *BattleClient) loadChunkShader() *glhf.Shader {
	var (
		vertexFormat = glhf.AttrFormat{
			{Name: "compressedValue", Type: glhf.UInt},
		}
		uniformFormat = glhf.AttrFormat{
			glhf.Attr{Name: "camProjectionView", Type: glhf.Mat4},
			glhf.Attr{Name: "modelTransform", Type: glhf.Mat4},
			glhf.Attr{Name: "light_position", Type: glhf.Vec3},
			glhf.Attr{Name: "light_color", Type: glhf.Vec3},
		}
		shader *glhf.Shader
	)

	var err error
	shader, err = glhf.NewBasicShader(vertexFormat, uniformFormat, chunkVertexShaderSource, chunkFragmentShaderSource)

	if err != nil {
		panic(err)
	}

	shader.Begin()
	shader.SetUniformAttr(ShaderProjectionViewMatrix, a.isoCamera.GetProjectionViewMatrix())

	model := mgl32.Ident4()
	shader.SetUniformAttr(ShaderModelMatrix, model)

	lightPos := mgl32.Vec3{1, 5, 0}
	shader.SetUniformAttr(2, lightPos)

	lightColor := mgl32.Vec3{0.4, 0.4, 0.4}
	shader.SetUniformAttr(3, lightColor)

	shader.End()
	return shader
}
