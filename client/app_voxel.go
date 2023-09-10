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
		if voxelMap.Contains(x, y, z) && voxelMap.CurrentlyDraws(x, y, z) {
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

func (a *BattleClient) PlaceBlockAndRemesh(pos voxel.Int3, block *voxel.Block) {
	if a.PlaceBlock(pos, block) {
		voxelMap := a.GetVoxelMap()
		voxelMap.GenerateAllMeshes()
	}
}

func (a *BattleClient) PlaceBlock(pos voxel.Int3, block *voxel.Block) bool {
	voxelMap := a.GetVoxelMap()
	if voxelMap.Contains(int32(pos.X), int32(pos.Y), int32(pos.Z)) {
		voxelMap.SetBlock(int32(pos.X), int32(pos.Y), int32(pos.Z), block)
		return true
	}
	return false
}

func (a *BattleClient) RemoveBlockAndRemesh() {
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

	loadedMap := voxel.NewMapFromConstruction(bf, a.chunkShader, construction, voxel.Int3{16, 16, 16})
	loadedMap.SetTerrainTexture(terrainTexture)

	loadedMap.GenerateAllMeshes()
	a.SetVoxelMap(loadedMap)
	return loadedMap
}

func (a *BattleClient) LoadEmptyWorld(mapSize voxel.Int3, chunkSizeHorizontal, chunkSizeHeight int32) *voxel.Map {
	listOfBlocks := game.GetDebugBlockNames()
	var loadedMap *voxel.Map

	terrainTexture, indexMap := a.GetAssets().LoadBlockTextureAtlas("star_odyssey_01")
	bl := game.NewBlockLibrary(listOfBlocks, indexMap)
	bl.ApplyGameplayRules(a.GameInstance)
	//bf := voxel.NewBlockFactory(textureIndices)
	loadedMap = voxel.NewMap(int32(mapSize.X), int32(mapSize.Y), int32(mapSize.Z), chunkSizeHorizontal, chunkSizeHeight)
	loadedMap.SetLogger(util.LogVoxelInfo, util.LogGameError)
	loadedMap.SetShader(a.chunkShader)
	loadedMap.SetTerrainTexture(terrainTexture)
	loadedMap.SetTextureIndexCallback(bl.GetTextureIndexForFaces)
	for x := int32(0); x < mapSize.X; x++ {
		for y := int32(0); y < mapSize.Y; y++ {
			for z := int32(0); z < mapSize.Z; z++ {
				loadedMap.NewChunk(int32(x), y, int32(z))
			}
		}
	}
	loadedMap.SetFloorAtHeight(0, bl.NewBlockFromName("bricks"))
	loadedMap.GenerateAllMeshes()
	a.SetVoxelMap(loadedMap)
	return loadedMap
}
func (a *BattleClient) LoadMap(filename string) {
	listOfBlocks := game.GetDebugBlockNames()
	var loadedMap *voxel.Map

    terrainTexture, indexMap := a.GetAssets().LoadBlockTextureAtlas("star_odyssey_01")
	// Create atlas and index from directory
	//	terrainTexture, indexMap := util.CreateBlockAtlasFromDirectory("./assets/textures/blocks/star_odyssey", listOfBlocks)
	//	terrainTexture.SaveAsPNG("./assets/textures/blocks/star_odyssey_01.png")
	//	indexMap.WriteAtlasIndex("./assets/textures/blocks/star_odyssey_01.idx")

	// NOTE: The order of the list of blocks does matter!
	bl := game.NewBlockLibrary(listOfBlocks, indexMap)
	bl.ApplyGameplayRules(a.GameInstance)
	loadedMap = a.GetVoxelMap()
	loadedMap.SetTerrainTexture(terrainTexture)
	loadedMap.SetTextureIndexCallback(bl.GetTextureIndexForFaces)
	loadedMap.SetShader(a.chunkShader)
	loadedMap.GenerateAllMeshes()

	a.SetVoxelMap(loadedMap)
	a.SetBlockLibrary(bl)
}
