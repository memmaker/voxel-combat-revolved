package game

import (
    "github.com/memmaker/battleground/engine/glhf"
    "github.com/memmaker/battleground/engine/voxel"
    "github.com/ojrac/opensimplex-go"
)

type Biome interface {
    Generate() (*voxel.Map, MapMetadata)
    GetName() string
}

type BiomeDesert struct {
    blockLib *BlockLibrary
    texture  *glhf.Texture
}

func NewBiomeDesert(g GameInstance) BiomeDesert {
    texture, bl := g.assets.LoadBlockTextureAtlas("vm_desert_01")
    return BiomeDesert{
        blockLib: bl,
        texture:  texture,
    }
}

func (b BiomeDesert) Generate() *voxel.Map {
    noise := opensimplex.New(32)
    width := int32(4)
    height := int32(4)
    depth := int32(4)
    maxX := width * 16
    maxY := height * 4
    maxZ := depth * 16
    m := voxel.NewMapWithEmptyChunks(width, height, depth, 16, 4)
    m.SetFloorAtHeight(0, b.blockLib.NewBlockFromName("blackstone"))

    for x := int32(0); x < maxX; x++ {
        for z := int32(0); z < maxZ; z++ {
            blockHeight := int32(noise.Eval2(float64(x)/float64(maxX), float64(z)/float64(maxZ)) * 10)
            if blockHeight > maxY {
                blockHeight = maxY
            }
            for y := int32(0); y < blockHeight; y++ {
                block := b.blockLib.NewBlockFromName("sand")
                m.SetBlock(x, y, z, block)
            }
        }
    }

    return m
}
