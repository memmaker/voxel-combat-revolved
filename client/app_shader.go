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

// v1 particles need for every instance:
// position, 3 floats
// lifetimeLeft, 1 float
// velocity, 3 floats
// size begin, 1 float

/*
{Name: "position", Type: glhf.Vec3},
{Name: "lifetimeLeft", Type: glhf.Float},
{Name: "velocity", Type: glhf.Vec3},
{Name: "sizeBegin", Type: glhf.Float},
*/

// and as uniform:
// color begin, end
// size end
// lifetime

/* particle shader
glhf.Attr{Name: "lifetime", Type: glhf.Float},
glhf.Attr{Name: "colorBeginEnd", Type: glhf.Vec2},
glhf.Attr{Name: "sizeEnd", Type: glhf.Float},
*/

type TransformFeedbackUniforms int32
type ParticleUniforms int32

const (
	TransformFeedbackUniformDeltaTime = TransformFeedbackUniforms(0)
)
func loadTransformFeedbackShader() *glhf.Shader {
	vertexFormat := glhf.AttrFormat{
		{Name: "position", Type: glhf.Vec3},
		{Name: "lifetimeLeft", Type: glhf.Float},
		{Name: "velocity", Type: glhf.Vec3},
		{Name: "sizeBegin", Type: glhf.Float},
	}
	uniformFormat := glhf.AttrFormat{
		glhf.Attr{Name: "deltaTime", Type: glhf.Float},
	}

	tfShader, shaderErr := glhf.NewShader(
		vertexFormat,
		uniformFormat,
		transformFeedbackVertexShaderSource,
		"",
		"",
		[]string{"VS_OUT.position\x00", "VS_OUT.lifetimeLeft\x00", "VS_OUT.velocity\x00", "VS_OUT.sizeBegin\x00"},
	)
	if shaderErr != nil {
		panic(shaderErr)
	}

	return tfShader
}

func loadParticleShader() *glhf.Shader {
	vertexFormat := glhf.AttrFormat{
		{Name: "position", Type: glhf.Vec3},
		{Name: "lifetimeLeft", Type: glhf.Float},
		{Name: "velocity", Type: glhf.Vec3},
		{Name: "sizeBegin", Type: glhf.Float},
	}
	uniformFormat := glhf.AttrFormat{
		glhf.Attr{Name: "projection", Type: glhf.Mat4},
		glhf.Attr{Name: "modelView", Type: glhf.Mat4},

		glhf.Attr{Name: "lifetime", Type: glhf.Float},
		glhf.Attr{Name: "colorBegin", Type: glhf.Vec4},
		glhf.Attr{Name: "colorEnd", Type: glhf.Vec4},
		glhf.Attr{Name: "sizeEnd", Type: glhf.Float},
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
			glhf.Attr{Name: "projection", Type: glhf.Mat4},       // 0
			glhf.Attr{Name: "model", Type: glhf.Mat4},            // 1
			glhf.Attr{Name: "appliedTintColor", Type: glhf.Vec4}, // 2
			glhf.Attr{Name: "discardedColor", Type: glhf.Vec4},   // 3
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
	ShaderDrawBillboards         = int32(5)
)
const (
	ShaderViewMatrix           = 0
	ShaderProjectionMatrix     = 1
	ShaderModelMatrix          = 2
	ShaderDrawMode             = 3
	ShaderDrawColor            = 4
	ShaderThickness            = 5
	ShaderViewport             = 6
	ShaderGlobalLightDirection = 7
	ShaderGlobalLightColor     = 8
	ShaderLightPosition        = 9
	ShaderLightColor           = 10
	ShaderMultiPurpose         = 11
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
			glhf.Attr{Name: "camView", Type: glhf.Mat4},        // 0
			glhf.Attr{Name: "camProjection", Type: glhf.Mat4},  // 1
			glhf.Attr{Name: "modelTransform", Type: glhf.Mat4}, // 2

			glhf.Attr{Name: "drawMode", Type: glhf.Int}, // 3
			glhf.Attr{Name: "color", Type: glhf.Vec4},   // 4

			glhf.Attr{Name: "thickness", Type: glhf.Float}, // 5

			glhf.Attr{Name: "viewport", Type: glhf.Vec2}, // 6

			glhf.Attr{Name: "global_light_direction", Type: glhf.Vec3}, // 7
			glhf.Attr{Name: "global_light_color", Type: glhf.Vec3},     // 8

			glhf.Attr{Name: "light_position", Type: glhf.Vec3}, // 9
			glhf.Attr{Name: "light_color", Type: glhf.Vec3},    // 10
			glhf.Attr{Name: "multi", Type: glhf.Float},         // 11
		}
		shader *glhf.Shader
	)
	var err error
	shader, err = glhf.NewBasicShader(vertexFormat, uniformFormat, defaultVertexShaderSource, defaultFragmentShaderSource)

	if err != nil {
		panic(err)
	}

	shader.Begin()
	shader.SetUniformAttr(ShaderViewMatrix, a.isoCamera.GetViewMatrix())
	shader.SetUniformAttr(ShaderProjectionMatrix, a.isoCamera.GetProjectionMatrix())

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
	shader.SetUniformAttr(0, a.isoCamera.GetProjectionViewMatrix())

	model := mgl32.Ident4()
	shader.SetUniformAttr(1, model)

	lightPos := mgl32.Vec3{1, 5, 0}
	shader.SetUniformAttr(2, lightPos)

	lightColor := mgl32.Vec3{0.4, 0.4, 0.4}
	shader.SetUniformAttr(3, lightColor)

	shader.End()
	return shader
}
