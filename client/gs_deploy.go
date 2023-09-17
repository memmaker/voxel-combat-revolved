package client

import (
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
	"github.com/memmaker/battleground/game"
	"math"
)

func NewGameStateDeployment(a *BattleClient) *GameStateDeployment {
	g := &GameStateDeployment{IsoMovementState: IsoMovementState{engine: a}}
	return g
}

type GameStateDeployment struct {
	IsoMovementState
	currentIndex   int
	validPositions map[voxel.Int3]bool
}

func (g *GameStateDeployment) OnMouseMoved(oldX, oldY, newX, newY float64) {
	g.IsoMovementState.OnMouseMoved(oldX, oldY, newX, newY)
	currentUnit := g.currentUnit()
	targetPos := g.engine.groundSelector.GetBlockPosition()

	if g.isPlacementAllowed(targetPos, currentUnit) {
		currentUnit.SetBlockPositionAndUpdateStance(targetPos)
	}
}

func (g *GameStateDeployment) isPlacementAllowed(targetPos voxel.Int3, currentUnit *Unit) bool {
	_, isSpawnPos := g.validPositions[targetPos]
	vMap := g.engine.GetVoxelMap()
	placeable, _ := vMap.IsUnitPlaceable(currentUnit, targetPos)

	return placeable && isSpawnPos
}

func (g *GameStateDeployment) OnMouseReleased(x float64, y float64) {

}

func (g *GameStateDeployment) OnServerMessage(msgType string, json string) {

}

func (g *GameStateDeployment) OnMouseClicked(x float64, y float64) {

}

func (g *GameStateDeployment) OnKeyPressed(key glfw.Key) {
	if key == glfw.Key1 {
		g.currentIndex -= 1
		if g.currentIndex < 0 {
			g.currentIndex = 0
		}
	} else if key == glfw.Key2 {
		g.currentIndex += 1
		if g.currentIndex >= len(g.engine.GetDeploymentQueue()) {
			g.currentIndex = len(g.engine.GetDeploymentQueue()) - 1
		}
	} else if key == glfw.KeyEnter {
		// send deployment to server
		g.tryToSubmitDeployment()
	} else {
		g.IsoMovementState.OnKeyPressed(key)
	}

}
func (g *GameStateDeployment) currentUnit() *Unit {
	return g.engine.GetDeploymentQueue()[g.currentIndex]
}
func (g *GameStateDeployment) Init(wasPopped bool) {
	g.engine.highlights.ClearAll()
	g.engine.SwitchToGroundSelector()
	g.engine.actionbar.Hide()
	center, validPositions := g.setSpawnHighlights(g.engine.GetMapMetadata(), g.engine.GetSpawnIndex())
	g.validPositions = validPositions

	startCam := g.engine.isoCamera.GetTransform()
	g.engine.isoCamera.SetLookTarget(center.ToBlockCenterVec3())
	endCam := g.engine.isoCamera
	g.engine.StartCameraLookAnimation(startCam, endCam, 0.5)

	g.engine.FlashText("DEPLOY!", 3)
}

func (g *GameStateDeployment) setSpawnHighlights(mapMeta *game.MapMetadata, playerIndex uint64) (voxel.Int3, map[voxel.Int3]bool) {
	minX := int32(math.MaxInt32)
	maxX := int32(math.MinInt32)
	minZ := int32(math.MaxInt32)
	maxZ := int32(math.MinInt32)

	spawnMap := make(map[voxel.Int3]bool)
	g.engine.highlights.ClearFlat(voxel.HighlightEditor)
	if len(mapMeta.SpawnPositions) == 0 || len(mapMeta.SpawnPositions) <= int(playerIndex) {
		println("ERROR - No spawn positions found for player", playerIndex)
		return voxel.Int3{}, spawnMap
	}
	if len(mapMeta.SpawnPositions[playerIndex]) > 0 {
		g.engine.highlights.AddFlat(voxel.HighlightEditor, mapMeta.SpawnPositions[playerIndex], mgl32.Vec3{0.1, 0.8, 0.1})
		for _, pos := range mapMeta.SpawnPositions[playerIndex] {
			if pos.X < minX {
				minX = pos.X
			} else if pos.X > maxX {
				maxX = pos.X
			}
			if pos.Z < minZ {
				minZ = pos.Z
			} else if pos.Z > maxZ {
				maxZ = pos.Z
			}
			spawnMap[pos] = true
		}
	}
	g.engine.highlights.ShowAsFlat(voxel.HighlightEditor)
	center := voxel.Int3{X: (minX + maxX) / 2, Y: 1, Z: (minZ + maxZ) / 2}
	return center, spawnMap
}

func (g *GameStateDeployment) tryToSubmitDeployment() {
	invalidPlacement := false
	placement := make(map[uint64]voxel.Int3)
	for _, unit := range g.engine.GetDeploymentQueue() {
		placedPos := unit.GetBlockPosition()
		if !g.isPlacementAllowed(placedPos, unit) {
			invalidPlacement = true
			break
		}

		placement[unit.UnitID()] = placedPos
	}
	if invalidPlacement {
		g.engine.FlashText("INVALID!", 2)
		return
	}

	util.MustSend(g.engine.server.SelectDeployment(placement))
	g.engine.PopState() // switch to wait
}
