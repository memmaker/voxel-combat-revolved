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
)

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
	shader, err = glhf.NewShader(vertexFormat, uniformFormat, guiVertexShaderSource, guiFragmentShaderSource)

	if err != nil {
		panic(err)
	}

	shader.Begin()
	shader.SetUniformAttr(0, util.Get2DPixelCoordOrthographicProjectionMatrix(a.WindowWidth, a.WindowHeight))
	shader.End()
	return shader
}

const (
	ShaderDrawTexturedQuads = int32(0)
	ShaderDrawColoredQuads  = int32(1)
	ShaderDrawCircle        = int32(3)
)
const (
	ShaderProjectionViewMatrix = 0
	ShaderModelMatrix          = 1
	ShaderDrawMode             = 2
	ShaderDrawColor            = 3
	ShaderThickness            = 4
	ShaderGlobalLightDirection = 5
	ShaderGlobalLightColor     = 6
	ShaderLightPosition        = 7
	ShaderLightColor           = 8
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

			glhf.Attr{Name: "global_light_direction", Type: glhf.Vec3}, // 5
			glhf.Attr{Name: "global_light_color", Type: glhf.Vec3},     // 6

			glhf.Attr{Name: "light_position", Type: glhf.Vec3}, // 7
			glhf.Attr{Name: "light_color", Type: glhf.Vec3},    // 8
		}
		shader *glhf.Shader
	)
	var err error
	shader, err = glhf.NewShader(vertexFormat, uniformFormat, defaultVertexShaderSource, defaultFragmentShaderSource)

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
	shader, err = glhf.NewShader(vertexFormat, uniformFormat, lineVertexShaderSource, lineFragmentShaderSource)

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
			{Name: "compressedValue", Type: glhf.Int},
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
	shader, err = glhf.NewShader(vertexFormat, uniformFormat, chunkVertexShaderSource, chunkFragmentShaderSource)

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
