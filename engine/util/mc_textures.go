package util

import (
	"fmt"
	"github.com/memmaker/battleground/engine/glhf"
	"github.com/memmaker/battleground/engine/voxel"
	"image"
	"image/png"
	"os"
	"path"
	"sort"
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
		imageWidth := img.Bounds().Dx()
		imageHeight := img.Bounds().Dy()

		texturesPerRow := 256 / imageWidth // 256 / 16 = 16
		// copy the image into the atlas
		tilePosX := textureIndex % texturesPerRow
		tilePosY := textureIndex / texturesPerRow
		offsetX := tilePosX * imageWidth
		offsetY := tilePosY * imageHeight
		for x := 0; x < imageWidth; x++ {
			for y := 0; y < imageHeight; y++ {
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
	return glhf.NewTexture(256, 256, false, pixels.Pix), indices
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

func tryMCStyleFaceNames(directory, blockName string) []string {
	var result []string

	texturePath := path.Join(directory, blockName+".png")
	if doesFileExist(texturePath) {
		result = append(result, blockName)
	}

	for _, suffix := range getMCSuffixes() {
		texturePath := path.Join(directory, blockName+suffix+".png")
		if doesFileExist(texturePath) {
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
func doesFileExist(filename string) bool {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return true
}
