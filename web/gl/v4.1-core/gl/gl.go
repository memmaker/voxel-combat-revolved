//go:build js

// Package gl is a WebGL2 stand-in for github.com/go-gl/gl/v4.1-core/gl, covering only the calls the game makes.
// It is swapped in via `replace` in go.web.mod (see build_web.sh); the desktop build never sees it.
package gl

import (
	"reflect"
	"strings"
	"syscall/js"
	"unsafe"
)

const (
	EXTENSIONS = 0x1F03
)

type DebugProc func(source, gltype, id, severity uint32, length int32, message string, userParam unsafe.Pointer)

var (
	c       js.Value
	objs    = []js.Value{js.Null()} // GL object names -> JS objects; 0 is null
	locs    []js.Value              // uniform locations
	scratch js.Value                // ArrayBuffer reused for uploads
	u8, f32 js.Value
	i32     js.Value
	u32     js.Value
)

func Init() error {
	c = js.Global().Get("document").Call("querySelector", "canvas").Call("getContext", "webgl2")
	return nil
}

// --- helpers ---

func obj(id uint32) js.Value {
	if int(id) >= len(objs) {
		return js.Null()
	}
	return objs[id]
}

func add(v js.Value) uint32 {
	if v.IsNull() || v.IsUndefined() {
		return 0
	}
	objs = append(objs, v)
	id := uint32(len(objs) - 1)
	v.Set("__id", id)
	return id
}

func gen(n int32, out *uint32, create string) {
	ids := unsafe.Slice(out, n)
	for i := range ids {
		ids[i] = add(c.Call(create))
	}
}

func del(n int32, ids *uint32, fn string) {
	for _, id := range unsafe.Slice(ids, n) {
		if id != 0 {
			c.Call(fn, obj(id))
			objs[id] = js.Null()
		}
	}
}

func ensure(n int) {
	if !scratch.IsUndefined() && scratch.Get("byteLength").Int() >= n {
		return
	}
	size := 1 << 16
	for size < n {
		size <<= 1
	}
	scratch = js.Global().Get("ArrayBuffer").New(size)
	u8 = js.Global().Get("Uint8Array").New(scratch)
	f32 = js.Global().Get("Float32Array").New(scratch)
	i32 = js.Global().Get("Int32Array").New(scratch)
	u32 = js.Global().Get("Uint32Array").New(scratch)
}

// bytes copies n bytes at p into the scratch buffer and returns a Uint8Array view of exactly n bytes.
func bytes(p unsafe.Pointer, n int) js.Value {
	ensure(n)
	view := u8.Call("subarray", 0, n)
	js.CopyBytesToJS(view, unsafe.Slice((*byte)(p), n))
	return view
}

func cstr(p *uint8) string {
	if p == nil {
		return ""
	}
	n := 0
	for *(*uint8)(unsafe.Add(unsafe.Pointer(p), n)) != 0 {
		n++
	}
	return string(unsafe.Slice(p, n))
}

func putStr(s string, bufSize int32, length *int32, out *uint8) {
	if out == nil || bufSize <= 0 {
		return
	}
	n := copy(unsafe.Slice(out, bufSize-1), s)
	unsafe.Slice(out, bufSize)[n] = 0
	if length != nil {
		*length = int32(n)
	}
}

func Ptr(data interface{}) unsafe.Pointer {
	if data == nil {
		return nil
	}
	v := reflect.ValueOf(data)
	switch v.Kind() {
	case reflect.Ptr, reflect.UnsafePointer:
		return v.UnsafePointer()
	case reflect.Slice:
		if v.Len() == 0 {
			return nil
		}
		return v.Index(0).Addr().UnsafePointer()
	case reflect.Uintptr:
		return unsafe.Pointer(uintptr(v.Uint()))
	}
	panic("gl.Ptr: unsupported type " + v.Type().String())
}

func Str(s string) *uint8 {
	if !strings.HasSuffix(s, "\x00") {
		s += "\x00"
	}
	return &[]byte(s)[0]
}

func Strs(strs ...string) (**uint8, func()) {
	ptrs := make([]*uint8, len(strs))
	for i, s := range strs {
		ptrs[i] = Str(s)
	}
	return &ptrs[0], func() {}
}

func GoStr(p *uint8) string { return cstr(p) }

// --- state ---

func Enable(cap uint32)                              { c.Call("enable", cap) }
func Disable(cap uint32)                             { c.Call("disable", cap) }
func BlendFunc(s, d uint32)                          { c.Call("blendFunc", s, d) }
func BlendEquation(mode uint32)                      { c.Call("blendEquation", mode) }
func DepthFunc(f uint32)                             { c.Call("depthFunc", f) }
func DepthMask(flag bool)                            { c.Call("depthMask", flag) }
func CullFace(mode uint32)                           { c.Call("cullFace", mode) }
func ClearColor(r, g, b, a float32)                  { c.Call("clearColor", r, g, b, a) }
func Clear(mask uint32)                              { c.Call("clear", mask) }
func Viewport(x, y, w, h int32)                      { c.Call("viewport", x, y, w, h) }
func Scissor(x, y, w, h int32)                       { c.Call("scissor", x, y, w, h) }
func Flush()                                         { c.Call("flush") }
func PolygonMode(face, mode uint32)                  {} // no wireframe in WebGL
func DebugMessageCallback(DebugProc, unsafe.Pointer) {}

// GetError always reports success: gl.getError() stalls the pipeline in browsers and the game calls it per draw.
func GetError() uint32 { return NO_ERROR }

func GetString(name uint32) *uint8 {
	return Str(c.Call("getParameter", name).String())
}

func GetIntegerv(pname uint32, data *int32) {
	v := c.Call("getParameter", pname)
	switch {
	case v.IsNull() || v.IsUndefined():
		*data = 0
	case v.Type() == js.TypeNumber:
		*data = int32(v.Int())
	default:
		*data = int32(v.Get("__id").Int())
	}
}

// --- shaders ---

func CreateShader(xtype uint32) uint32 { return add(c.Call("createShader", xtype)) }
func CreateProgram() uint32            { return add(c.Call("createProgram")) }
func DeleteShader(s uint32)            { c.Call("deleteShader", obj(s)) }
func DeleteProgram(p uint32)           { c.Call("deleteProgram", obj(p)) }
func CompileShader(s uint32)           { c.Call("compileShader", obj(s)) }
func AttachShader(p, s uint32)         { c.Call("attachShader", obj(p), obj(s)) }
func LinkProgram(p uint32)             { c.Call("linkProgram", obj(p)) }
func UseProgram(p uint32)              { c.Call("useProgram", obj(p)) }

// ShaderSource rewrites desktop GLSL 330 core to GLSL ES 3.00.
func ShaderSource(s uint32, count int32, xstring **uint8, length *int32) {
	var src strings.Builder
	for i, p := range unsafe.Slice(xstring, count) {
		if length != nil && unsafe.Slice(length, count)[i] >= 0 {
			src.Write(unsafe.Slice(p, unsafe.Slice(length, count)[i]))
		} else {
			src.WriteString(cstr(p))
		}
	}
	code := strings.Replace(src.String(), "#version 330 core",
		"#version 300 es\nprecision highp float;\nprecision highp int;\nprecision highp sampler2D;", 1)
	code = strings.ReplaceAll(code, "#extension all : disable", "")
	c.Call("shaderSource", obj(s), code)
}

func GetShaderiv(s uint32, pname uint32, params *int32) {
	if pname == INFO_LOG_LENGTH {
		*params = logLen(c.Call("getShaderInfoLog", obj(s)).String())
		return
	}
	*params = b2i(c.Call("getShaderParameter", obj(s), pname))
}

func GetProgramiv(p uint32, pname uint32, params *int32) {
	if pname == INFO_LOG_LENGTH {
		*params = logLen(c.Call("getProgramInfoLog", obj(p)).String())
		return
	}
	*params = b2i(c.Call("getProgramParameter", obj(p), pname))
}

func logLen(s string) int32 {
	if s == "" {
		return 0
	}
	return int32(len(s) + 1)
}

func b2i(v js.Value) int32 {
	if v.Type() == js.TypeBoolean {
		if v.Bool() {
			return 1
		}
		return 0
	}
	return int32(v.Int())
}

func GetShaderInfoLog(s uint32, bufSize int32, length *int32, infoLog *uint8) {
	msg := c.Call("getShaderInfoLog", obj(s)).String()
	if msg != "" {
		js.Global().Get("console").Call("error", msg)
	}
	putStr(msg, bufSize, length, infoLog)
}

func GetProgramInfoLog(p uint32, bufSize int32, length *int32, infoLog *uint8) {
	msg := c.Call("getProgramInfoLog", obj(p)).String()
	if msg != "" {
		js.Global().Get("console").Call("error", msg)
	}
	putStr(msg, bufSize, length, infoLog)
}

func GetAttribLocation(p uint32, name *uint8) int32 {
	return int32(c.Call("getAttribLocation", obj(p), cstr(name)).Int())
}

func GetUniformLocation(p uint32, name *uint8) int32 {
	loc := c.Call("getUniformLocation", obj(p), cstr(name))
	if loc.IsNull() {
		return -1
	}
	locs = append(locs, loc)
	return int32(len(locs) - 1)
}

func TransformFeedbackVaryings(p uint32, count int32, varyings **uint8, bufferMode uint32) {
	names := make([]any, count)
	for i, v := range unsafe.Slice(varyings, count) {
		names[i] = cstr(v)
	}
	c.Call("transformFeedbackVaryings", obj(p), js.ValueOf(names), bufferMode)
}

// --- uniforms ---

func loc(l int32) js.Value {
	if l < 0 {
		return js.Null()
	}
	return locs[l]
}

func floats(p *float32, n int) js.Value {
	ensure(n * 4)
	js.CopyBytesToJS(u8.Call("subarray", 0, n*4), unsafe.Slice((*byte)(unsafe.Pointer(p)), n*4))
	return f32
}

func uf(fn string, l int32, count int32, p *float32, per int) {
	if l < 0 {
		return
	}
	n := int(count) * per
	c.Call(fn, loc(l), floats(p, n), 0, n)
}

func um(fn string, l int32, count int32, transpose bool, p *float32, per int) {
	if l < 0 {
		return
	}
	n := int(count) * per
	c.Call(fn, loc(l), transpose, floats(p, n), 0, n)
}

func Uniform1fv(l, count int32, v *float32) { uf("uniform1fv", l, count, v, 1) }
func Uniform2fv(l, count int32, v *float32) { uf("uniform2fv", l, count, v, 2) }
func Uniform3fv(l, count int32, v *float32) { uf("uniform3fv", l, count, v, 3) }
func Uniform4fv(l, count int32, v *float32) { uf("uniform4fv", l, count, v, 4) }
func Uniform1iv(l, count int32, v *int32) {
	if l >= 0 {
		floats((*float32)(unsafe.Pointer(v)), int(count))
		c.Call("uniform1iv", loc(l), i32, 0, count)
	}
}
func Uniform1uiv(l, count int32, v *uint32) {
	if l >= 0 {
		floats((*float32)(unsafe.Pointer(v)), int(count))
		c.Call("uniform1uiv", loc(l), u32, 0, count)
	}
}
func UniformMatrix2fv(l, n int32, t bool, v *float32)   { um("uniformMatrix2fv", l, n, t, v, 4) }
func UniformMatrix3fv(l, n int32, t bool, v *float32)   { um("uniformMatrix3fv", l, n, t, v, 9) }
func UniformMatrix4fv(l, n int32, t bool, v *float32)   { um("uniformMatrix4fv", l, n, t, v, 16) }
func UniformMatrix2x3fv(l, n int32, t bool, v *float32) { um("uniformMatrix2x3fv", l, n, t, v, 6) }
func UniformMatrix3x2fv(l, n int32, t bool, v *float32) { um("uniformMatrix3x2fv", l, n, t, v, 6) }
func UniformMatrix2x4fv(l, n int32, t bool, v *float32) { um("uniformMatrix2x4fv", l, n, t, v, 8) }
func UniformMatrix4x2fv(l, n int32, t bool, v *float32) { um("uniformMatrix4x2fv", l, n, t, v, 8) }
func UniformMatrix3x4fv(l, n int32, t bool, v *float32) { um("uniformMatrix3x4fv", l, n, t, v, 12) }
func UniformMatrix4x3fv(l, n int32, t bool, v *float32) { um("uniformMatrix4x3fv", l, n, t, v, 12) }

// --- buffers / vertex arrays ---

func GenBuffers(n int32, b *uint32)         { gen(n, b, "createBuffer") }
func DeleteBuffers(n int32, b *uint32)      { del(n, b, "deleteBuffer") }
func BindBuffer(target, b uint32)           { c.Call("bindBuffer", target, obj(b)) }
func BindBufferBase(target, i, b uint32)    { c.Call("bindBufferBase", target, i, obj(b)) }
func GenVertexArrays(n int32, a *uint32)    { gen(n, a, "createVertexArray") }
func DeleteVertexArrays(n int32, a *uint32) { del(n, a, "deleteVertexArray") }
func BindVertexArray(a uint32)              { c.Call("bindVertexArray", obj(a)) }
func EnableVertexAttribArray(i uint32)      { c.Call("enableVertexAttribArray", i) }
func VertexAttribDivisor(i, d uint32)       { c.Call("vertexAttribDivisor", i, d) }

func BufferData(target uint32, size int, data unsafe.Pointer, usage uint32) {
	if data == nil {
		c.Call("bufferData", target, size, usage)
		return
	}
	c.Call("bufferData", target, bytes(data, size), usage)
}

func BufferSubData(target uint32, offset, size int, data unsafe.Pointer) {
	c.Call("bufferSubData", target, offset, bytes(data, size))
}

func GetBufferSubData(target uint32, offset, size int, data unsafe.Pointer) {
	arr := js.Global().Get("Uint8Array").New(size)
	c.Call("getBufferSubData", target, offset, arr)
	js.CopyBytesToGo(unsafe.Slice((*byte)(data), size), arr)
}

func VertexAttribPointerWithOffset(i uint32, size int32, xtype uint32, norm bool, stride int32, offset uintptr) {
	c.Call("vertexAttribPointer", i, size, xtype, norm, stride, offset)
}

func VertexAttribIPointerWithOffset(i uint32, size int32, xtype uint32, stride int32, offset uintptr) {
	c.Call("vertexAttribIPointer", i, size, xtype, stride, offset)
}

// --- drawing ---

func DrawArrays(mode uint32, first, count int32) { c.Call("drawArrays", mode, first, count) }
func DrawElements(mode uint32, count int32, xtype uint32, indices unsafe.Pointer) {
	c.Call("drawElements", mode, count, xtype, uintptr(indices))
}
func MultiDrawArrays(mode uint32, first, count *int32, drawcount int32) {
	f, n := unsafe.Slice(first, drawcount), unsafe.Slice(count, drawcount)
	for i := range f {
		c.Call("drawArrays", mode, f[i], n[i])
	}
}

// Transform feedback: WebGL2 has it but no DrawTransformFeedback; the particle systems are disabled on the web.
func GenTransformFeedbacks(n int32, ids *uint32)   { gen(n, ids, "createTransformFeedback") }
func BindTransformFeedback(target, id uint32)      { c.Call("bindTransformFeedback", target, obj(id)) }
func BeginTransformFeedback(mode uint32)           { c.Call("beginTransformFeedback", mode) }
func EndTransformFeedback()                        { c.Call("endTransformFeedback") }
func DrawTransformFeedback(mode uint32, id uint32) {}

// --- textures ---

func GenTextures(n int32, t *uint32)    { gen(n, t, "createTexture") }
func DeleteTextures(n int32, t *uint32) { del(n, t, "deleteTexture") }
func BindTexture(target, t uint32)      { c.Call("bindTexture", target, obj(t)) }

func TexParameteri(target, pname uint32, param int32) {
	if param == CLAMP_TO_BORDER {
		param = 0x812F // CLAMP_TO_EDGE
	}
	c.Call("texParameteri", target, pname, param)
}

func TexParameterfv(target, pname uint32, params *float32) {} // only TEXTURE_BORDER_COLOR, unsupported

func TexImage2D(target uint32, level, internalformat, w, h, border int32, format, xtype uint32, pixels unsafe.Pointer) {
	var data any = nil
	if pixels != nil {
		data = bytes(pixels, int(w*h*4)) // the game only uploads RGBA/UNSIGNED_BYTE
	}
	c.Call("texImage2D", target, level, internalformat, w, h, border, format, xtype, data)
}

func TexSubImage2D(target uint32, level, x, y, w, h int32, format, xtype uint32, pixels unsafe.Pointer) {
	c.Call("texSubImage2D", target, level, x, y, w, h, format, xtype, bytes(pixels, int(w*h*4)))
}

func GetTexImage(target uint32, level int32, format, xtype uint32, pixels unsafe.Pointer) {} // debug-only, unsupported

// --- framebuffers ---

func GenFramebuffers(n int32, f *uint32)    { gen(n, f, "createFramebuffer") }
func DeleteFramebuffers(n int32, f *uint32) { del(n, f, "deleteFramebuffer") }
func BindFramebuffer(target, f uint32)      { c.Call("bindFramebuffer", target, obj(f)) }
func FramebufferTexture2D(target, attachment, textarget, t uint32, level int32) {
	c.Call("framebufferTexture2D", target, attachment, textarget, obj(t), level)
}
func BlitFramebuffer(x0, y0, x1, y1, dx0, dy0, dx1, dy1 int32, mask, filter uint32) {
	c.Call("blitFramebuffer", x0, y0, x1, y1, dx0, dy0, dx1, dy1, mask, filter)
}
