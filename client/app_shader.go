package client

import (
	_ "embed"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/glhf"
	"github.com/memmaker/battleground/engine/util"
)

var (
	//go:embed shader/model.vert
	modelVertexShaderSource string

	//go:embed shader/model.frag
	modelFragmentShaderSource string

	//go:embed shader/chunk.vert
	chunkVertexShaderSource string

	//go:embed shader/chunk.frag
	chunkFragmentShaderSource string

	//go:embed shader/highlight.vert
	highlightVertexShaderSource string

	//go:embed shader/highlight.frag
	highlightFragmentShaderSource string

	//go:embed shader/line.vert
	lineVertexShaderSource string

	//go:embed shader/line.frag
	lineFragmentShaderSource string

	//go:embed shader/gui.vert
	guiVertexShaderSource string

	//go:embed shader/gui.frag
	guiFragmentShaderSource string
)

func (a *BattleGame) loadGuiShader() *glhf.Shader {
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
	shader.SetUniformAttr(0, util.Get2DOrthographicProjectionMatrix(a.WindowWidth, a.WindowHeight))
	shader.End()
	return shader
}

func (a *BattleGame) loadLineShader() *glhf.Shader {
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

func (a *BattleGame) loadChunkShader() *glhf.Shader {
	var (
		vertexFormat = glhf.AttrFormat{
			{Name: "compressedValue", Type: glhf.Int},
		}
		uniformFormat = glhf.AttrFormat{
			glhf.Attr{Name: "projection", Type: glhf.Mat4},
			glhf.Attr{Name: "camera", Type: glhf.Mat4},
			glhf.Attr{Name: "model", Type: glhf.Mat4},
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
	shader.SetUniformAttr(0, a.isoCamera.GetProjectionMatrix())

	shader.SetUniformAttr(1, a.isoCamera.GetViewMatrix())

	model := mgl32.Ident4()
	shader.SetUniformAttr(2, model)

	lightPos := mgl32.Vec3{1, 5, 0}
	shader.SetUniformAttr(3, lightPos)

	lightColor := mgl32.Vec3{1, 1, 1}
	shader.SetUniformAttr(4, lightColor)

	shader.End()
	return shader
}

func (a *BattleGame) loadHighlightShader() *glhf.Shader {
	var (
		vertexFormat = glhf.AttrFormat{
			{Name: "compressedValue", Type: glhf.Int},
		}
		uniformFormat = glhf.AttrFormat{
			glhf.Attr{Name: "projection", Type: glhf.Mat4},
			glhf.Attr{Name: "camera", Type: glhf.Mat4},
			glhf.Attr{Name: "model", Type: glhf.Mat4},
			glhf.Attr{Name: "light_position", Type: glhf.Vec3},
			glhf.Attr{Name: "light_color", Type: glhf.Vec3},
		}
		shader *glhf.Shader
	)

	var err error
	shader, err = glhf.NewShader(vertexFormat, uniformFormat, highlightVertexShaderSource, highlightFragmentShaderSource)

	if err != nil {
		panic(err)
	}

	shader.Begin()
	shader.SetUniformAttr(0, a.isoCamera.GetProjectionMatrix())

	shader.SetUniformAttr(1, a.isoCamera.GetViewMatrix())

	model := mgl32.Ident4()
	shader.SetUniformAttr(2, model)

	lightPos := mgl32.Vec3{1, 5, 0}
	shader.SetUniformAttr(3, lightPos)

	lightColor := mgl32.Vec3{1, 1, 1}
	shader.SetUniformAttr(4, lightColor)

	shader.End()
	return shader
}
func (a *BattleGame) loadModelShader() *glhf.Shader {
	var (
		vertexFormat = glhf.AttrFormat{
			{Name: "position", Type: glhf.Vec3},
			{Name: "normal", Type: glhf.Vec3},
			{Name: "texCoord", Type: glhf.Vec2},
		}
		uniformFormat = glhf.AttrFormat{
			glhf.Attr{Name: "projection", Type: glhf.Mat4},
			glhf.Attr{Name: "camera", Type: glhf.Mat4},
			glhf.Attr{Name: "model", Type: glhf.Mat4},
			glhf.Attr{Name: "light_position", Type: glhf.Vec3},
			glhf.Attr{Name: "light_color", Type: glhf.Vec3},
		}
		shader *glhf.Shader
	)

	var err error
	shader, err = glhf.NewShader(vertexFormat, uniformFormat, modelVertexShaderSource, modelFragmentShaderSource)

	if err != nil {
		panic(err)
	}

	shader.Begin()
	shader.SetUniformAttr(0, a.isoCamera.GetProjectionMatrix())

	shader.SetUniformAttr(1, a.isoCamera.GetViewMatrix())

	model := mgl32.Ident4()
	shader.SetUniformAttr(2, model)

	lightPos := mgl32.Vec3{1, 5, 0}
	shader.SetUniformAttr(3, lightPos)

	lightColor := mgl32.Vec3{1, 1, 1}
	shader.SetUniformAttr(4, lightColor)

	shader.End()
	return shader
}
