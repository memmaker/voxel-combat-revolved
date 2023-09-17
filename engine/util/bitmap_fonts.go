package util

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/glhf"
	"image"
	"image/color"
	"math"
	"os"
	"path"
	"strings"
)

type BitmapFontMesh struct {
	vertices                *glhf.VertexSlice[glhf.GlFloat]
	pos                     mgl32.Vec3
	shader                  *glhf.Shader
	texture                 *glhf.Texture
	characterToTextureIndex func(character rune) uint16
	scale                   int
	discardColor            mgl32.Vec3
	useDiscardColor         bool
	tintColor               mgl32.Vec3
	useTintColor            bool

	paddingBetweenCharacters int
	paddingBetweenLines      int
	originIsCenter           bool
	isHidden                 bool
	lastText                 []string
}

func NewBitmapFontMesh(shader *glhf.Shader, texture *glhf.Texture, mapper func(character rune) uint16) *BitmapFontMesh {
	b := &BitmapFontMesh{
		shader:                  shader,
		texture:                 texture,
		characterToTextureIndex: mapper,
		scale:                   1,
		paddingBetweenLines:     2,
	}
	return b
}
func (t *BitmapFontMesh) SetDiscardColor(color mgl32.Vec3) {
	t.discardColor = color
	t.useDiscardColor = true
}

func (t *BitmapFontMesh) SetCenterOrigin(center bool) {
	t.originIsCenter = center
}

func (t *BitmapFontMesh) SetTintColor(color mgl32.Vec3) {
	t.tintColor = color
	t.useTintColor = true
}
func (t *BitmapFontMesh) SetAtlasFontMapper(mapper func(character rune) uint16) {
	t.characterToTextureIndex = mapper
}
func (t *BitmapFontMesh) SetScale(scale int) {
	t.scale = scale
}
func (t *BitmapFontMesh) SetText(text string) {
	t.SetMultilineText([]string{text})
}
func (t *BitmapFontMesh) getDimensions(text []string) (int, int) {
	maxChars := 0
	for _, line := range text {
		if len(line) > maxChars {
			maxChars = len(line)
		}
	}
	return maxChars, len(text)
}
func (t *BitmapFontMesh) SetMultilineText(text []string) {
	if t.texture == nil {
		println("BitmapFontMesh: texture is nil")
		return
	}
	if t.characterToTextureIndex == nil {
		println("BitmapFontMesh: characterToTextureIndex is nil")
		return
	}
	t.lastText = text
	t.isHidden = false
	paddingBetweenCharacters := t.paddingBetweenCharacters
	paddingBetweenLines := t.paddingBetweenLines

	curDrawPos := mgl32.Vec3{}
	charWidth, charHeight := t.texture.GetAtlasItemSize()
	charWidth *= t.scale
	charHeight *= t.scale
	gWidth := glhf.GlFloat(charWidth)
	gHeight := glhf.GlFloat(charHeight)

	if t.originIsCenter {
		tWidth, tHeight := t.CalculateSize(text)
		curDrawPos = curDrawPos.Add(mgl32.Vec3{float32(-tWidth / 2), float32(-tHeight / 2), 0})
	}

	var rawVertices []glhf.GlFloat
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

func (t *BitmapFontMesh) CalculateSize(text []string) (int, int) {
	charWidth, charHeight := t.texture.GetAtlasItemSize()
	charWidth *= t.scale
	charHeight *= t.scale
	charsX, charsY := t.getDimensions(text)
	tWidth := (charsX * (charWidth + t.paddingBetweenCharacters)) - t.paddingBetweenCharacters
	tHeight := (charsY * (charHeight + t.paddingBetweenLines)) - t.paddingBetweenLines
	return tWidth, tHeight
}
func (t *BitmapFontMesh) Draw() {
	if t.vertices == nil || t.isHidden {
		return
	}
	t.shader.SetUniformAttr(1, t.GetTransformMatrix())

	if t.useTintColor {
		t.shader.SetUniformAttr(2, t.tintColor.Vec4(1.0)) // tint
	} else {
		t.shader.SetUniformAttr(2, mgl32.Vec4{0, 0, 0, 0})
	}
	if t.useDiscardColor {
		t.shader.SetUniformAttr(3, t.discardColor.Vec4(1.0)) // discard
	} else {
		t.shader.SetUniformAttr(3, mgl32.Vec4{0, 0, 0, 0})
	}

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

func (t *BitmapFontMesh) SetPosition(vec2 mgl32.Vec2) {
	t.pos = mgl32.Vec3{vec2.X(), vec2.Y(), 0}
}

func (t *BitmapFontMesh) Hide() {
	t.isHidden = true
}

func (t *BitmapFontMesh) GetText() string {
	return strings.Join(t.lastText, "\n")
}

type BitmapFontIndex map[rune]uint16

func (i BitmapFontIndex) WriteAtlasIndex(filename string) {
	file, err := os.Create(filename)
	if err != nil {
		println("could not create atlas index file")
		return
	}
	defer file.Close()
	for k, v := range i {
		_, writeErr := file.WriteString(fmt.Sprintf("%d:%d\n", k, v))
		if writeErr != nil {
			println("could not write to atlas index file")
			return
		}
	}
}

func (i BitmapFontIndex) GetMapper() func(character rune) uint16 {
	return func(character rune) uint16 {
		return i[character]
	}
}
func NewBitmapFontIndexFromFile(filename string) BitmapFontIndex {
	index := map[rune]uint16{}
	file, err := os.Open(filename)
	if err != nil {
		println("could not open atlas index file")
		return index
	}
	defer file.Close()
	var k rune
	var v uint16
	for {
		_, scanError := fmt.Fscanf(file, "%d:%d\n", &k, &v)
		if scanError != nil {
			if scanError.Error() != "EOF" {
				LogIOError("could not scan atlas index file")
			}
			break
		}
		index[k] = v
	}
	return index
}

type AtlasDescription struct {
	PositionOfCapitalA     [2]int
	PositionOfZero         *[2]int
	PositionOfSpecialChain *[2]int
	//Rows                  int
	Cols                  int
	SpecialCharacterChain []rune
}

func NewIndexFromDescription(desc AtlasDescription) BitmapFontIndex {
	result := map[rune]uint16{}

	//rows := desc.Rows
	cols := desc.Cols

	positionOfCapitalA := desc.PositionOfCapitalA
	indexOfCapitalA := positionOfCapitalA[0] + positionOfCapitalA[1]*cols

	indexOfZero := indexOfCapitalA + 26
	if desc.PositionOfZero != nil {
		positionOfZero := *desc.PositionOfZero
		indexOfZero = positionOfZero[0] + positionOfZero[1]*cols
	}
	indexOfSpecialChain := indexOfZero + 10
	if desc.PositionOfSpecialChain != nil {
		positionOfSpecialChain := *desc.PositionOfSpecialChain
		indexOfSpecialChain = positionOfSpecialChain[0] + positionOfSpecialChain[1]*cols
	}

	for i := 0; i < 26; i++ {
		result[rune(i+65)] = uint16(indexOfCapitalA + i)
	}

	for i := 0; i < 10; i++ {
		result[rune(i+48)] = uint16(indexOfZero + i)
	}

	for i, special := range desc.SpecialCharacterChain {
		result[special] = uint16(indexOfSpecialChain + i)
	}

	return result
}

func CreateAtlasFromPBMs(directory string, glyphWidth, glyphHeight int) (*glhf.Texture, BitmapFontIndex) {
	// 967 files
	// 8x14 pixels

	// 8*16 = 128
	// 14*16 = 224
	indices := map[rune]uint16{}
	textureIndex := uint16(0)
	entries, readError := os.ReadDir(directory)
	if readError != nil {
		println(fmt.Sprintf("[Atlas] Error reading directory %s", directory))
		return nil, nil
	}
	fileCount := len(entries)
	squareCount := math.Ceil(math.Sqrt(float64(fileCount)))

	padding := int(0)

	atlasWidth := int(math.Ceil(squareCount * float64(glyphWidth+padding)))
	atlasHeight := int(math.Ceil(squareCount * float64(glyphHeight+padding)))

	if atlasWidth%4 != 0 {
		atlasWidth += 4 - (atlasWidth % 4)
	}

	if atlasHeight%4 != 0 {
		atlasHeight += 4 - (atlasHeight % 4)
	}

	pixels := image.NewNRGBA(image.Rect(0, 0, atlasWidth, atlasHeight)) // iterate over the files in the directory
	var drawColor color.Color
	for _, dirEntry := range entries {
		extension := strings.ToLower(path.Ext(dirEntry.Name()))
		if extension != ".pbm" {
			continue
		}
		nameWithoutExtension := strings.TrimSuffix(dirEntry.Name(), extension)
		hexString := nameWithoutExtension

		glyph := runeFromHexString(hexString)

		texturePath := path.Join(directory, dirEntry.Name())
		file, err := os.Open(texturePath)
		if err != nil {
			println(fmt.Sprintf("[Atlas] Error loading %s from %s", dirEntry, texturePath))
			continue
		}
		img, _, err := image.Decode(file)
		if err != nil {
			continue
		}
		file.Close()
		if glyphWidth != img.Bounds().Dx() {
			println(fmt.Sprintf("[Atlas] Error loading %s from %s: width mismatch", dirEntry, texturePath))
			continue
		}
		if glyphHeight != img.Bounds().Dy() {
			println(fmt.Sprintf("[Atlas] Error loading %s from %s: height mismatch", dirEntry, texturePath))
			continue
		}

		texturesPerRow := uint16(atlasWidth / glyphWidth) // 256 / 16 = 16
		// copy the image into the atlas
		tilePosX := textureIndex % texturesPerRow
		tilePosY := textureIndex / texturesPerRow
		offsetX := tilePosX * (uint16(glyphWidth) + uint16(padding))
		offsetY := tilePosY * (uint16(glyphHeight) + uint16(padding))
		for x := 0; x < glyphWidth; x++ {
			for y := 0; y < glyphHeight; y++ {
				r, _, _, _ := img.At(x, y).RGBA()

				drawColor = color.Transparent
				if r < 0x8000 { // black
					drawColor = color.White
				}
				pixels.Set(int(offsetX)+x, int(offsetY)+y, drawColor)
			}
		}
		indices[glyph] = textureIndex
		println(fmt.Sprintf("[Atlas] %U %d -> %s", glyph, textureIndex, dirEntry))
		textureIndex++
	}
	/*
		// debug write the atlas to a file
		file, err := os.Create(path.Join(directory, "debug_atlas.png"))
		if err != nil {
			println("could not create debug_atlas.png")
		}
		err = png.Encode(file, pixels)
		if err != nil {
			println("could not encode debug_atlas.png")
		}
		file.Close()
	*/

	texture := glhf.NewTexture(atlasWidth, atlasHeight, false, pixels.Pix)
	texture.SetAtlasItemSize(glyphWidth, glyphHeight)
	return texture, indices
}

func runeFromHexString(hexString string) rune {
	if len(hexString) < 8 {
		for i := len(hexString); i < 8; i++ {
			hexString = "0" + hexString
		}
	}
	codePointAsBytes, _ := hex.DecodeString(hexString)
	var glyph rune
	reader := bytes.NewReader(codePointAsBytes)
	readError := binary.Read(reader, binary.BigEndian, &glyph)
	if readError != nil {
		println(fmt.Sprintf("[runeFromHexString] Error reading %s", hexString))
	}
	return glyph
}
