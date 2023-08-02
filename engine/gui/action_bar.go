package gui

import (
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/glhf"
)

/*
IDEA:
We want to render a bunch of textured quads at the bottom center of the screen.
*/

type ActionBar struct {
	shader            *glhf.Shader
	texture           *glhf.Texture
	vertex            *glhf.VertexSlice[glhf.GlFloat]
	itemSizeX         int
	itemSizeY         int
	screenWidth       int
	screenHeight      int
	actions           []ActionItem
	bounds            Rectangle
	currentHoverIndex int
}

type Rectangle struct {
	TopLeft     mgl32.Vec2
	BottomRight mgl32.Vec2
}

func (r Rectangle) Contains(x float64, y float64) bool {
	return x >= float64(r.TopLeft.X()) && x <= float64(r.BottomRight.X()) && y >= float64(r.TopLeft.Y()) && y <= float64(r.BottomRight.Y())
}

type ActionItem struct {
	Name         string
	TextureIndex byte
	Execute      func()
	Hotkey       glfw.Key
	Bounds       Rectangle
}

func (i ActionItem) WithBounds(rectangle Rectangle) ActionItem {
	i.Bounds = rectangle
	return i
}

func NewActionBar(shader *glhf.Shader, textureAtlas *glhf.Texture, screenWidth, screenHeight, atlasItemSizeX, atlasItemSizeY int) *ActionBar {
	return &ActionBar{shader: shader, texture: textureAtlas, itemSizeX: atlasItemSizeX, itemSizeY: atlasItemSizeY, screenWidth: screenWidth, screenHeight: screenHeight}
}

func (a *ActionBar) SetActions(actions []ActionItem) {
	a.actions = make([]ActionItem, len(actions))
	vertexCount := len(actions) * 6
	a.vertex = glhf.MakeVertexSlice(a.shader, vertexCount, vertexCount)
	vertexData := make([]glhf.GlFloat, 0, vertexCount*4)
	itemSize := float64(a.screenWidth) / 20.0
	paddingBetween := itemSize * 0.25
	barWidth := itemSize*float64(len(actions)) + paddingBetween*float64(len(actions)-1)
	xOffset := (float64(a.screenWidth) - barWidth) * 0.5
	yPos := float64(a.screenHeight) - itemSize*1.5
	size := glhf.GlFloat(itemSize)
	a.bounds = Rectangle{
		TopLeft:     mgl32.Vec2{float32(xOffset), float32(yPos)},
		BottomRight: mgl32.Vec2{float32(xOffset + barWidth), float32(yPos + itemSize)},
	}
	for actionIndex, action := range actions {
		drawX := glhf.GlFloat(xOffset + itemSize*float64(actionIndex) + paddingBetween*float64(actionIndex))
		drawY := glhf.GlFloat(yPos)
		leftU, topV, rightU, bottomV := a.texture.GetUV(action.TextureIndex, a.itemSizeX, a.itemSizeY)
		// append pos x, pos y, tex u, tex v
		vertexData = append(vertexData, []glhf.GlFloat{
			// first triangle
			// Top-left
			drawX, drawY, leftU, topV,
			// Bottom-left
			drawX, drawY + size, leftU, bottomV,
			// Bottom-right
			drawX + size, drawY + size, rightU, bottomV,

			// second triangle
			// Top-left
			drawX, drawY, leftU, topV,
			// Bottom-right
			drawX + size, drawY + size, rightU, bottomV,
			// Top-right
			drawX + size, drawY, rightU, topV,
		}...)
		a.actions[actionIndex] = action.WithBounds(Rectangle{
			TopLeft:     mgl32.Vec2{float32(drawX), float32(drawY)},
			BottomRight: mgl32.Vec2{float32(drawX + size), float32(drawY + size)},
		})
	}
	a.vertex.Begin()
	a.vertex.SetVertexData(vertexData)
	a.vertex.End()
}

func (a *ActionBar) Draw() {
	if a.actions == nil || len(a.actions) == 0 {
		return
	}
	a.shader.SetUniformAttr(1, mgl32.Translate3D(0, 0, 0))
	a.texture.Begin()
	a.vertex.Begin()
	a.vertex.Draw()
	a.vertex.End()
	a.texture.End()
}

func (a *ActionBar) IsMouseOver(screenX, screenY float64) bool {
	overBar := a.bounds.Contains(screenX, screenY)
	if !overBar {
		return false
	}

	for index, action := range a.actions {
		if action.Bounds.Contains(screenX, screenY) {
			a.currentHoverIndex = index
			return true
		}
	}
	a.currentHoverIndex = -1
	return false
}

func (a *ActionBar) OnMouseClicked(x float64, y float64) {
	if a.IsMouseOver(x, y) {
		a.actions[a.currentHoverIndex].Execute()
	}
}
