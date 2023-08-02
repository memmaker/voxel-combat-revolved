package client

import (
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
)

func (a *BattleGame) createTestMap(sizeX, sizeY, sizeZ int32) {
	testMap := a.voxelMap
	for x := int32(0); x < sizeX; x++ {
		for y := int32(0); y < sizeY; y++ {
			for z := int32(0); z < sizeZ; z++ {
				chunk := testMap.NewChunk(x, y, z)
				createTestBlocks(chunk)
			}
		}
	}
	testMap.GenerateAllMeshes()
}

func createTestBlocks(testChunk *voxel.Chunk) {
	testBlock := voxel.NewTestBlock(1)
	// fill the lowest layer with blocks (y=0)
	for x := int32(0); x < voxel.CHUNK_SIZE; x++ {
		for z := int32(0); z < voxel.CHUNK_SIZE; z++ {
			testChunk.SetBlock(x, 0, z, testBlock)
		}
	}
}

func (a *BattleGame) debugFunc() {
	//chunk := a.voxelMap.GetChunk(0, 0, 0)
	util.CheckForGLError()
	a.Print("Hello World")
}

func (a *BattleGame) updateDebugInfo() {
	if !a.showDebugInfo {
		return
	}
	//camPos := a.isoCamera.GetPosition()
	/*
	       posString := fmt.Sprintf("Pos: %.2f, %.2f, %.2f", camPos.X(), camPos.Y(), camPos.Z())
	   	dirString := fmt.Sprintf("Dir: %.2f, %.2f, %.2f", a.isoCamera.GetFront().X(), a.isoCamera.GetFront().Y(), a.isoCamera.GetFront().Z())
	   	chunk := a.voxelMap.GetChunkFromPosition(camPos)
	   	chunkString := "Chunk: none"
	   	if chunk != nil {
	   		chunkPos := chunk.Position()
	   		chunkString = fmt.Sprintf("Chunk: %d, %d, %d", chunkPos.X, chunkPos.Y, chunkPos.Z)
	   	}

	*/

	selectedBlockString := "Block: none"
	if a.lastHitInfo != nil {
		selectedBlockString = fmt.Sprintf("Col.Block: %d, %d, %d", a.lastHitInfo.CollisionGridPosition.X, a.lastHitInfo.CollisionGridPosition.Y, a.lastHitInfo.CollisionGridPosition.Z)
		selectedBlockString += fmt.Sprintf("\nHitSide: %d", a.lastHitInfo.Side)
		selectedBlockString += fmt.Sprintf("\nPrev.Block: %d, %d, %d", a.lastHitInfo.PreviousGridPosition.X, a.lastHitInfo.PreviousGridPosition.Y, a.lastHitInfo.PreviousGridPosition.Z)
		selectedBlockString += fmt.Sprintf("\nHit WorldPos: %.2f, %.2f, %.2f", a.lastHitInfo.CollisionWorldPosition.X(), a.lastHitInfo.CollisionWorldPosition.Y(), a.lastHitInfo.CollisionWorldPosition.Z())
	}
	unitPos := a.allUnits[0].GetPosition()
	unitPosString := fmt.Sprintf("Unit: %.2f, %.2f, %.2f", unitPos.X(), unitPos.Y(), unitPos.Z())

	animString := a.allUnits[0].model.GetAnimationDebugString()

	//timerString := a.timer.String()
	//debugInfo := fmt.Sprintf("%s\n%s\n%s\n%s\n%s", posString, dirString, chunkString, selectedBlockString, timerString)
	debugInfo := fmt.Sprintf("\n%s\n%s", unitPosString, animString)
	a.Print(debugInfo)
}

func (a *BattleGame) placeDebugLine(startEnd [2]mgl32.Vec3) {
	a.debugObjects = a.debugObjects[:0]
	//camDirection := a.player.cam.GetFront()
	var lines [][2]mgl32.Vec3
	lines = [][2]mgl32.Vec3{
		startEnd,
	}
	rayLine := util.NewLineMesh(a.lineShader, lines)
	a.debugObjects = append(a.debugObjects, rayLine)
}
