package game

import (
	"github.com/faiface/mainthread"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
)

func (a *BattleGame) updateSelectedBlock(rayStart, rayEnd mgl32.Vec3) {
	voxelMap := a.voxelMap
	var visitedBlocks []util.IntVec3
	stopRay := func(x, y, z int) bool {
		visitedBlocks = append(visitedBlocks, util.IntVec3{x, y, z})
		if voxelMap.Contains(int32(x), int32(y), int32(z)) {
			block := voxelMap.GetGlobalBlock(int32(x), int32(y), int32(z))
			if block != nil && !block.IsAir() {
				return true
			} else {
				// TODO: check if objects or actors are in the way
			}
		}
		return false
	}
	hitInfo := util.DDARaycast(rayStart, rayEnd, stopRay)
	if hitInfo.Hit && (voxelMap.ContainsGrid(hitInfo.CollisionGridPosition) || voxelMap.ContainsGrid(hitInfo.PreviousGridPosition)) {
		a.blockSelector.SetPosition(hitInfo.PreviousGridPosition.ToVec3())
		a.lastHitInfo = &hitInfo
		a.lastVisitedBlocks = visitedBlocks
	} else {
		a.lastHitInfo = nil
		a.lastVisitedBlocks = nil
	}
}

func (a *BattleGame) PlaceBlockAtCurrentSelection() {
	if a.lastHitInfo == nil {
		return
	}
	previousGridPosition := a.lastHitInfo.PreviousGridPosition

	a.PlaceBlock(previousGridPosition, voxel.NewTestBlock(a.blockTypeToPlace))
}

func (a *BattleGame) PlaceBlock(pos util.IntVec3, block *voxel.Block) {
	voxelMap := a.voxelMap
	if voxelMap.Contains(int32(pos.X()), int32(pos.Y()), int32(pos.Z())) {
		voxelMap.SetBlock(int32(pos.X()), int32(pos.Y()), int32(pos.Z()), block)
		voxelMap.GenerateAllMeshes()
	}
}

func (a *BattleGame) RemoveBlock() {
	voxelMap := a.voxelMap
	if a.lastHitInfo == nil {
		return
	}
	collisionGridPosition := a.lastHitInfo.CollisionGridPosition

	if voxelMap.Contains(int32(collisionGridPosition.X()), int32(collisionGridPosition.Y()), int32(collisionGridPosition.Z())) {
		voxelMap.SetBlock(int32(collisionGridPosition.X()), int32(collisionGridPosition.Y()), int32(collisionGridPosition.Z()), voxel.NewAirBlock())
		voxelMap.GenerateAllMeshes()
	}
}

func (a *BattleGame) LoadVoxelMap(filename string) *voxel.Map {
	construction := util.LoadConstruction(filename)
	listOfBlocks := voxel.GetBlocksNeededByConstruction(construction)
	listOfBlockEntities := voxel.GetBlockEntitiesNeededByConstruction(construction)
	listOfBlocks = append(listOfBlocks, listOfBlockEntities...)

	terrainTexture, textureIndices := util.CreateAtlasFromDirectory("assets/textures/blocks/minecraft", listOfBlocks)
	bf := voxel.NewBlockFactory(textureIndices)

	loadedMap := voxel.NewMapFromConstruction(bf, a.chunkShader, construction)
	loadedMap.SetTerrainTexture(terrainTexture)

	loadedMap.GenerateAllMeshes()
	a.SetVoxelMap(loadedMap)
	return loadedMap
}

func (a *BattleGame) LoadEmptyWorld() *voxel.Map {
	listOfBlocks := []string{
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
	mainthread.Call(func() {
		terrainTexture, textureIndices := util.CreateAtlasFromDirectory("assets/textures/blocks/minecraft", listOfBlocks)

		bf := voxel.NewBlockFactory(textureIndices)
		sizeHorizontal := 3
		sizeVertical := 1
		loadedMap = voxel.NewMap(int32(sizeHorizontal), int32(sizeVertical), int32(sizeHorizontal))
		loadedMap.SetChunkShader(a.chunkShader)
		loadedMap.SetTerrainTexture(terrainTexture)
		for x := 0; x < sizeHorizontal; x++ {
			for z := 0; z < sizeHorizontal; z++ {
				loadedMap.NewChunk(int32(x), 0, int32(z))
			}
		}
		loadedMap.SetFloorAtHeight(0, bf.GetBlockByName("stone"))
		//loadedMap.SetSetRandomStuff(bf.GetBlockByName("stone"))
		loadedMap.GenerateAllMeshes()
		a.SetVoxelMap(loadedMap)
	})

	return loadedMap
}

func (a *BattleGame) SetVoxelMap(testMap *voxel.Map) {
	a.voxelMap = testMap
}

func (a *BattleGame) SetBlockSelector(selector PositionDrawable) {
	a.blockSelector = selector
}
