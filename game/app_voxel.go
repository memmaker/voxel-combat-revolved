package game

import (
	"github.com/faiface/mainthread"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
)

type RayCastHit struct {
	util.HitInfo3D
	VisitedBlocks []voxel.Int3
	UnitHit       *Unit
}

func (a *BattleGame) RayCastUnits(rayStart, rayEnd mgl32.Vec3, sourceUnit, targetUnit voxel.MapObject) *RayCastHit {
	voxelMap := a.voxelMap
	var visitedBlocks []voxel.Int3
	var unitHit *Unit
	stopRay := func(x, y, z int32) bool {
		visitedBlocks = append(visitedBlocks, voxel.Int3{X: x, Y: y, Z: z})
		if voxelMap.Contains(x, y, z) {
			block := voxelMap.GetGlobalBlock(x, y, z)
			if block != nil && !block.IsAir() {
				return true
			} else if block.IsOccupied() && (block.GetOccupant() != sourceUnit && block.GetOccupant() == targetUnit) {
				unitHit = block.GetOccupant().(*Unit)
				return true
			}
		}
		return false
	}
	hitInfo := util.DDARaycast(rayStart, rayEnd, stopRay)
	if hitInfo.Hit && (voxelMap.ContainsGrid(hitInfo.CollisionGridPosition) || voxelMap.ContainsGrid(hitInfo.PreviousGridPosition)) {
		a.lastHitInfo = &RayCastHit{HitInfo3D: hitInfo, VisitedBlocks: visitedBlocks, UnitHit: unitHit}
	} else {
		a.lastHitInfo = nil
	}
	return a.lastHitInfo
}
func (a *BattleGame) RayCast(rayStart, rayEnd mgl32.Vec3) *RayCastHit {
	voxelMap := a.voxelMap
	var visitedBlocks []voxel.Int3
	var unitHit *Unit
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
	if hitInfo.Hit && (voxelMap.ContainsGrid(hitInfo.CollisionGridPosition) || voxelMap.ContainsGrid(hitInfo.PreviousGridPosition)) {
		a.lastHitInfo = &RayCastHit{HitInfo3D: hitInfo, VisitedBlocks: visitedBlocks, UnitHit: unitHit}
	} else {
		a.lastHitInfo = nil
	}
	return a.lastHitInfo
}

func (a *BattleGame) PlaceBlockAtCurrentSelection() {
	if a.lastHitInfo == nil {
		return
	}
	previousGridPosition := a.lastHitInfo.PreviousGridPosition

	a.PlaceBlock(previousGridPosition, voxel.NewTestBlock(a.blockTypeToPlace))
}

func (a *BattleGame) PlaceBlock(pos voxel.Int3, block *voxel.Block) {
	voxelMap := a.voxelMap
	if voxelMap.Contains(int32(pos.X), int32(pos.Y), int32(pos.Z)) {
		voxelMap.SetBlock(int32(pos.X), int32(pos.Y), int32(pos.Z), block)
		voxelMap.GenerateAllMeshes()
	}
}

func (a *BattleGame) RemoveBlock() {
	voxelMap := a.voxelMap
	if a.lastHitInfo == nil {
		return
	}
	collisionGridPosition := a.lastHitInfo.CollisionGridPosition

	if voxelMap.Contains(int32(collisionGridPosition.X), int32(collisionGridPosition.Y), int32(collisionGridPosition.Z)) {
		voxelMap.SetBlock(int32(collisionGridPosition.X), int32(collisionGridPosition.Y), int32(collisionGridPosition.Z), voxel.NewAirBlock())
		voxelMap.GenerateAllMeshes()
	}
}

func (a *BattleGame) LoadVoxelMap(filename string) *voxel.Map {
	construction := voxel.LoadConstruction(filename)
	listOfBlocks := voxel.GetBlocksNeededByConstruction(construction)
	listOfBlockEntities := voxel.GetBlockEntitiesNeededByConstruction(construction)
	listOfBlocks = append(listOfBlocks, listOfBlockEntities...)

	terrainTexture, textureIndices := util.CreateAtlasFromDirectory("assets/textures/blocks/minecraft", listOfBlocks)
	bf := voxel.NewBlockFactory(textureIndices)

	loadedMap := voxel.NewMapFromConstruction(bf, a.chunkShader, a.highlightShader, construction)
	loadedMap.SetTerrainTexture(terrainTexture)

	loadedMap.GenerateAllMeshes()
	a.SetVoxelMap(loadedMap)
	return loadedMap
}

func (a *BattleGame) LoadEmptyWorld() *voxel.Map {
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
	mainthread.Call(func() {
		terrainTexture, textureIndices := util.CreateAtlasFromDirectory("assets/textures/blocks/minecraft", listOfBlocks)

		bf := voxel.NewBlockFactory(textureIndices)
		sizeHorizontal := 3
		sizeVertical := 1
		loadedMap = voxel.NewMap(int32(sizeHorizontal), int32(sizeVertical), int32(sizeHorizontal))
		loadedMap.SetShader(a.chunkShader, a.highlightShader)
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
func (a *BattleGame) LoadMap(filename string) *voxel.Map {
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
	mainthread.Call(func() {
		terrainTexture, _ := util.CreateAtlasFromDirectory("assets/textures/blocks/minecraft", listOfBlocks)
		//bf := voxel.NewBlockFactory(textureIndices)
		sizeHorizontal := 3
		sizeVertical := 1
		loadedMap = voxel.NewMap(int32(sizeHorizontal), int32(sizeVertical), int32(sizeHorizontal))
		loadedMap.SetShader(a.chunkShader, a.highlightShader)
		loadedMap.SetTerrainTexture(terrainTexture)
		loadedMap.LoadFromDisk(filename)
		a.SetVoxelMap(loadedMap)
	})

	return loadedMap
}

func (a *BattleGame) SetVoxelMap(testMap *voxel.Map) {
	a.voxelMap = testMap
	a.voxelMap.SetUnitMovedHandler(a.OnUnitMoved)
}
