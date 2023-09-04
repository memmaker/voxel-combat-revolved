package game

import (
    "github.com/go-gl/mathgl/mgl32"
    "github.com/memmaker/battleground/engine/glhf"
    "github.com/memmaker/battleground/engine/util"
    "io"
    "os"
    "path"
)

type Assets struct {
    paths map[AssetType]string
}

type AssetType int

const (
    AssetTypeBlockTextures AssetType = iota
    AssetTypeBitmapFonts
    AssetTypeMeshes
    AssetTypeMaps
    AssetTypeSkins
)

func NewAssets() *Assets {
    return &Assets{
        paths: map[AssetType]string{
            AssetTypeBlockTextures: "./assets/textures/blocks/",
            AssetTypeBitmapFonts:   "./assets/fonts/",
            AssetTypeMeshes:        "./assets/models/",
            AssetTypeMaps:          "./assets/maps/",
            AssetTypeSkins:         "./assets/textures/skins/",
        },
    }
}

func (a *Assets) LoadBlockTextureAtlas(filename string) (*glhf.Texture, util.NameIndex) {
    filePath := path.Join(a.paths[AssetTypeBlockTextures], filename)
    texture := mustLoadTexture(filePath + ".png")
    texture.SetAtlasItemSize(16, 16)
    indexMap := util.NewBlockIndexFromFile(filePath + ".idx")
    return texture, indexMap
}

func (a *Assets) LoadBitmapFont(fontName string, glyphWidth int, glyphHeight int) (*glhf.Texture, util.BitmapFontIndex) {
    filePath := path.Join(a.paths[AssetTypeBitmapFonts], fontName)
    fontTextureAtlas := mustLoadTexture(filePath + ".png")
    fontTextureAtlas.SetAtlasItemSize(glyphWidth, glyphHeight)
    atlasIndex := util.NewBitmapFontIndexFromFile(filePath + ".idx")
    return fontTextureAtlas, atlasIndex
}

func (a *Assets) LoadMesh(name string) *util.CompoundMesh {
    filePath := path.Join(a.paths[AssetTypeMeshes], name+".glb")
    return util.LoadGLTFWithTextures(filePath)
}

func (a *Assets) LoadMeshWithoutTextures(name string) *util.CompoundMesh {
    filePath := path.Join(a.paths[AssetTypeMeshes], name+".glb")
    return util.LoadGLTF(filePath, nil)
}
func (a *Assets) LoadMeshWithColor(name string, forcedColor mgl32.Vec3) *util.CompoundMesh {
    filePath := path.Join(a.paths[AssetTypeMeshes], name+".glb")
    return util.LoadGLTF(filePath, &forcedColor)
}

func (a *Assets) LoadMap(filename string) []byte {
    filePath := path.Join(a.paths[AssetTypeMaps], filename+".bin")
    file, err := os.Open(filePath)
    if err != nil {
        panic(err)
    }
    defer file.Close()
    data, err := io.ReadAll(file)
    if err != nil {
        panic(err)
    }
    return data
}

func (a *Assets) LoadMapMetadata(filename string) *MapMetadata {
    filePath := path.Join(a.paths[AssetTypeMaps], filename+".bin.meta")
    return NewMapMetadataFromFile(filePath)
}
func (a *Assets) LoadSkin(file string) *glhf.Texture {
    filePath := path.Join(a.paths[AssetTypeSkins], file+".png")
    return mustLoadTexture(filePath)
}

func mustLoadTexture(filePath string) *glhf.Texture {
    file, err := os.Open(filePath)
    if err != nil {
        panic(err)
    }
    defer file.Close()
    texture, err := util.NewTextureFromReader(file, false)
    if err != nil {
        panic(err)
    }
    return texture
}
