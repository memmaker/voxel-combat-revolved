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
	va   *vertexArray[V]
	i, j int
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
		va: newIndexedVertexArray[GlFloat](shader, cap, nil),
		i:  0,
		j:  len,
	}
}

func MakeIntVertexSlice(shader *Shader, len, cap int, indices []uint32) *VertexSlice[GlInt] {
	if len > cap {
		panic("failed to make vertex slice: len > cap")
	}
	return &VertexSlice[GlInt]{
		va: newIndexedVertexArray[GlInt](shader, cap, indices),
		i:  0,
		j:  len,
	}
}

type GlInt int32
type GlFloat float32

func MakeIndexedVertexSlice(shader *Shader, len, cap int, indices []uint32) *VertexSlice[GlFloat] {
	if len > cap {
		panic("failed to make vertex slice: len > cap")
	}
	return &VertexSlice[GlFloat]{
		va: newIndexedVertexArray[GlFloat](shader, cap, indices),
		i:  0,
		j:  len,
	}
}

// VertexFormat returns the format of vertex attributes inside the underlying vertex array of this
// VertexSlice.
func (vs *VertexSlice[V]) VertexFormat() AttrFormat {
	return vs.va.format
}

// Stride returns the number of float32 elements occupied by one vertex.
func (vs *VertexSlice[V]) Stride() int {
	return vs.va.stride / 4
}

// Len returns the length of the VertexSlice (number of vertices).
func (vs *VertexSlice[V]) Len() int {
	return vs.j - vs.i
}

// Cap returns the capacity of an underlying vertex array.
func (vs *VertexSlice[V]) Cap() int {
	return vs.va.cap - vs.i
}

// SetLen resizes the VertexSlice to length len.
func (vs *VertexSlice[V]) SetLen(len int) {
	vs.End() // vs must have been Begin-ed before calling this method
	*vs = vs.grow(len)
	vs.Begin()
}

// grow returns supplied vs with length changed to len. Allocates new underlying vertex array if
// necessary. The original content is preserved.
func (vs VertexSlice[V]) grow(len int) VertexSlice[V] {
	if len <= vs.Cap() {
		// capacity sufficient
		return VertexSlice[V]{
			va: vs.va,
			i:  vs.i,
			j:  vs.i + len,
		}
	}

	// grow the capacity
	newCap := vs.Cap()
	if newCap < 1024 {
		newCap += newCap
	} else {
		newCap += newCap / 4
	}
	if newCap < len {
		newCap = len
	}
	newVs := VertexSlice[V]{
		va: newIndexedVertexArray[V](vs.va.shader, newCap, vs.va.indices),
		i:  0,
		j:  len,
	}
	// preserve the original content
	newVs.Begin()
	newVs.Slice(0, vs.Len()).SetVertexData(vs.VertexData())
	newVs.End()
	return newVs
}

// Slice returns a sub-slice of this VertexSlice covering the range [i, j) (relative to this
// VertexSlice).
//
// Note, that the returned VertexSlice shares an underlying vertex array with the original
// VertexSlice. Modifying the contents of one modifies corresponding contents of the other.
func (vs *VertexSlice[V]) Slice(i, j int) *VertexSlice[V] {
	if i < 0 || j < i || j > vs.va.cap {
		panic("failed to slice vertex slice: index out of range")
	}
	return &VertexSlice[V]{
		va: vs.va,
		i:  vs.i + i,
		j:  vs.i + j,
	}
}

// SetVertexData sets the contents of the VertexSlice.
//
// The data is a slice of float32's, where each vertex attribute occupies a certain number of
// elements. Namely, Float occupies 1, Vec2 occupies 2, Vec3 occupies 3 and Vec4 occupies 4. The
// attribues in the data slice must be in the same order as in the vertex format of this Vertex
// Slice.
//
// If the length of vertices does not match the length of the VertexSlice, this methdo panics.
func (vs *VertexSlice[V]) SetVertexData(data []V) {
	if len(data)/vs.Stride() != vs.Len() && len(vs.va.indices) == 0 {
		fmt.Println(len(data)/vs.Stride(), vs.Len())
		panic("set vertex data: wrong length of vertices")
	}
	vs.va.setVertexData(vs.i, vs.j, data)
}

// VertexData returns the contents of the VertexSlice.
//
// The data is in the same format as with SetVertexData.
func (vs *VertexSlice[V]) VertexData() []V {
	return vs.va.vertexData(vs.i, vs.j)
}

// Draw draws the content of the VertexSlice.
func (vs *VertexSlice[V]) Draw() {
	vs.va.draw(vs.i, vs.j)
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
		case Int, Float, Vec2, Vec3, Vec4:
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

	gl.GenVertexArrays(1, &va.vao.obj)

	va.vao.bind()

	gl.GenBuffers(1, &va.vbo.obj)
	defer va.vbo.bind().restore()

	emptyData := make([]byte, cap*va.stride)
	gl.BufferData(gl.ARRAY_BUFFER, len(emptyData), gl.Ptr(emptyData), gl.STATIC_DRAW)

	for i, attr := range va.format {
		loc := gl.GetAttribLocation(shader.program.obj, gl.Str(attr.Name+"\x00"))

		var size int32
		glType := uint32(gl.FLOAT)
		switch attr.Type {
		case Int:
			size = 1
			glType = gl.INT
		case Float:
			size = 1
		case Vec2:
			size = 2
		case Vec3:
			size = 3
		case Vec4:
			size = 4
		}

		gl.VertexAttribPointerWithOffset(
			uint32(loc),
			size,
			glType,
			false,
			int32(va.stride),
			uintptr(va.offset[i]),
		)
		gl.EnableVertexAttribArray(uint32(loc))
	}

	va.vao.restore()

	runtime.SetFinalizer(va, (*vertexArray[V]).delete)

	return va
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

func (va *vertexArray[V]) draw(i, j int) {
	if len(va.indices) > 0 {
		gl.DrawElements(va.primitiveType, int32(len(va.indices)), gl.UNSIGNED_INT, gl.Ptr(nil))
	} else {
		gl.DrawArrays(va.primitiveType, int32(i), int32(j-i))
	}
}
func (va *vertexArray[V]) multiDraw(startIndices, counts []int32) {
	gl.MultiDrawArrays(va.primitiveType, &startIndices[0], &counts[0], int32(len(startIndices)))
}

func (va *vertexArray[V]) setVertexData(i, j int, data []V) {
	if j-i == 0 {
		// avoid setting 0 bytes of buffer data
		return
	}
	gl.BufferSubData(gl.ARRAY_BUFFER, i*va.stride, len(data)*4, gl.Ptr(data))
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
