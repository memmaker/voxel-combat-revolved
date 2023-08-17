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

func (a *BattleClient) LoadVoxelMap(filename string) *voxel.Map {
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

	//bf := voxel.NewBlockFactory(textureIndices)
	sizeHorizontal := 3
	sizeVertical := 1
	loadedMap = voxel.NewMap(int32(sizeHorizontal), int32(sizeVertical), int32(sizeHorizontal))
	loadedMap.SetShader(a.chunkShader)
	loadedMap.SetTerrainTexture(terrainTextureAtlas)
	loadedMap.SetTextureIndexCallback(bl.GetTextureIndexForFaces, bl.GetTextureIndexByName("selection"))
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
	listOfBlocks := []string{
		"debug2",
		"selection",
		"bricks",
		"clay",
		"copper_block",
		"weathered_copper",
		"weathered_cut_copper",
		"diamond_block",
		"emerald_block",
		"granite",
		"gravel",
		"iron_block",
		"sandstone",
		"ancient_debris",
		"barrel",
		"bedrock",
		"birch_planks",
		"black_terracotta",
		"yellow_terracotta",
		"white_glazed_terracotta",
		"black_wool",
		"brown_terracotta",
		"pink_terracotta",
		"chiseled_quartz_block",
		"cracked_nether_bricks",
		"crafting_table",
		"deepslate_tiles",
		"dispenser",
		"dried_kelp",
		"fletching_table",
		"exposed_copper",
		"furnace",
		"piston",
		"red_nether_bricks",
		"smithing_table",
		"stripped_oak_log",
		"stripped_spruce_log",
		"observer",
		"target",
		"tnt",
	}
	var loadedMap *voxel.Map
	terrainTexture, indexMap := util.CreateBlockAtlasFromDirectory("assets/textures/blocks/star_odyssey", listOfBlocks)
	bl := game.NewBlockLibrary(listOfBlocks, indexMap)

	//bf := voxel.NewBlockFactory(textureIndices)
	sizeHorizontal := 3
	sizeVertical := 1
	loadedMap = voxel.NewMap(int32(sizeHorizontal), int32(sizeVertical), int32(sizeHorizontal))
	loadedMap.SetShader(a.chunkShader)
	loadedMap.SetTerrainTexture(terrainTexture)
	loadedMap.SetTextureIndexCallback(bl.GetTextureIndexForFaces, bl.GetTextureIndexByName("selection"))
	loadedMap.LoadFromDisk(filename)
	loadedMap.GenerateAllMeshes()
	a.SetVoxelMap(loadedMap)
	a.SetBlockLibrary(bl)
}
