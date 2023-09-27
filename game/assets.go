package game

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/glhf"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
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
func (a *Assets) LoadMapWithDetails(mapFile string, details *MissionDetails) *DefaultMapInfo {
	mapMetadata := a.LoadMapMetadata(mapFile)
	details.SyncFromMap(mapMetadata)
	library := a.LoadBlockLibrary(mapMetadata.Blocks)
	return &DefaultMapInfo{
		mapFile:      mapFile,
		details:      details,
		metaData:     &mapMetadata,
		blockLibrary: library,
		loadedMap:    voxel.NewMapFromSource(a.LoadMap(mapFile), nil, nil),
	}
}

func (a *Assets) LoadBiomeWithDetails(biome Biome, details *MissionDetails) MapInfo {
	biomeName := biome.GetName()
	atlas, library := a.LoadBlockTextureAtlas(biomeName)

	biomeMap, mapMetadata := biome.Generate()
	biomeMap.SetTerrainTexture(atlas)

	details.SyncFromMap(mapMetadata)

	return &DefaultMapInfo{
		mapFile:      biomeName,
		details:      details,
		metaData:     &mapMetadata,
		blockLibrary: library,
		loadedMap:    biomeMap,
	}
}
func (a *Assets) LoadBlockTextureAtlas(filename string) (*glhf.Texture, *BlockLibrary) {
	filePath := path.Join(a.paths[AssetTypeBlockTextures], filename)
	texture := mustLoadTexture(filePath + ".png")
	texture.SetAtlasItemSize(16, 16)
	indexMap := util.NewBlockIndexFromFile(filePath + ".idx")
	blockList := util.NewBlockListFromFile(filePath + ".txt")
	bl := NewBlockLibrary(blockList, indexMap)
	return texture, bl
}

func (a *Assets) LoadBlockLibrary(filename string) *BlockLibrary {
	filePath := path.Join(a.paths[AssetTypeBlockTextures], filename)
	indexMap := util.NewBlockIndexFromFile(filePath + ".idx")
	blockList := util.NewBlockListFromFile(filePath + ".txt")
	bl := NewBlockLibrary(blockList, indexMap)
	return bl
}

func (a *Assets) LoadBitmapFont(fontName string, glyphWidth int, glyphHeight int) (*glhf.Texture, util.BitmapFontIndex) {
	filePath := path.Join(a.paths[AssetTypeBitmapFonts], fontName)
	fontTextureAtlas := mustLoadTexture(filePath + ".png")
	fontTextureAtlas.SetAtlasItemSize(glyphWidth, glyphHeight)
	atlasIndex := util.NewBitmapFontIndexFromFile(filePath + ".idx")
	return fontTextureAtlas, atlasIndex
}

func (a *Assets) LoadBitmapFontWithoutIndex(fontName string, glyphWidth int, glyphHeight int) *glhf.Texture {
	filePath := path.Join(a.paths[AssetTypeBitmapFonts], fontName)
	fontTextureAtlas := mustLoadTexture(filePath + ".png")
	fontTextureAtlas.SetAtlasItemSize(glyphWidth, glyphHeight)
	return fontTextureAtlas
}

func (a *Assets) LoadMesh(name string) *util.CompoundMesh {
	filePath := a.getModelFile(name)
	return util.LoadGLTFWithTextures(filePath)
}

func (a *Assets) LoadAnimatedMeshWithTextures(name string, animationMap map[string]string) *util.CompoundMesh {
	filePath := a.getModelFile(name)
	return util.LoadGLTFWithAnimationAndTextures(filePath, animationMap)
}
func (a *Assets) LoadMeshWithAnimationMap(name string, animationMap map[string]string) *util.CompoundMesh {
	filePath := a.getModelFile(name)
	return util.LoadGLTF(filePath, animationMap, nil)
}
func (a *Assets) LoadMeshWithoutTextures(name string) *util.CompoundMesh {
	filePath := a.getModelFile(name)
	return util.LoadGLTF(filePath, nil, nil)
}
func (a *Assets) LoadMeshWithColor(name string, forcedColor mgl32.Vec3) *util.CompoundMesh {
	filePath := a.getModelFile(name)
	return util.LoadGLTF(filePath, nil, &forcedColor)
}

func (a *Assets) getModelFile(name string) string {
	binPath := path.Join(a.paths[AssetTypeMeshes], name+".glb")
	if util.DoesFileExist(binPath) {
		return binPath
	}
	return path.Join(a.paths[AssetTypeMeshes], name+".gltf")
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

func (a *Assets) LoadMapMetadata(filename string) MapMetadata {
	filePath := path.Join(a.paths[AssetTypeMaps], filename+".bin.meta")
	return NewMapMetadataFromFile(filePath)
}
func (a *Assets) LoadSkin(file string) *glhf.Texture {
	filePath := path.Join(a.paths[AssetTypeSkins], file+".png")
	return mustLoadTexture(filePath)
}

func (a *Assets) GetMapPath(mapName string) string {
	return path.Join(a.paths[AssetTypeMaps], mapName+".bin")
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
