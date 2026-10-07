package client

import (
	"fmt"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/glhf"
	"github.com/memmaker/battleground/engine/gui"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/game"
)

// palette indices for the UI rect batch
const (
	uiBackground = iota
	uiPanel
	uiPanelHover
	uiAccent
	uiAccentDim
	uiWarning
	uiDisabled
	uiShade
)

var uiPalette = []mgl32.Vec4{
	{0.03, 0.05, 0.07, 1},
	{0.07, 0.11, 0.14, 0.96},
	{0.11, 0.20, 0.24, 1},
	game.ColorTechTeal,
	{0.10, 0.35, 0.33, 1},
	game.ColorNegativeRed,
	{0.20, 0.22, 0.24, 1},
	{0.0, 0.0, 0.0, 0.6},
}

var (
	uiTextColor     = mgl32.Vec3{0.85, 0.95, 0.95}
	uiTextDim       = mgl32.Vec3{0.45, 0.6, 0.6}
	uiTextHighlight = game.ColorTechTeal.Vec3()
)

type uiTextKey struct {
	text  string
	scale int
	color mgl32.Vec3
}

// UI is a small immediate-mode GUI for menus and overlays: every frame the screen is described
// with Button/Label calls and drawn with one rect batch plus cached text meshes.
type UI struct {
	shader      *glhf.Shader
	rects       *gui.RectBatch
	fontTexture *glhf.Texture
	fontMapper  func(rune) uint16
	charWidth   int
	charHeight  int

	textCache    map[uiTextKey]*util.BitmapFontMesh
	textUsed     map[uiTextKey]bool
	queuedLabels []queuedLabel

	screenWidth, screenHeight int
	mouseX, mouseY            float64
	clicked                   bool
	consumedClick             bool
}

type queuedLabel struct {
	mesh *util.BitmapFontMesh
	pos  mgl32.Vec2
}

func NewUI(shader *glhf.Shader, fontTexture *glhf.Texture, fontMapper func(rune) uint16, charWidth, charHeight int) *UI {
	return &UI{
		shader:      shader,
		rects:       gui.NewRectBatchRGBA(shader, uiPalette),
		fontTexture: fontTexture,
		fontMapper:  fontMapper,
		charWidth:   charWidth,
		charHeight:  charHeight,
		textCache:   make(map[uiTextKey]*util.BitmapFontMesh),
		textUsed:    make(map[uiTextKey]bool),
	}
}

// Begin starts a new frame. clicked is true for exactly one frame after a left mouse release.
func (u *UI) Begin(screenWidth, screenHeight int, mouseX, mouseY float64, clicked bool) {
	u.screenWidth, u.screenHeight = screenWidth, screenHeight
	u.mouseX, u.mouseY = mouseX, mouseY
	u.clicked = clicked
	u.consumedClick = false
	u.rects.Clear()
	u.queuedLabels = u.queuedLabels[:0]
	u.textUsed = make(map[uiTextKey]bool)
}

// ConsumedClick reports whether a widget handled this frame's click.
func (u *UI) ConsumedClick() bool {
	return u.consumedClick
}

func (u *UI) TextSize(text string, scale int) (float32, float32) {
	return float32(len([]rune(text)) * u.charWidth * scale), float32(u.charHeight * scale)
}

func (u *UI) mesh(text string, scale int, color mgl32.Vec3) *util.BitmapFontMesh {
	key := uiTextKey{text: text, scale: scale, color: color}
	u.textUsed[key] = true
	if mesh, cached := u.textCache[key]; cached {
		return mesh
	}
	mesh := util.NewBitmapFontMesh(u.shader, u.fontTexture, u.fontMapper)
	mesh.SetScale(scale)
	mesh.SetTintColor(color)
	mesh.SetText(text)
	u.textCache[key] = mesh
	return mesh
}

func (u *UI) Label(x, y float32, text string, scale int, color mgl32.Vec3) {
	if text == "" {
		return
	}
	u.queuedLabels = append(u.queuedLabels, queuedLabel{mesh: u.mesh(text, scale, color), pos: mgl32.Vec2{x, y}})
}

// CenteredLabel draws text horizontally centered on centerX.
func (u *UI) CenteredLabel(centerX, y float32, text string, scale int, color mgl32.Vec3) {
	width, _ := u.TextSize(text, scale)
	u.Label(centerX-width/2, y, text, scale, color)
}

func (u *UI) Panel(x, y, width, height float32) {
	u.rects.Add(x, y, width, height, uiPanel)
	u.rects.AddFrame(x, y, width, height, 1, uiAccentDim)
}

// Shade darkens the whole screen behind an overlay.
func (u *UI) Shade() {
	u.rects.Add(0, 0, float32(u.screenWidth), float32(u.screenHeight), uiShade)
}

func (u *UI) Fill(x, y, width, height float32, colorIndex int) {
	u.rects.Add(x, y, width, height, colorIndex)
}

func (u *UI) isHovered(x, y, width, height float32) bool {
	mx, my := float32(u.mouseX), float32(u.mouseY)
	return mx >= x && mx <= x+width && my >= y && my <= y+height
}

// Button draws a button and returns true when it was clicked this frame.
func (u *UI) Button(x, y, width, height float32, text string, enabled bool) bool {
	hovered := enabled && u.isHovered(x, y, width, height)
	background := uiPanel
	border := uiAccentDim
	textColor := uiTextColor
	if !enabled {
		background, border, textColor = uiDisabled, uiDisabled, uiTextDim
	} else if hovered {
		background, border, textColor = uiPanelHover, uiAccent, uiTextHighlight
	}
	u.rects.Add(x, y, width, height, background)
	u.rects.AddFrame(x, y, width, height, 1, border)
	textWidth, textHeight := u.TextSize(text, 1)
	u.Label(x+(width-textWidth)/2, y+(height-textHeight)/2, text, 1, textColor)
	if hovered && u.clicked && !u.consumedClick {
		u.consumedClick = true
		GetAudio().Play(SoundUIClick)
		return true
	}
	return false
}

// Selector draws "< value >" arrows to cycle through options and returns the new index.
func (u *UI) Selector(x, y, width, height float32, label string, options []string, index int) int {
	if len(options) == 0 {
		return index
	}
	labelWidth := width * 0.4
	u.Label(x, y+(height-float32(u.charHeight))/2, label, 1, uiTextDim)
	arrowWidth := height
	valueX := x + labelWidth
	valueWidth := width - labelWidth - arrowWidth*2 - 4
	if u.Button(valueX, y, arrowWidth, height, "<", true) {
		index = (index - 1 + len(options)) % len(options)
	}
	u.rects.Add(valueX+arrowWidth+2, y, valueWidth, height, uiPanel)
	value := options[index]
	textWidth, _ := u.TextSize(value, 1)
	u.Label(valueX+arrowWidth+2+(valueWidth-textWidth)/2, y+(height-float32(u.charHeight))/2, value, 1, uiTextColor)
	if u.Button(valueX+arrowWidth+valueWidth+4, y, arrowWidth, height, ">", true) {
		index = (index + 1) % len(options)
	}
	return index
}

// Toggle draws an on/off switch and returns the new value.
func (u *UI) Toggle(x, y, width, height float32, label string, value bool) bool {
	options := []string{"OFF", "ON"}
	index := 0
	if value {
		index = 1
	}
	return u.Selector(x, y, width, height, label, options, index) == 1
}

// TextField draws an editable text field. Typing is handled via UI.HandleTypedKey on the focused field.
func (u *UI) TextField(x, y, width, height float32, label, value string, focused bool) bool {
	labelWidth := width * 0.4
	u.Label(x, y+(height-float32(u.charHeight))/2, label, 1, uiTextDim)
	fieldX, fieldWidth := x+labelWidth, width-labelWidth
	border := uiAccentDim
	if focused {
		border = uiAccent
	}
	u.rects.Add(fieldX, y, fieldWidth, height, uiBackground)
	u.rects.AddFrame(fieldX, y, fieldWidth, height, 1, border)
	shown := value
	if focused {
		shown += "_"
	}
	u.Label(fieldX+6, y+(height-float32(u.charHeight))/2, shown, 1, uiTextColor)
	if u.isHovered(fieldX, y, fieldWidth, height) && u.clicked && !u.consumedClick {
		u.consumedClick = true
		return true
	}
	return false
}

// Draw renders all queued widgets. The gui shader must be active.
func (u *UI) Draw() {
	gl.Disable(gl.DEPTH_TEST)
	u.shader.SetUniformAttr(0, util.Get2DPixelCoordOrthographicProjectionMatrix(u.screenWidth, u.screenHeight))
	u.rects.Draw()
	for _, label := range u.queuedLabels {
		label.mesh.SetPosition(label.pos)
		label.mesh.Draw()
	}
	gl.Enable(gl.DEPTH_TEST)
	// forget text meshes that were not used this frame, so changing texts don't pile up
	for key := range u.textCache {
		if !u.textUsed[key] {
			delete(u.textCache, key)
		}
	}
}

func percent(value float64) string {
	return fmt.Sprintf("%d%%", int(value*100+0.5))
}
