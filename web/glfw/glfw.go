//go:build js

// Package glfw is a browser stand-in for github.com/go-gl/glfw/v3.3/glfw backed by the page's <canvas>,
// covering only what the game uses. Swapped in via `replace` in go.web.mod (see build_web.sh).
package glfw

import (
	"strings"
	"syscall/js"
)

type (
	Key         int
	Action      int
	ModifierKey int
	MouseButton int
	Hint        int
	InputMode   int
)

const (
	True  = 1
	False = 0

	Release Action = 0
	Press   Action = 1
	Repeat  Action = 2

	MouseButtonLeft   MouseButton = 0
	MouseButtonRight  MouseButton = 1
	MouseButtonMiddle MouseButton = 2

	CursorMode     InputMode = 0x00033001
	RawMouseMotion InputMode = 0x00033005
	CursorNormal             = 0x00034001
	CursorDisabled           = 0x00034003

	ContextVersionMajor Hint = iota
	ContextVersionMinor
	OpenGLProfile
	OpenGLForwardCompatible
	Resizable
	OpenGLDebugContext
	OpenGLCoreProfile = 0
)

const (
	KeySpace  Key = 32
	KeyComma  Key = 44
	Key0      Key = 48
	Key1      Key = 49
	Key2      Key = 50
	Key3      Key = 51
	KeyA      Key = 65
	KeyC      Key = 67
	KeyD      Key = 68
	KeyE      Key = 69
	KeyF      Key = 70
	KeyJ      Key = 74
	KeyM      Key = 77
	KeyN      Key = 78
	KeyQ      Key = 81
	KeyR      Key = 82
	KeyS      Key = 83
	KeyT      Key = 84
	KeyW      Key = 87
	KeyZ      Key = 90
	KeyEscape Key = 256
	KeyEnter  Key = 257
	KeyTab    Key = 258
	KeyDelete Key = 261
	KeyF1     Key = 290
	KeyF2     Key = 291
	KeyF5     Key = 294
	KeyF6     Key = 295
	KeyF7     Key = 296
	KeyF8     Key = 297
	KeyF9     Key = 298
	KeyF10    Key = 299
	KeyF11    Key = 300
	KeyF12    Key = 301
)

// keyFromCode maps KeyboardEvent.code to GLFW key codes.
func keyFromCode(code string) Key {
	switch {
	case strings.HasPrefix(code, "Key") && len(code) == 4:
		return Key(code[3])
	case strings.HasPrefix(code, "Digit") && len(code) == 6:
		return Key(code[5])
	case len(code) >= 2 && code[0] == 'F' && code[1] >= '1' && code[1] <= '9':
		n := 0
		for _, ch := range code[1:] {
			n = n*10 + int(ch-'0')
		}
		return KeyF1 + Key(n-1)
	}
	switch code {
	case "Space":
		return KeySpace
	case "Comma":
		return KeyComma
	case "Escape":
		return KeyEscape
	case "Enter", "NumpadEnter":
		return KeyEnter
	case "Tab":
		return KeyTab
	case "Delete", "Backspace":
		return KeyDelete
	}
	return -1
}

type Monitor struct{}
type VidMode struct{ Width, Height, RefreshRate int }

func GetPrimaryMonitor() *Monitor { return &Monitor{} }
func (m *Monitor) GetVideoMode() *VidMode {
	s := js.Global().Get("screen")
	return &VidMode{s.Get("width").Int(), s.Get("height").Int(), 60}
}

type Window struct {
	canvas      js.Value
	w, h        int
	keys        map[Key]bool
	events      []func()
	cursorMode  int
	mx, my      float64
	keyCb       func(*Window, Key, int, Action, ModifierKey)
	posCb       func(*Window, float64, float64)
	buttonCb    func(*Window, MouseButton, Action, ModifierKey)
	scrollCb    func(*Window, float64, float64)
	frame       chan struct{}
	rafCallback js.Func
}

func Init() error                   { return nil }
func Terminate()                    {}
func WindowHint(Hint, int)          {}
func SwapInterval(int)              {}
func RawMouseMotionSupported() bool { return false }
func GetTime() float64              { return js.Global().Get("performance").Call("now").Float() / 1000 }

var current *Window

func PollEvents() {
	ev := current.events
	current.events = nil
	for _, f := range ev {
		f()
	}
}

func mods(e js.Value) ModifierKey {
	var m ModifierKey
	if e.Get("shiftKey").Bool() {
		m |= 1
	}
	if e.Get("ctrlKey").Bool() {
		m |= 2
	}
	if e.Get("altKey").Bool() {
		m |= 4
	}
	return m
}

// CreateWindow binds to the page's <canvas>; the requested size is ignored in favour of the canvas' CSS size.
func CreateWindow(width, height int, title string, monitor *Monitor, share *Window) (*Window, error) {
	doc := js.Global().Get("document")
	canvas := doc.Call("querySelector", "canvas")
	w := &Window{canvas: canvas, keys: map[Key]bool{}, frame: make(chan struct{}, 1), cursorMode: CursorNormal}
	w.w, w.h = canvas.Get("clientWidth").Int(), canvas.Get("clientHeight").Int()
	if w.w == 0 || w.h == 0 {
		w.w, w.h = width, height
	}
	canvas.Set("width", w.w)
	canvas.Set("height", w.h)
	canvas.Call("getContext", "webgl2", map[string]any{"antialias": true, "alpha": false, "preserveDrawingBuffer": true})
	current = w
	w.rafCallback = js.FuncOf(func(js.Value, []js.Value) any {
		select {
		case w.frame <- struct{}{}:
		default:
		}
		return nil
	})

	locked := func() bool { return doc.Get("pointerLockElement").Equal(canvas) }
	on := func(target js.Value, name string, fn func(e js.Value)) {
		target.Call("addEventListener", name, js.FuncOf(func(_ js.Value, args []js.Value) any { fn(args[0]); return nil }))
	}
	key := func(e js.Value, a Action) {
		k := keyFromCode(e.Get("code").String())
		if k < 0 {
			return
		}
		if k == KeyTab || k >= KeyF1 || k == KeySpace {
			e.Call("preventDefault")
		}
		if a == Press && e.Get("repeat").Bool() {
			a = Repeat
		}
		m := mods(e)
		w.events = append(w.events, func() {
			w.keys[k] = a != Release
			if w.keyCb != nil {
				w.keyCb(w, k, 0, a, m)
			}
		})
	}
	on(js.Global(), "keydown", func(e js.Value) { key(e, Press) })
	on(js.Global(), "keyup", func(e js.Value) { key(e, Release) })
	on(canvas, "contextmenu", func(e js.Value) { e.Call("preventDefault") })
	button := func(e js.Value, a Action) {
		b := MouseButton(e.Get("button").Int())
		switch b {
		case 1:
			b = MouseButtonMiddle
		case 2:
			b = MouseButtonRight
		}
		if a == Press {
			canvas.Call("focus")
			if w.cursorMode == CursorDisabled && !locked() {
				lockPointer(canvas)
			}
		}
		m := mods(e)
		w.events = append(w.events, func() {
			if w.buttonCb != nil {
				w.buttonCb(w, b, a, m)
			}
		})
	}
	on(canvas, "mousedown", func(e js.Value) { button(e, Press) })
	on(js.Global(), "mouseup", func(e js.Value) { button(e, Release) })
	on(js.Global(), "mousemove", func(e js.Value) {
		if locked() {
			w.mx += e.Get("movementX").Float()
			w.my += e.Get("movementY").Float()
		} else {
			r := canvas.Call("getBoundingClientRect")
			w.mx = (e.Get("clientX").Float() - r.Get("left").Float()) * float64(w.w) / r.Get("width").Float()
			w.my = (e.Get("clientY").Float() - r.Get("top").Float()) * float64(w.h) / r.Get("height").Float()
		}
		x, y := w.mx, w.my
		w.events = append(w.events, func() {
			if w.posCb != nil {
				w.posCb(w, x, y)
			}
		})
	})
	on(canvas, "wheel", func(e js.Value) {
		e.Call("preventDefault")
		dy := -e.Get("deltaY").Float() / 100
		w.events = append(w.events, func() {
			if w.scrollCb != nil {
				w.scrollCb(w, 0, dy)
			}
		})
	})
	return w, nil
}

func (w *Window) MakeContextCurrent()                          {}
func (w *Window) GetSize() (int, int)                          { return w.w, w.h }
func (w *Window) ShouldClose() bool                            { return false }
func (w *Window) SetShouldClose(bool)                          {}
func (w *Window) SetTitle(string)                              {}
func (w *Window) GetMonitor() *Monitor                         { return nil }
func (w *Window) SetMonitor(*Monitor, int, int, int, int, int) {}

// SwapBuffers yields to the browser until the next animation frame, which presents the canvas.
func (w *Window) SwapBuffers() {
	js.Global().Call("requestAnimationFrame", w.rafCallback)
	<-w.frame
}

func (w *Window) GetKey(k Key) Action {
	if w.keys[k] {
		return Press
	}
	return Release
}

func (w *Window) GetInputMode(mode InputMode) int {
	if mode == CursorMode {
		return w.cursorMode
	}
	return 0
}

func (w *Window) SetInputMode(mode InputMode, value int) {
	if mode != CursorMode {
		return
	}
	w.cursorMode = value
	doc := js.Global().Get("document")
	if value == CursorDisabled {
		lockPointer(w.canvas) // may need a click (user gesture); mousedown retries
	} else if doc.Get("pointerLockElement").Equal(w.canvas) {
		doc.Call("exitPointerLock")
	}
}

func (w *Window) SetKeyCallback(cb func(*Window, Key, int, Action, ModifierKey)) {
	w.keyCb = cb
}
func (w *Window) SetCursorPosCallback(cb func(*Window, float64, float64)) {
	w.posCb = cb
}
func (w *Window) SetMouseButtonCallback(cb func(*Window, MouseButton, Action, ModifierKey)) {
	w.buttonCb = cb
}
func (w *Window) SetScrollCallback(cb func(*Window, float64, float64)) {
	w.scrollCb = cb
}

// lockPointer requests pointer lock, swallowing the promise rejection browsers raise without a user gesture.
func lockPointer(canvas js.Value) {
	if p := canvas.Call("requestPointerLock"); p.Type() == js.TypeObject {
		p.Call("catch", js.FuncOf(func(js.Value, []js.Value) any { return nil }))
	}
}
