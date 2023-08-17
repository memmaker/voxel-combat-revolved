package client

import (
	"fmt"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/memmaker/battleground/engine/gui"
	"github.com/memmaker/battleground/engine/voxel"
	"math"
)

type GameStateEditMap struct {
	IsoMovementState
	blockTypeToPlace byte
	pencil           BlockPlacer
	blockPage        int
}

func (g *GameStateEditMap) OnServerMessage(msgType string, json string) {

}

func (g *GameStateEditMap) OnKeyPressed(key glfw.Key) {
	if key == glfw.KeyF {
		g.PlaceBlockAtCurrentSelection()
	}

	if key == glfw.KeyR {
		g.engine.RemoveBlock()
	}

	if key == glfw.KeyO {
		g.engine.GetVoxelMap().SaveToDisk()
	}

	if key == glfw.KeyP {
		g.engine.GetVoxelMap().LoadFromDisk("assets/maps/map.bin")
		g.engine.GetVoxelMap().GenerateAllMeshes()
	}

	if key == glfw.Key1 {
		// previous page
		if g.blockPage > 0 {
			g.blockPage--
			g.setBlockPage(g.blockPage)
		}
	}

	if key == glfw.Key2 {
		// next page
		itemsPerPage := 10
		lastPage := g.lastPage(itemsPerPage)
		g.blockPage = g.blockPage + 1
		if g.blockPage > lastPage {
			g.blockPage = lastPage
		}
		g.setBlockPage(g.blockPage)
	}

	if key == glfw.KeyComma {
		fill := !g.pencil.GetFill()
		g.pencil.SetFill(fill)
		g.engine.Print(fmt.Sprintf("Fill: %v", fill))
	}

	if key == glfw.KeyDelete {
		g.ClearMap()
	}
}

func (g *GameStateEditMap) lastPage(itemsPerPage int) int {
	return int(math.Floor(float64(g.engine.blockLibrary.LastBlockID()-1) / float64(itemsPerPage)))
}
func (g *GameStateEditMap) PlaceBlockAtCurrentSelection() {
	if g.engine.lastHitInfo == nil {
		return
	}
	previousGridPosition := g.engine.lastHitInfo.PreviousGridPosition

	g.engine.PlaceBlock(previousGridPosition, voxel.NewBlock(g.blockTypeToPlace))
}

func (g *GameStateEditMap) PlaceRange(selection []voxel.Int3) {
	for _, pos := range selection {
		g.engine.PlaceBlock(pos, voxel.NewBlock(g.blockTypeToPlace))
	}
}

func (g *GameStateEditMap) Init(bool) {
	g.engine.SwitchToBlockSelector()
	g.engine.GetVoxelMap().ClearHighlights()

	g.changeActionBar()
	g.pencil = NewRectanglePlacer()
	println(fmt.Sprintf("[GameStateEditMap] Entered"))
}

func (g *GameStateEditMap) changeActionBar() {
	g.engine.actionbar = gui.NewActionBar(g.engine.guiShader, g.engine.GetVoxelMap().GetTerrainTexture(), g.engine.WindowWidth, g.engine.WindowHeight, 16, 16)
	g.blockPage = 0
	g.setBlockPage(g.blockPage)
}

func (g *GameStateEditMap) setBlockPage(page int) {
	itemsPerPage := 10
	lastPage := g.lastPage(itemsPerPage)
	if page > lastPage {
		page = lastPage
	} else if page < 0 {
		page = 0
	}
	firstItem := (page * itemsPerPage) + 1
	lastItem := firstItem + itemsPerPage
	if lastItem > int(g.engine.blockLibrary.LastBlockID()) {
		lastItem = int(g.engine.blockLibrary.LastBlockID())
	}
	blockLib := g.engine.blockLibrary
	actions := make([]gui.ActionItem, 0)
	for i := firstItem; i < lastItem; i++ {
		index := i
		blockdef := blockLib.GetBlockDefinition(byte(i))
		actions = append(actions, gui.ActionItem{
			Name:         blockdef.UniqueName,
			TextureIndex: blockdef.TextureIndicesForFaces[voxel.South],
			Execute:      func() { g.changeBlockTypeToPlace(byte(index)) },
			Hotkey:       glfw.Key(int(glfw.Key0) + (i - firstItem)),
		})
	}
	g.engine.actionbar.SetActions(actions)
}

func (g *GameStateEditMap) changeBlockTypeToPlace(blockType byte) {
	g.blockTypeToPlace = blockType
	blockDef := g.engine.blockLibrary.GetBlockDefinition(g.blockTypeToPlace)
	g.engine.Print(fmt.Sprintf("Block: %s", blockDef.UniqueName))
}

func (g *GameStateEditMap) OnMouseClicked(x float64, y float64) {
	pos := g.engine.blockSelector.GetBlockPosition()
	println(fmt.Sprintf("Clicked at %d, %d, %d", pos.X, pos.Y, pos.Z))
	g.pencil.StartDragAt(pos)
}
func (g *GameStateEditMap) OnMouseReleased(x float64, y float64) {
	pos := g.engine.blockSelector.GetBlockPosition()
	println(fmt.Sprintf("Released at %d, %d, %d", pos.X, pos.Y, pos.Z))
	selection := g.pencil.StopDragAt(pos)
	g.PlaceRange(selection)
}
func (g *GameStateEditMap) ClearMap() {
	loadedMap := g.engine.GetVoxelMap()
	loadedMap.ClearAllChunks()
	loadedMap.SetFloorAtHeight(0, g.engine.blockLibrary.NewBlockFromName("bricks"))
	loadedMap.GenerateAllMeshes()
}
