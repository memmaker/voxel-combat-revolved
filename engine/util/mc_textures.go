package util

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/memmaker/battleground/engine/glhf"
	"github.com/memmaker/battleground/engine/voxel"
	_ "github.com/spakin/netpbm"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path"
	"sort"
	"strings"
)

// plan:
// 1. read the png files and store them in a map by their name and texture index
// 2. add the bitmaps to one 256x256 bitmap atlas
// 3. return the atlas
// 4. allow for resolving the name to the index

func CreateAtlasFromDirectory(directory string, whiteList []string) (*glhf.Texture, map[string]byte) {
	indices := map[string]byte{}
	pixels := image.NewNRGBA(image.Rect(0, 0, 256, 256)) // iterate over the files in the directory
	textureIndex := 0
	itemSizeX := 0
	itemSizeY := 0
	for _, blockName := range whiteList {
		texturePath := path.Join(directory, blockName+".png")
		file, err := os.Open(texturePath)
		if err != nil {
			println(fmt.Sprintf("[Atlas] Error loading %s from %s", blockName, texturePath))
			continue
		}
		img, _, err := image.Decode(file)
		if err != nil {
			continue
		}
		file.Close()
		itemSizeX = img.Bounds().Dx()
		itemSizeY = img.Bounds().Dy()

		texturesPerRow := 256 / itemSizeX // 256 / 16 = 16
		// copy the image into the atlas
		tilePosX := textureIndex % texturesPerRow
		tilePosY := textureIndex / texturesPerRow
		offsetX := tilePosX * itemSizeX
		offsetY := tilePosY * itemSizeY
		for x := 0; x < itemSizeX; x++ {
			for y := 0; y < itemSizeY; y++ {
				pixels.Set(offsetX+x, offsetY+y, img.At(x, y))
			}
		}
		indices[blockName] = byte(textureIndex)
		println(fmt.Sprintf("[Atlas] %d -> %s", textureIndex, blockName))
		textureIndex++
	}
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
	texture := glhf.NewTexture(256, 256, false, pixels.Pix)
	texture.SetAtlasItemSize(itemSizeX, itemSizeY)
	return texture, indices
}

func CreateAtlasFromPBMs(directory string, glyphWidth, glyphHeight int) (*glhf.Texture, map[rune]uint16) {
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

func createIndicesDirectory(directory string, whiteList []string) map[string]byte {
	indices := map[string]byte{}
	textureIndex := 0
	for _, blockName := range whiteList {
		texturePath := path.Join(directory, blockName+".png")
		if !DoesFileExist(texturePath) {
			continue
		}
		indices[blockName] = byte(textureIndex)
		println(fmt.Sprintf("[Index] %d -> %s", textureIndex, blockName))
		textureIndex++
	}
	return indices
}

func CreateBlockAtlasFromDirectory(directory string, blocksNeeded []string) (*glhf.Texture, map[string]byte) {
	sort.SliceStable(blocksNeeded, func(i, j int) bool {
		return blocksNeeded[i] < blocksNeeded[j]
	})
	var allFaceTextureNames []string
	for i := 0; i < len(blocksNeeded); i++ {
		allFaceTextureNames = append(allFaceTextureNames, tryMCStyleFaceNames(directory, blocksNeeded[i])...)
	}

	return CreateAtlasFromDirectory(directory, allFaceTextureNames)
}

func CreateIndexMapFromDirectory(directory string, blocksNeeded []string) map[string]byte {
	sort.SliceStable(blocksNeeded, func(i, j int) bool {
		return blocksNeeded[i] < blocksNeeded[j]
	})
	var allFaceTextureNames []string
	for i := 0; i < len(blocksNeeded); i++ {
		allFaceTextureNames = append(allFaceTextureNames, tryMCStyleFaceNames(directory, blocksNeeded[i])...)
	}
	return createIndicesDirectory(directory, allFaceTextureNames)
}

func tryMCStyleFaceNames(directory, blockName string) []string {
	var result []string

	texturePath := path.Join(directory, blockName+".png")
	if DoesFileExist(texturePath) {
		result = append(result, blockName)
	}

	for _, suffix := range getMCSuffixes() {
		texturePath = path.Join(directory, blockName+suffix+".png")
		if DoesFileExist(texturePath) {
			result = append(result, blockName+suffix)
		}
	}
	return result
}
func getMCSuffixes() []string {
	return []string{"_top", "_bottom", "_side", "_sides", "_front", "_back", "_side1", "_side2", "_side3"}
}
func MapFaceToTextureIndex(blockname string, face voxel.FaceType, availableSuffixes map[string]byte) byte {
	switch face {
	case voxel.Top:
		if textureIndex, ok := availableSuffixes[blockname+"_top"]; ok {
			return textureIndex
		}
		return availableSuffixes[blockname]
	case voxel.Bottom:
		if textureIndex, ok := availableSuffixes[blockname+"_bottom"]; ok {
			return textureIndex
		}
		return availableSuffixes[blockname]
	case voxel.North:
		if textureIndex, ok := availableSuffixes[blockname+"_back"]; ok {
			return textureIndex
		}
		if textureIndex, ok := availableSuffixes[blockname+"_side2"]; ok {
			return textureIndex
		}
		if textureIndex, ok := availableSuffixes[blockname+"_side"]; ok {
			return textureIndex
		}
		if textureIndex, ok := availableSuffixes[blockname+"_sides"]; ok {
			return textureIndex
		}
		return availableSuffixes[blockname]
	case voxel.South:
		if textureIndex, ok := availableSuffixes[blockname+"_front"]; ok {
			return textureIndex
		}
		if textureIndex, ok := availableSuffixes[blockname+"_side"]; ok {
			return textureIndex
		}
		if textureIndex, ok := availableSuffixes[blockname+"_sides"]; ok {
			return textureIndex
		}
		return availableSuffixes[blockname]
	case voxel.East:
		if textureIndex, ok := availableSuffixes[blockname+"_side3"]; ok {
			return textureIndex
		}
		if textureIndex, ok := availableSuffixes[blockname+"_side"]; ok {
			return textureIndex
		}
		if textureIndex, ok := availableSuffixes[blockname+"_sides"]; ok {
			return textureIndex
		}
		return availableSuffixes[blockname]
	case voxel.West:
		if textureIndex, ok := availableSuffixes[blockname+"_side1"]; ok {
			return textureIndex
		}
		if textureIndex, ok := availableSuffixes[blockname+"_side"]; ok {
			return textureIndex
		}
		if textureIndex, ok := availableSuffixes[blockname+"_sides"]; ok {
			return textureIndex
		}
		return availableSuffixes[blockname]
	}
	return availableSuffixes[blockname]
}
func DoesFileExist(filename string) bool {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return true
}
