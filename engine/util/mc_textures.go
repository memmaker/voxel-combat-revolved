package util

import (
	"fmt"
	"github.com/memmaker/battleground/engine/glhf"
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

func CreateAtlasFromDirectory(directory string, blocksNeeded []string) (*glhf.Texture, map[string]byte) {
	sort.SliceStable(blocksNeeded, func(i, j int) bool {
		return blocksNeeded[i] < blocksNeeded[j]
	})
	debugNames := []string{"debug", "debug2"}
	blocksNeeded = append(debugNames, blocksNeeded...)
	indices := map[string]byte{}
	pixels := image.NewNRGBA(image.Rect(0, 0, 256, 256)) // iterate over the files in the directory
	textureIndex := 0
	for _, blockName := range blocksNeeded {
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
		// copy the image into the atlas
		tilePosX := textureIndex % 16
		tilePosY := textureIndex / 16
		offsetX := tilePosX * 16
		offsetY := tilePosY * 16
		for x := 0; x < 16; x++ {
			for y := 0; y < 16; y++ {
				pixels.Set(offsetX+x, offsetY+y, img.At(x, y))
			}
		}
		indices[blockName] = byte(textureIndex)
		println(fmt.Sprintf("[Atlas] %d -> %s", textureIndex, blockName))
		textureIndex++
	}
	// debug write the atlas to a file
	file, err := os.Create("debug_atlas.png")
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
