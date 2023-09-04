package util

import (
	"fmt"
	"github.com/memmaker/battleground/engine/glhf"
	"github.com/memmaker/battleground/engine/voxel"
	_ "github.com/spakin/netpbm"
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
type NameIndex map[string]byte

func (i NameIndex) WriteAtlasIndex(filename string) {
    file, err := os.Create(filename)
    if err != nil {
        println("could not create debug_atlas.png")
    }
    for name, index := range i {
        file.WriteString(fmt.Sprintf("%s %d\n", name, index))
    }
    file.Close()
}

func NewBlockIndexFromFile(filename string) NameIndex {
    file, err := os.Open(filename)
    if err != nil {
        println("could not open index file")
        return nil
    }
    defer file.Close()
    indices := map[string]byte{}
    var index byte
    var name string
    for {
        _, scanErr := fmt.Fscanf(file, "%s %d\n", &name, &index)
        if scanErr != nil {
            break
        }
        indices[name] = index
    }
    return indices
}
func CreateFixed256PxAtlasFromDirectory(directory string, whiteList []string) (*glhf.Texture, NameIndex) {
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

func createIndicesDirectory(directory string, whiteList []string) NameIndex {
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

func CreateBlockAtlasFromDirectory(directory string, blocksNeeded []string) (*glhf.Texture, NameIndex) {
	sort.SliceStable(blocksNeeded, func(i, j int) bool {
		return blocksNeeded[i] < blocksNeeded[j]
	})
	var allFaceTextureNames []string
	for i := 0; i < len(blocksNeeded); i++ {
		allFaceTextureNames = append(allFaceTextureNames, tryMCStyleFaceNames(directory, blocksNeeded[i])...)
	}

    return CreateFixed256PxAtlasFromDirectory(directory, allFaceTextureNames)
}

func CreateIndexMapFromDirectory(directory string, blocksNeeded []string) NameIndex {
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
func MapFaceToTextureIndex(blockname string, face voxel.FaceType, availableSuffixes NameIndex) byte {
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
