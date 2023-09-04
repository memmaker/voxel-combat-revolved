package util

import (
    "github.com/go-gl/mathgl/mgl32"
    "github.com/memmaker/battleground/engine/glhf"
)

type BitmapFontMesh struct {
    vertices                *glhf.VertexSlice[glhf.GlFloat]
    pos                     mgl32.Vec3
    shader                  *glhf.Shader
    texture                 *glhf.Texture
    characterToTextureIndex func(character rune) uint16
    scale                   int
}

func NewBitmapFontMesh(shader *glhf.Shader, texture *glhf.Texture, mapper func(character rune) uint16) *BitmapFontMesh {
    b := &BitmapFontMesh{
        shader:                  shader,
        texture:                 texture,
        characterToTextureIndex: mapper,
        scale:                   1,
    }
    return b
}

func (t *BitmapFontMesh) SetAtlasFontMapper(mapper func(character rune) uint16) {
    t.characterToTextureIndex = mapper
}
func (t *BitmapFontMesh) SetScale(scale int) {
    t.scale = scale
}
func (t *BitmapFontMesh) SetText(text []string) {
    if t.texture == nil {
        println("BitmapFontMesh: texture is nil")
        return
    }
    if t.characterToTextureIndex == nil {
        println("BitmapFontMesh: characterToTextureIndex is nil")
        return
    }
    paddingBetweenCharacters := 0
    paddingBetweenLines := 2
    var rawVertices []glhf.GlFloat
    curDrawPos := t.pos
    charWidth, charHeight := t.texture.GetAtlasItemSize()
    charWidth *= t.scale
    charHeight *= t.scale
    gWidth := glhf.GlFloat(charWidth)
    gHeight := glhf.GlFloat(charHeight)

    for _, line := range text {
        for _, character := range line {
            if character == ' ' {
                curDrawPos = curDrawPos.Add(mgl32.Vec3{float32(charWidth + paddingBetweenCharacters), 0, 0})
                continue
            }
            leftU, topV, rightU, bottomV := t.texture.GetUV(t.characterToTextureIndex(character))

            rawVertices = append(rawVertices, []glhf.GlFloat{
                // first triangle
                // Top-left
                glhf.GlFloat(curDrawPos.X()), glhf.GlFloat(curDrawPos.Y()), leftU, topV,
                // Bottom-left
                glhf.GlFloat(curDrawPos.X()), glhf.GlFloat(curDrawPos.Y()) + gHeight, leftU, bottomV,
                // Bottom-right
                glhf.GlFloat(curDrawPos.X()) + gWidth, glhf.GlFloat(curDrawPos.Y()) + gHeight, rightU, bottomV,

                // second triangle
                // Top-left
                glhf.GlFloat(curDrawPos.X()), glhf.GlFloat(curDrawPos.Y()), leftU, topV,
                // Bottom-right
                glhf.GlFloat(curDrawPos.X()) + gWidth, glhf.GlFloat(curDrawPos.Y()) + gHeight, rightU, bottomV,
                // Top-right
                glhf.GlFloat(curDrawPos.X()) + gWidth, glhf.GlFloat(curDrawPos.Y()), rightU, topV,
            }...)

            curDrawPos = curDrawPos.Add(mgl32.Vec3{float32(charWidth + paddingBetweenCharacters), 0, 0})
        }

        curDrawPos = mgl32.Vec3{t.pos.X(), curDrawPos.Y() + float32(charHeight+paddingBetweenLines), t.pos.Z()}
    }
    t.setVertices(rawVertices)
}
func (t *BitmapFontMesh) Draw() {
    if t.vertices == nil {
        return
    }
    t.shader.SetUniformAttr(1, t.GetTransformMatrix())

    t.texture.Begin()

    t.vertices.Begin()
    t.vertices.Draw()
    t.vertices.End()

    t.texture.End()
}

func (t *BitmapFontMesh) SetTexture(text *glhf.Texture) {
    t.texture = text
}

func (t *BitmapFontMesh) setVertices(rawVertices []glhf.GlFloat) {
    vertices := glhf.MakeVertexSlice(t.shader, len(rawVertices)/4, len(rawVertices)/4)
    vertices.Begin()
    vertices.SetVertexData(rawVertices)
    vertices.End()
    t.vertices = vertices
}

func (t *BitmapFontMesh) GetTransformMatrix() mgl32.Mat4 {
    return mgl32.Translate3D(t.pos.X(), t.pos.Y(), t.pos.Z())
}

func (t *BitmapFontMesh) Clear() {
    t.vertices = nil
}
