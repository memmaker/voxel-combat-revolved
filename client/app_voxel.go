package client

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
	"github.com/memmaker/battleground/game"
)

func (a *BattleClient) RayCast(rayStart, rayEnd mgl32.Vec3) *game.RayCastHit {
	voxelMap := a.GetVoxelMap()
	var visitedBlocks []voxel.Int3
	var unitHit voxel.MapObject
	stopRay := func(x, y, z int32) bool {
		visitedBlocks = append(visitedBlocks, voxel.Int3{X: x, Y: y, Z: z})
		if voxelMap.Contains(x, y, z) {
			block := voxelMap.GetGlobalBlock(x, y, z)
			if block != nil && !block.IsAir() {
				return true
			} else if block.IsOccupied() {
				unitHit = block.GetOccupant().(*Unit)
				return true
			}
		}
		return false
	}
	hitInfo := util.DDARaycast(rayStart, rayEnd, stopRay)
	insideMap := voxelMap.ContainsGrid(hitInfo.CollisionGridPosition) || voxelMap.ContainsGrid(hitInfo.PreviousGridPosition)
	a.lastHitInfo = &game.RayCastHit{HitInfo3D: hitInfo, VisitedBlocks: visitedBlocks, UnitHit: unitHit, InsideMap: insideMap}
	return a.lastHitInfo
}

func (a *BattleClient) RayCastGround(rayStart, rayEnd mgl32.Vec3) *game.RayCastHit {
	voxelMap := a.GetVoxelMap()
	var visitedBlocks []voxel.Int3
	var unitHit voxel.MapObject
	stopRay := func(x, y, z int32) bool {
		visitedBlocks = append(visitedBlocks, voxel.Int3{X: x, Y: y, Z: z})
		if voxelMap.Contains(x, y, z) {
			block := voxelMap.GetGlobalBlock(x, y, z)
			if block != nil && !block.IsAir() {
				return true
			} else if block.IsOccupied() {
				unitHit = block.GetOccupant()
			}
		}
		return false
	}
	hitInfo := util.DDARaycast(rayStart, rayEnd, stopRay)
	insideMap := voxelMap.ContainsGrid(hitInfo.CollisionGridPosition) || voxelMap.ContainsGrid(hitInfo.PreviousGridPosition)
	a.lastHitInfo = &game.RayCastHit{HitInfo3D: hitInfo, VisitedBlocks: visitedBlocks, UnitHit: unitHit, InsideMap: insideMap}
	return a.lastHitInfo
}

func (a *BattleClient) PlaceBlock(pos voxel.Int3, block *voxel.Block) {
	voxelMap := a.GetVoxelMap()
	if voxelMap.Contains(int32(pos.X), int32(pos.Y), int32(pos.Z)) {
		voxelMap.SetBlock(int32(pos.X), int32(pos.Y), int32(pos.Z), block)
		voxelMap.GenerateAllMeshes()
	}
}

func (a *BattleClient) RemoveBlock() {
	voxelMap := a.GetVoxelMap()
	if a.lastHitInfo == nil {
		return
	}
	collisionGridPosition := a.lastHitInfo.CollisionGridPosition

	if voxelMap.Contains(int32(collisionGridPosition.X), int32(collisionGridPosition.Y), int32(collisionGridPosition.Z)) {
		voxelMap.SetBlock(int32(collisionGridPosition.X), int32(collisionGridPosition.Y), int32(collisionGridPosition.Z), voxel.NewAirBlock())
		voxelMap.GenerateAllMeshes()
	}
}

func (a *BattleClient) LoadConstructionFile(filename string) *voxel.Map {
	construction := voxel.LoadConstruction(filename)
	listOfBlocks := voxel.GetBlocksNeededByConstruction(construction)
	listOfBlockEntities := voxel.GetBlockEntitiesNeededByConstruction(construction)
	listOfBlocks = append(listOfBlocks, listOfBlockEntities...)

	terrainTexture, textureIndices := util.CreateBlockAtlasFromDirectory("assets/textures/blocks/minecraft", listOfBlocks)
	bf := voxel.NewBlockFactory(textureIndices)

	loadedMap := voxel.NewMapFromConstruction(bf, a.chunkShader, construction)
	loadedMap.SetTerrainTexture(terrainTexture)

	loadedMap.GenerateAllMeshes()
	a.SetVoxelMap(loadedMap)
	return loadedMap
}

func (a *BattleClient) LoadEmptyWorld() *voxel.Map {
	listOfBlocks := []string{
		"selection",
		"brick",
		"clay",
		"copper_block",
		"diamond_block",
		"emerald_block",
		"granite",
		"gravel",
		"iron_block",
		"sand",
		"sandstone",
		"stone",
	}
	var loadedMap *voxel.Map
	terrainTextureAtlas, indexMap := util.CreateBlockAtlasFromDirectory("assets/textures/blocks/minecraft", listOfBlocks)
	bl := game.NewBlockLibrary(listOfBlocks, indexMap)
	bl.ApplyGameplayRules(a.GameInstance)
	//bf := voxel.NewBlockFactory(textureIndices)
	sizeHorizontal := 3
	sizeVertical := 1
	loadedMap = voxel.NewMap(int32(sizeHorizontal), int32(sizeVertical), int32(sizeHorizontal))
	loadedMap.SetShader(a.chunkShader)
	loadedMap.SetTerrainTexture(terrainTextureAtlas)
	loadedMap.SetTextureIndexCallback(bl.GetTextureIndexForFaces)
	for x := 0; x < sizeHorizontal; x++ {
		for z := 0; z < sizeHorizontal; z++ {
			loadedMap.NewChunk(int32(x), 0, int32(z))
		}
	}
	loadedMap.SetFloorAtHeight(0, bl.NewBlockFromName("stone"))
	loadedMap.GenerateAllMeshes()
	a.SetVoxelMap(loadedMap)
	return loadedMap
}
func (a *BattleClient) LoadMap(filename string) {
	listOfBlocks := game.GetDebugBlockNames()
	var loadedMap *voxel.Map

	terrainTexture := util.MustLoadTexture("./assets/textures/blocks/star_odyssey_01.png")
	terrainTexture.SetAtlasItemSize(16, 16)
	indexMap := util.NewBlockIndexFromFile("./assets/textures/blocks/star_odyssey_01.idx")

	// Create atlas and index from directory
	//	terrainTexture, indexMap := util.CreateBlockAtlasFromDirectory("./assets/textures/blocks/star_odyssey", listOfBlocks)
	//	terrainTexture.SaveAsPNG("./assets/textures/blocks/star_odyssey_01.png")
	//	indexMap.WriteAtlasIndex("./assets/textures/blocks/star_odyssey_01.idx")

	// NOTE: The order of the list of blocks does matter!
	bl := game.NewBlockLibrary(listOfBlocks, indexMap)
	bl.ApplyGameplayRules(a.GameInstance)

	loadedMap = voxel.NewMapFromFile(filename, a.chunkShader, terrainTexture)
	loadedMap.SetTextureIndexCallback(bl.GetTextureIndexForFaces)
	loadedMap.GenerateAllMeshes()

	a.SetVoxelMap(loadedMap)
	a.SetBlockLibrary(bl)
}
