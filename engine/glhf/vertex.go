package glhf

import (
	"fmt"
	"runtime"

	"github.com/faiface/mainthread"
	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/pkg/errors"
)

// VertexSlice points to a portion of (or possibly whole) vertex array. It is used as a pointer,
// contrary to Go's builtin slices. This is, so that append can be 'in-place'. That's for the good,
// because Begin/End-ing a VertexSlice would become super confusing, if append returned a new
// VertexSlice.
//
// It also implements all basic slice-like operations: appending, sub-slicing, etc.
//
// Note that you need to Begin a VertexSlice before getting or updating it's elements or drawing it.
// After you're done with it, you need to End it.
type VertexSlice[V any] struct {
	va                   *vertexArray[V]
	startIndex, endIndex int
}

// MakeVertexSlice allocates a new vertex array with specified capacity and returns a VertexSlice
// that points to it's first len elements.
//
// Note, that a vertex array is specialized for a specific shader and can't be used with another
// shader.
func MakeVertexSlice(shader *Shader, len, cap int) *VertexSlice[GlFloat] {
	if len > cap {
		panic("failed to make vertex slice: len > cap")
	}
	return &VertexSlice[GlFloat]{
		va:         newIndexedVertexArray[GlFloat](shader, cap, nil),
		startIndex: 0,
		endIndex:   len,
	}
}

func MakeIntVertexSlice(shader *Shader, len, cap int, indices []uint32) *VertexSlice[GlInt] {
	if len > cap {
		panic("failed to make vertex slice: len > cap")
	}
	return &VertexSlice[GlInt]{
		va:         newIndexedVertexArray[GlInt](shader, cap, indices),
		startIndex: 0,
		endIndex:   len,
	}
}

func MakeUIntVertexSlice(shader *Shader, len, cap int, indices []uint32) *VertexSlice[GlUInt] {
	if len > cap {
		panic("failed to make vertex slice: len > cap")
	}
	return &VertexSlice[GlUInt]{
		va:         newIndexedVertexArray[GlUInt](shader, cap, indices),
		startIndex: 0,
		endIndex:   len,
	}
}

type GlInt int32
type GlUInt uint32
type GlFloat float32

func MakeIndexedVertexSlice(shader *Shader, len, cap int, indices []uint32) *VertexSlice[GlFloat] {
	if len > cap {
		panic("failed to make vertex slice: len > cap")
	}
	return &VertexSlice[GlFloat]{
		va:         newIndexedVertexArray[GlFloat](shader, cap, indices),
		startIndex: 0,
		endIndex:   len,
	}
}

// VertexFormat returns the format of vertex attributes inside the underlying vertex array of this
// VertexSlice.
func (vs *VertexSlice[V]) VertexFormat() AttrFormat {
	return vs.va.format
}

// Stride returns the number of float32/int32 elements occupied by one vertex.
func (vs *VertexSlice[V]) Stride() int {
	return vs.va.stride / 4
}

// Len returns the length of the VertexSlice (number of vertices).
func (vs *VertexSlice[V]) Len() int {
	return vs.endIndex - vs.startIndex
}

// Cap returns the capacity of an underlying vertex array.
func (vs *VertexSlice[V]) Cap() int {
	return vs.va.cap - vs.startIndex
}

// Slice returns a sub-slice of this VertexSlice covering the range [startIndex, endIndex) (relative to this
// VertexSlice).
//
// Note, that the returned VertexSlice shares an underlying vertex array with the original
// VertexSlice. Modifying the contents of one modifies corresponding contents of the other.
func (vs *VertexSlice[V]) Slice(i, j int) *VertexSlice[V] {
	if i < 0 || j < i || j > vs.va.cap {
		panic("failed to slice vertex slice: index out of range")
	}
	return &VertexSlice[V]{
		va:         vs.va,
		startIndex: vs.startIndex + i,
		endIndex:   vs.startIndex + j,
	}
}

// SetVertexData sets the contents of the VertexSlice.
//
// The data is a slice of float32's or int32's, where each vertex attribute occupies a certain number of
// elements. Namely, Float occupies 1, Vec2 occupies 2, Vec3 occupies 3 and Vec4 occupies 4. The
// attribues in the data slice must be in the same order as in the vertex format of this Vertex
// Slice.
//
// If the length of vertices does not match the length of the VertexSlice, this method panics.
func (vs *VertexSlice[V]) SetVertexData(data []V) {
	if len(data)/vs.Stride() != vs.Len() && len(vs.va.indices) == 0 {
		fmt.Println(len(data)/vs.Stride(), vs.Len())
		panic("set vertex data: wrong length of vertices")
	}
	vs.va.setVertexDataWithOffset(vs.startIndex, vs.endIndex, data)
}

// VertexData returns the contents of the VertexSlice.
//
// The data is in the same format as with SetVertexData.
func (vs *VertexSlice[V]) VertexData() []V {
	return vs.va.vertexData(vs.startIndex, vs.endIndex)
}

// Draw draws the content of the VertexSlice.
func (vs *VertexSlice[V]) Draw() {
	vs.va.draw(vs.startIndex, vs.endIndex)
}

func (vs *VertexSlice[V]) MultiDraw(startIndices, counts []int32) {
	vs.va.multiDraw(startIndices, counts)
}

// Begin binds the underlying vertex array. Calling this method is necessary before using the VertexSlice.
func (vs *VertexSlice[V]) Begin() {
	vs.va.begin()
}

// End unbinds the underlying vertex array. Call this method when you're done with VertexSlice.
func (vs *VertexSlice[V]) End() {
	vs.va.end()
}

func (vs *VertexSlice[V]) SetPrimitiveType(glPrimitiveType uint32) {
	vs.va.setPrimitiveType(glPrimitiveType)
}

type vertexArray[V any] struct {
	vao, vbo      binder
	cap           int
	format        AttrFormat
	stride        int
	offset        []int
	shader        *Shader
	indices       []uint32
	ibo           binder
	primitiveType uint32
}

const vertexArrayMinCap = 4

func newIndexedVertexArray[V any](shader *Shader, cap int, indices []uint32) *vertexArray[V] {
	if cap < vertexArrayMinCap {
		cap = vertexArrayMinCap
	}

	va := &vertexArray[V]{
		primitiveType: gl.TRIANGLES,
		vao: binder{
			restoreLoc: gl.VERTEX_ARRAY_BINDING,
			bindFunc: func(obj uint32) {
				gl.BindVertexArray(obj)
			},
		},
		vbo: binder{
			restoreLoc: gl.ARRAY_BUFFER_BINDING,
			bindFunc: func(obj uint32) {
				gl.BindBuffer(gl.ARRAY_BUFFER, obj)
			},
		},
		ibo: binder{
			restoreLoc: gl.ELEMENT_ARRAY_BUFFER_BINDING,
			bindFunc: func(obj uint32) {
				gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, obj)
			},
		},
		indices: indices,
		cap:     cap,
		format:  shader.VertexFormat(),
		stride:  shader.VertexFormat().Size(),
		offset:  make([]int, len(shader.VertexFormat())),
		shader:  shader,
	}

	offset := 0
	for i, attr := range va.format {
		switch attr.Type {
		case Int, UInt, Float, Vec2, Vec3, Vec4, Mat4:
		default:
			panic(errors.New("failed to create vertex array: invalid attribute type"))
		}
		va.offset[i] = offset
		offset += attr.Type.Size()
	}
	if len(indices) > 0 {
		gl.GenBuffers(1, &va.ibo.obj) // create buffer and get the name
		glError := gl.GetError()
		if glError != gl.NO_ERROR {
			println("failed to create vertex array: failed to generate index buffer:", glError)
		}
		va.ibo.bind()
		gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, len(indices)*4, gl.Ptr(indices), gl.STATIC_DRAW) // set the index buffer data
		glError = gl.GetError()
		if glError != gl.NO_ERROR {
			println("failed to create vertex array: failed to set index buffer data:", glError)
		}
		va.ibo.restore()
	}

	gl.GenVertexArrays(1, &va.vao.obj) // create a vertex array object

	va.vao.bind()

	gl.GenBuffers(1, &va.vbo.obj) // create buffer
	defer va.vbo.bind().restore()

	emptyData := make([]byte, cap*va.stride) // creaty an empty buffer of the right size
	gl.BufferData(gl.ARRAY_BUFFER, len(emptyData), gl.Ptr(emptyData), gl.STATIC_DRAW)

	va.setAttributesForArray()

	va.vao.restore()

	runtime.SetFinalizer(va, (*vertexArray[V]).delete)

	return va
}

func (va *vertexArray[V]) setAttributesForArray() {
	for i, attr := range va.format {
		loc := gl.GetAttribLocation(va.shader.program.obj, gl.Str(attr.Name+"\x00")) // get variable location index from shader

		var size int32
		glType := uint32(gl.FLOAT)
		isFloat := true
		isArrayOfArrays := false
		switch attr.Type {
		case Int:
			size = 1
			glType = gl.INT
			isFloat = false
		case UInt:
			size = 1
			glType = gl.UNSIGNED_INT
			isFloat = false
		case Float:
			size = 1
		case Vec2:
			size = 2
		case Vec3:
			size = 3
		case Vec4:
			size = 4
		case Mat4:
			size = 4
			isArrayOfArrays = true
		}

		if isArrayOfArrays {
			startLocation := uint32(loc)
			for matrixOffset := uint32(0); matrixOffset < 4; matrixOffset++ {
				gl.VertexAttribPointerWithOffset(startLocation+matrixOffset, size, gl.FLOAT, false, int32(va.stride), uintptr(matrixOffset*uint32(size)*SizeOfFloat32))
				gl.VertexAttribDivisor(startLocation+matrixOffset, 1)
				gl.EnableVertexAttribArray(startLocation + matrixOffset)
			}
		} else if isFloat {
			gl.VertexAttribPointerWithOffset(
				uint32(loc),
				size,
				glType,
				false,
				int32(va.stride),
				uintptr(va.offset[i]),
			)
			gl.EnableVertexAttribArray(uint32(loc)) // Enable and use this attribute for rendering the associated array
		} else {
			gl.VertexAttribIPointerWithOffset(
				uint32(loc),
				size,
				glType,
				int32(va.stride),
				uintptr(va.offset[i]),
			)
			gl.EnableVertexAttribArray(uint32(loc)) // Enable and use this attribute for rendering the associated array
		}
	}
}

func (va *vertexArray[V]) delete() {
	mainthread.CallNonBlock(func() {
		gl.DeleteVertexArrays(1, &va.vao.obj)
		gl.DeleteBuffers(1, &va.vbo.obj)
		gl.DeleteBuffers(1, &va.ibo.obj)
	})
}

func (va *vertexArray[V]) begin() {
	va.vao.bind()
	va.vbo.bind()
	if len(va.indices) > 0 {
		va.ibo.bind()
	}
}

func (va *vertexArray[V]) end() {
	if len(va.indices) > 0 {
		va.ibo.restore()
	}
	va.vbo.restore()
	va.vao.restore()
}

func (va *vertexArray[V]) draw(startIndex, endIndex int) {
	if len(va.indices) > 0 {
		gl.DrawElements(va.primitiveType, int32(len(va.indices)), gl.UNSIGNED_INT, gl.Ptr(nil))
	} else {
		gl.DrawArrays(va.primitiveType, int32(startIndex), int32(endIndex-startIndex))
	}
}

func (va *vertexArray[V]) drawFromFeedback() {
	gl.DrawTransformFeedback(va.primitiveType, va.vbo.obj)
}
func (va *vertexArray[V]) multiDraw(startIndices, counts []int32) {
	gl.MultiDrawArrays(va.primitiveType, &startIndices[0], &counts[0], int32(len(startIndices)))
}

func (va *vertexArray[V]) setVertexDataWithOffset(i, j int, data []V) {
	if j-i == 0 {
		// avoid setting 0 bytes of buffer data
		return
	}
	gl.BufferSubData(gl.ARRAY_BUFFER, i*va.stride, len(data)*4, gl.Ptr(data))
}

func (va *vertexArray[V]) setVertexData(data []V) {
	if len(data) == 0 {
		// avoid setting 0 bytes of buffer data
		return
	}
	gl.BufferSubData(gl.ARRAY_BUFFER, 0, len(data)*4, gl.Ptr(data))
}

func (va *vertexArray[V]) vertexData(i, j int) []V {
	if j-i == 0 {
		// avoid getting 0 bytes of buffer data
		return nil
	}
	data := make([]V, (j-i)*va.stride/4)
	gl.GetBufferSubData(gl.ARRAY_BUFFER, i*va.stride, len(data)*4, gl.Ptr(data))
	return data
}

func (va *vertexArray[V]) setPrimitiveType(primitiveType uint32) {
	va.primitiveType = primitiveType
}
