package glhf

import (
	"fmt"
	"runtime"

	"github.com/faiface/mainthread"
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

// Shader is an OpenGL transformFeedbackShader program.
type Shader struct {
	program    binder
	vertexFmt  AttrFormat
	uniformFmt AttrFormat
	uniformLoc []int32
}

func NewBasicShader(vertexFmt, uniformFmt AttrFormat, vertexShader, fragmentShader string) (*Shader, error) {
	return NewShader(vertexFmt, uniformFmt, vertexShader, "", fragmentShader, nil)
}

// NewShader creates a new transformFeedbackShader program from the specified vertex, geometry and fragment transformFeedbackShader
// sources.
// Note that vertexShader and fragmentShader parameters must contain the source code, they're
// not filenames.
func NewShader(vertexFmt, uniformFmt AttrFormat, vertexShader, geometryShader, fragmentShader string, outputVariables []string) (*Shader, error) {
	shader := &Shader{
		program: binder{
			restoreLoc: gl.CURRENT_PROGRAM,
			bindFunc: func(obj uint32) {
				gl.UseProgram(obj)
			},
		},
		vertexFmt:  vertexFmt,
		uniformFmt: uniformFmt,
		uniformLoc: make([]int32, len(uniformFmt)),
	}

	var errorMessages []string
	var vshader, gshader, fshader uint32

	// vertex shader
	{
		vshader = gl.CreateShader(gl.VERTEX_SHADER)
		src, free := gl.Strs(vertexShader)
		defer free()
		length := int32(len(vertexShader))
		gl.ShaderSource(vshader, 1, src, &length)
		AppendGLErrorMessage(errorMessages, "failed to set vertex transformFeedbackShader source: %d")

		gl.CompileShader(vshader)
		AppendGLErrorMessage(errorMessages, "failed to compile vertex transformFeedbackShader: %d")

		var success int32
		gl.GetShaderiv(vshader, gl.COMPILE_STATUS, &success)
		if success == gl.FALSE {
			var logLen int32
			gl.GetShaderiv(vshader, gl.INFO_LOG_LENGTH, &logLen)

			infoLog := make([]byte, logLen)
			gl.GetShaderInfoLog(vshader, logLen, nil, &infoLog[0])
			if len(infoLog) > 0 {
				errorMessages = append(errorMessages, fmt.Sprintf("error compiling vertex shader: %s", string(infoLog)))
			}
			return nil, fmt.Errorf("error compiling vertex shader: %s", string(infoLog))
		}

		defer gl.DeleteShader(vshader)
	}

	// geometry shader
	if geometryShader != "" {
		gshader = gl.CreateShader(gl.GEOMETRY_SHADER)
		src, free := gl.Strs(geometryShader)
		defer free()
		length := int32(len(geometryShader))
		gl.ShaderSource(gshader, 1, src, &length)
		AppendGLErrorMessage(errorMessages, "failed to set geometry shader source: %d")

		gl.CompileShader(gshader)
		AppendGLErrorMessage(errorMessages, "failed to compile geometry shader: %d")

		var success int32
		gl.GetShaderiv(gshader, gl.COMPILE_STATUS, &success)
		if success == gl.FALSE {
			var logLen int32
			gl.GetShaderiv(gshader, gl.INFO_LOG_LENGTH, &logLen)

			infoLog := make([]byte, logLen)
			gl.GetShaderInfoLog(gshader, logLen, nil, &infoLog[0])
			if len(infoLog) > 0 {
				errorMessages = append(errorMessages, fmt.Sprintf("error compiling geometry shader: %s", string(infoLog)))
			}
			return nil, fmt.Errorf("error compiling geometry shader: %s", string(infoLog))
		}

		defer gl.DeleteShader(gshader)
	}

	// fragment shader
	if fragmentShader != "" {
		fshader = gl.CreateShader(gl.FRAGMENT_SHADER)
		src, free := gl.Strs(fragmentShader)
		defer free()
		length := int32(len(fragmentShader))
		gl.ShaderSource(fshader, 1, src, &length)
		AppendGLErrorMessage(errorMessages, "failed to set fragment transformFeedbackShader source: %d")

		gl.CompileShader(fshader)
		AppendGLErrorMessage(errorMessages, "failed to compile fragment transformFeedbackShader: %d")

		var success int32
		gl.GetShaderiv(fshader, gl.COMPILE_STATUS, &success)
		if success == gl.FALSE {
			var logLen int32
			gl.GetShaderiv(fshader, gl.INFO_LOG_LENGTH, &logLen)

			infoLog := make([]byte, logLen)
			gl.GetShaderInfoLog(fshader, logLen, nil, &infoLog[0])
			if len(infoLog) > 0 {
				errorMessages = append(errorMessages, fmt.Sprintf("error compiling fragment transformFeedbackShader: %s", string(infoLog)))
			}
			return nil, fmt.Errorf("error compiling fragment transformFeedbackShader: %s", string(infoLog))
		}

		defer gl.DeleteShader(fshader)
	}

	// shader program
	{
		shader.program.obj = gl.CreateProgram()
		gl.AttachShader(shader.program.obj, vshader)
		AppendGLErrorMessage(errorMessages, "failed to attach vertex shader: %d")

		if geometryShader != "" {
			gl.AttachShader(shader.program.obj, gshader)
			AppendGLErrorMessage(errorMessages, "failed to attach geometry shader: %d")
		}

		if fragmentShader != "" {
			gl.AttachShader(shader.program.obj, fshader)
			AppendGLErrorMessage(errorMessages, "failed to attach fragment shader: %d")
		}

		if outputVariables != nil && len(outputVariables) > 0 {
			// this must be right, since it works perfectly fine for the vertex shader only!
			// when using 4 variables or less..
			cstrs, free := gl.Strs(outputVariables...)
			gl.TransformFeedbackVaryings(shader.program.obj, int32(len(outputVariables)), cstrs, gl.INTERLEAVED_ATTRIBS)
			AppendGLErrorMessage(errorMessages, "failed to set transform feedback varyings: %d")
			free()
		}

		gl.LinkProgram(shader.program.obj)
		AppendGLErrorMessage(errorMessages, "failed to link shader program: %d")

		var success int32
		gl.GetProgramiv(shader.program.obj, gl.LINK_STATUS, &success)
		if success == gl.FALSE {
			var logLen int32
			gl.GetProgramiv(shader.program.obj, gl.INFO_LOG_LENGTH, &logLen)

			infoLog := make([]byte, logLen)
			gl.GetProgramInfoLog(shader.program.obj, logLen, nil, &infoLog[0])
			if len(infoLog) > 0 {
				errorMessages = append(errorMessages, fmt.Sprintf("error linking shader program: %s", string(infoLog)))
			}
			return nil, fmt.Errorf("error linking shader program: %s", string(infoLog))
		} else {
			var logLen int32
			gl.GetProgramiv(shader.program.obj, gl.INFO_LOG_LENGTH, &logLen)
			if logLen > 0 {
				infoLog := make([]byte, logLen)
				gl.GetProgramInfoLog(shader.program.obj, logLen, nil, &infoLog[0])
				if len(infoLog) > 0 {
					errorMessages = append(errorMessages, fmt.Sprintf("error linking shader program: %s", string(infoLog)))
				}
			}
		}
	}

	// uniforms
	for i, uniform := range uniformFmt {
		loc := gl.GetUniformLocation(shader.program.obj, gl.Str(uniform.Name+"\x00"))
		shader.uniformLoc[i] = loc
	}

	runtime.SetFinalizer(shader, (*Shader).delete)

	if len(errorMessages) > 0 {
		return nil, fmt.Errorf("errors during shader creation:\n%s", errorMessages)
	}

	return shader, nil
}

func AppendGLErrorMessage(errorMessages []string, errorMessage string) []string {
	glError := gl.GetError()
	if glError != gl.NO_ERROR {
		errorMessages = append(errorMessages, fmt.Sprintf(errorMessage, glError))
	}
	return errorMessages
}

func (s *Shader) delete() {
	mainthread.CallNonBlock(func() {
		gl.DeleteProgram(s.program.obj)
	})
}

// ID returns the OpenGL ID of this Shader.
func (s *Shader) ID() uint32 {
	return s.program.obj
}

// VertexFormat returns the vertex attribute format of this Shader. Do not change it.
func (s *Shader) VertexFormat() AttrFormat {
	return s.vertexFmt
}

// UniformFormat returns the uniform attribute format of this Shader. Do not change it.
func (s *Shader) UniformFormat() AttrFormat {
	return s.uniformFmt
}

// SetUniformAttr sets the value of a uniform attribute of this Shader. The attribute is
// specified by the index in the Shader's uniform format.
//
// If the uniform attribute does not exist in the Shader, this method returns false.
//
// Supplied value must correspond to the type of the attribute. Correct types are these
// (right-hand is the type of the value):
//
//	Attr{Type: Int}:   int32
//	Attr{Type: Float}: float32
//	Attr{Type: Vec2}:  mgl32.Vec2
//	Attr{Type: Vec3}:  mgl32.Vec3
//	Attr{Type: Vec4}:  mgl32.Vec4
//	Attr{Type: Mat2}:  mgl32.Mat2
//	Attr{Type: Mat23}: mgl32.Mat2x3
//	Attr{Type: Mat24}: mgl32.Mat2x4
//	Attr{Type: Mat3}:  mgl32.Mat3
//	Attr{Type: Mat32}: mgl32.Mat3x2
//	Attr{Type: Mat34}: mgl32.Mat3x4
//	Attr{Type: Mat4}:  mgl32.Mat4
//	Attr{Type: Mat42}: mgl32.Mat4x2
//	Attr{Type: Mat43}: mgl32.Mat4x3
//
// No other types are supported.
//
// The Shader must be bound before calling this method.
func (s *Shader) SetUniformAttr(uniform int, value interface{}) (ok bool) {
	if s.uniformLoc[uniform] < 0 {
		return false
	}

	switch s.uniformFmt[uniform].Type {
	case Int:
		value := value.(int32)
		gl.Uniform1iv(s.uniformLoc[uniform], 1, &value)
	case UInt:
		value := value.(uint32)
		gl.Uniform1uiv(s.uniformLoc[uniform], 1, &value)
	case Float:
		value := value.(float32)
		gl.Uniform1fv(s.uniformLoc[uniform], 1, &value)
	case Vec2:
		value := value.(mgl32.Vec2)
		gl.Uniform2fv(s.uniformLoc[uniform], 1, &value[0])
	case Vec3:
		value := value.(mgl32.Vec3)
		gl.Uniform3fv(s.uniformLoc[uniform], 1, &value[0])
	case Vec4:
		value := value.(mgl32.Vec4)
		gl.Uniform4fv(s.uniformLoc[uniform], 1, &value[0])
	case Mat2:
		value := value.(mgl32.Mat2)
		gl.UniformMatrix2fv(s.uniformLoc[uniform], 1, false, &value[0])
	case Mat23:
		value := value.(mgl32.Mat2x3)
		gl.UniformMatrix2x3fv(s.uniformLoc[uniform], 1, false, &value[0])
	case Mat24:
		value := value.(mgl32.Mat2x4)
		gl.UniformMatrix2x4fv(s.uniformLoc[uniform], 1, false, &value[0])
	case Mat3:
		value := value.(mgl32.Mat3)
		gl.UniformMatrix3fv(s.uniformLoc[uniform], 1, false, &value[0])
	case Mat32:
		value := value.(mgl32.Mat3x2)
		gl.UniformMatrix3x2fv(s.uniformLoc[uniform], 1, false, &value[0])
	case Mat34:
		value := value.(mgl32.Mat3x4)
		gl.UniformMatrix3x4fv(s.uniformLoc[uniform], 1, false, &value[0])
	case Mat4:
		value := value.(mgl32.Mat4)
		gl.UniformMatrix4fv(s.uniformLoc[uniform], 1, false, &value[0])
	case Mat42:
		value := value.(mgl32.Mat4x2)
		gl.UniformMatrix4x2fv(s.uniformLoc[uniform], 1, false, &value[0])
	case Mat43:
		value := value.(mgl32.Mat4x3)
		gl.UniformMatrix4x3fv(s.uniformLoc[uniform], 1, false, &value[0])
	default:
		panic("set uniform attr: invalid attribute type")
	}

	return true
}

// Begin binds the Shader program. This is necessary before using the Shader.
func (s *Shader) Begin() {
	s.program.bind()
}

// End unbinds the Shader program and restores the previous one.
func (s *Shader) End() {
	s.program.restore()
}
