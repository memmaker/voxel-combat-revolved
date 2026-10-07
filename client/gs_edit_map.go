package client

import (
	"fmt"
	"regexp"
	"sort"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/voxel"
	"github.com/memmaker/battleground/game"
)

type editorTool int

const (
	toolBlock editorTool = iota
	toolWall
	toolFill
	toolErase
	toolSpawnA
	toolSpawnB
	toolObjective
	toolLamp
)

var editorToolNames = []string{"Block", "Wall", "Floor", "Erase", "Spawn A", "Spawn B", "Objective", "Lamp"}

var editorToolHints = []string{
	"Click a face to put a block next to it.",
	"Drag a rectangle: builds walls of the chosen height ( [ / ] ).",
	"Drag a rectangle: fills a floor or roof on that level.",
	"Click a block to remove it, drag to clear an area.",
	"Drag over the ground to mark where team A deploys.",
	"Drag over the ground to mark where team B deploys.",
	"Click the ground: places a destructible target for Defend.",
	"Click to place a ceiling lamp that lights the room.",
}

// blockChange records a single block edit for undo.
type blockChange struct {
	pos voxel.Int3
	old byte
}

type editorSnapshot struct {
	blocks []blockChange
	meta   game.MapMetadata
}

// GameStateEditMap is the map editor: tools, block palette, undo and saving.
type GameStateEditMap struct {
	IsoMovementState
	tool        editorTool
	blockType   byte
	wallHeight  int32
	dragging    bool
	dragStart   voxel.Int3
	dragCurrent voxel.Int3
	undo        []editorSnapshot
	current     *editorSnapshot
	mapName     string
	nameFocused bool
	status      string
}

func NewEditorState(a *BattleClient) *GameStateEditMap {
	g := &GameStateEditMap{IsoMovementState: IsoMovementState{engine: a}, wallHeight: 3, mapName: "my_map"}
	g.blockType = a.GetBlockLibrary().GetBlockDefinitionByName("bricks").BlockID
	return g
}

func (g *GameStateEditMap) Init(bool) {
	g.engine.SwitchToBlockSelector()
	g.engine.highlights.ClearAll()
	g.engine.lines.Clear()
	g.engine.actionbar.SetActions(nil)
	g.engine.hud.SetHidden(true)
	if name := g.engine.GetMapFile(); name != "" && !game.IsGeneratedMap(name) {
		g.mapName = name
	}
	g.updateMarkers()
	// look at the middle of the map
	size := g.engine.GetVoxelMap().GetSize()
	voxelMap := g.engine.GetVoxelMap()
	g.engine.isoCamera.SetLookTarget(mgl32.Vec3{float32(size.X*voxelMap.ChunkSizeHorizontal) / 2, 1, float32(size.Z*voxelMap.ChunkSizeHorizontal) / 2})
	g.engine.isoCamera.ZoomOut(1, 40) // overview of the whole map
}

func (g *GameStateEditMap) OnServerMessage(string, string) {}

func (g *GameStateEditMap) meta() *game.MapMetadata {
	meta := g.engine.GetMapMetadata()
	for len(meta.SpawnPositions) < 2 {
		meta.SpawnPositions = append(meta.SpawnPositions, []voxel.Int3{})
	}
	return meta
}

// ---- undo ----

func (g *GameStateEditMap) beginChange() {
	meta := *g.meta()
	meta.SpawnPositions = [][]voxel.Int3{append([]voxel.Int3{}, meta.SpawnPositions[0]...), append([]voxel.Int3{}, meta.SpawnPositions[1]...)}
	meta.PoIPlacements = append([]voxel.Int3{}, meta.PoIPlacements...)
	meta.Lamps = append([]voxel.Int3{}, meta.Lamps...)
	g.current = &editorSnapshot{meta: meta}
}

func (g *GameStateEditMap) endChange() {
	if g.current != nil {
		g.undo = append(g.undo, *g.current)
		if len(g.undo) > 100 {
			g.undo = g.undo[1:]
		}
		g.current = nil
	}
	g.updateMarkers()
}

func (g *GameStateEditMap) setBlock(pos voxel.Int3, id byte) {
	voxelMap := g.engine.GetVoxelMap()
	if !voxelMap.ContainsGrid(pos) || pos.Y < 0 {
		return
	}
	old := voxelMap.GetBlockFromVec(pos)
	if old.ID == id {
		return
	}
	if g.current != nil {
		g.current.blocks = append(g.current.blocks, blockChange{pos: pos, old: old.ID})
	}
	if id == 0 {
		voxelMap.SetAir(pos)
	} else {
		voxelMap.SetBlock(pos.X, pos.Y, pos.Z, voxel.NewBlock(id))
	}
}

func (g *GameStateEditMap) Undo() {
	if len(g.undo) == 0 {
		g.status = "Nothing to undo"
		return
	}
	last := g.undo[len(g.undo)-1]
	g.undo = g.undo[:len(g.undo)-1]
	for i := len(last.blocks) - 1; i >= 0; i-- {
		change := last.blocks[i]
		if change.old == 0 {
			g.engine.GetVoxelMap().SetAir(change.pos)
		} else {
			g.engine.GetVoxelMap().SetBlock(change.pos.X, change.pos.Y, change.pos.Z, voxel.NewBlock(change.old))
		}
	}
	*g.engine.GetMapMetadata() = last.meta
	g.updateMarkers()
	g.status = "Undone"
}

// ---- tools ----

// targets returns the block under the cursor and the empty block in front of the face under the cursor.
func (g *GameStateEditMap) targets() (hit voxel.Int3, front voxel.Int3, ok bool) {
	info := g.engine.lastHitInfo
	if info == nil || !info.Hit {
		return voxel.Int3{}, voxel.Int3{}, false
	}
	return info.CollisionGridPosition, info.PreviousGridPosition, true
}

func (g *GameStateEditMap) OnMouseClicked(x float64, y float64) {
	hit, front, ok := g.targets()
	if !ok {
		return
	}
	switch g.tool {
	case toolBlock:
		g.beginChange()
		g.setBlock(front, g.blockType)
		g.endChange()
	case toolObjective:
		g.beginChange()
		target := g.engine.GetBlockLibrary().GetBlockDefinitionByName("target").BlockID
		g.setBlock(front, target)
		meta := g.meta()
		meta.PoIPlacements = append(meta.PoIPlacements, front.Add(voxel.Int3{Y: 1})) // objectives sit below their marker
		g.endChange()
	case toolLamp:
		g.beginChange()
		meta := g.meta()
		meta.Lamps = append(meta.Lamps, front)
		g.engine.GetVoxelMap().SetLamps(meta.Lamps)
		g.endChange()
	default:
		g.dragging = true
		if g.tool == toolErase {
			g.dragStart = hit
		} else {
			g.dragStart = front
		}
		g.dragCurrent = g.dragStart
	}
}

func (g *GameStateEditMap) OnMouseMoved(oldX float64, oldY float64, newX float64, newY float64) {
	g.IsoMovementState.OnMouseMoved(oldX, oldY, newX, newY)
	if !g.dragging {
		return
	}
	hit, front, ok := g.targets()
	if !ok {
		return
	}
	if g.tool == toolErase {
		g.dragCurrent = hit
	} else {
		g.dragCurrent = front
	}
	g.engine.SetSelectedBlocks(g.dragArea())
}

func (g *GameStateEditMap) dragArea() []voxel.Int3 {
	start, end := g.dragStart, g.dragCurrent
	var result []voxel.Int3
	for x := min(start.X, end.X); x <= max(start.X, end.X); x++ {
		for z := min(start.Z, end.Z); z <= max(start.Z, end.Z); z++ {
			onEdge := x == start.X || x == end.X || z == start.Z || z == end.Z
			if g.tool == toolWall && !onEdge {
				continue
			}
			result = append(result, voxel.Int3{X: x, Y: start.Y, Z: z})
		}
	}
	return result
}

func (g *GameStateEditMap) OnMouseReleased(x float64, y float64) {
	if !g.dragging {
		return
	}
	g.dragging = false
	area := g.dragArea()
	g.engine.SetSelectedBlocks(nil)
	g.beginChange()
	meta := g.meta()
	switch g.tool {
	case toolWall:
		for _, pos := range area {
			for h := int32(0); h < g.wallHeight; h++ {
				g.setBlock(pos.Add(voxel.Int3{Y: h}), g.blockType)
			}
		}
	case toolFill:
		for _, pos := range area {
			g.setBlock(pos, g.blockType)
		}
	case toolErase:
		for _, pos := range area {
			if pos.Y > 0 { // keep the indestructible ground layer
				g.setBlock(pos, 0)
			}
		}
	case toolSpawnA, toolSpawnB:
		team := 0
		if g.tool == toolSpawnB {
			team = 1
		}
		meta.SpawnPositions[team] = mergePositions(meta.SpawnPositions[team], area)
	}
	g.endChange()
}

func mergePositions(existing, added []voxel.Int3) []voxel.Int3 {
	seen := make(map[voxel.Int3]bool)
	for _, pos := range existing {
		seen[pos] = true
	}
	for _, pos := range added {
		if !seen[pos] {
			existing = append(existing, pos)
			seen[pos] = true
		}
	}
	return existing
}

func (g *GameStateEditMap) updateMarkers() {
	meta := g.meta()
	highlights := g.engine.highlights
	highlights.ClearFlat(voxel.HighlightEditor)
	highlights.AddFlat(voxel.HighlightEditor, meta.SpawnPositions[0], game.ColorTechTeal.Vec3())
	highlights.AddFlat(voxel.HighlightEditor, meta.SpawnPositions[1], game.ColorNegativeRed.Vec3())
	highlights.AddFlat(voxel.HighlightEditor, meta.PoIPlacements, mgl32.Vec3{1, 0.6, 0})
	highlights.AddFlat(voxel.HighlightEditor, meta.Lamps, mgl32.Vec3{1, 1, 0.4})
	highlights.ShowAsFlat(voxel.HighlightEditor)
}

// ---- keys ----

var mapNamePattern = regexp.MustCompile(`^[a-z0-9_\-]{1,32}$`)

func (g *GameStateEditMap) OnKeyPressed(key glfw.Key) {
	if g.nameFocused {
		g.typeName(key)
		return
	}
	switch {
	case key >= glfw.Key1 && key <= glfw.Key8:
		g.tool = editorTool(key - glfw.Key1)
	case key == glfw.KeyU:
		g.Undo()
	case key == glfw.KeyLeftBracket && g.wallHeight > 1:
		g.wallHeight--
	case key == glfw.KeyRightBracket && g.wallHeight < 12:
		g.wallHeight++
	default:
		g.IsoMovementState.OnKeyPressed(key)
	}
}

func (g *GameStateEditMap) typeName(key glfw.Key) {
	switch {
	case key == glfw.KeyBackspace || key == glfw.KeyDelete:
		if len(g.mapName) > 0 {
			g.mapName = g.mapName[:len(g.mapName)-1]
		}
	case key == glfw.KeyEnter || key == glfw.KeyEscape:
		g.nameFocused = false
	case key >= glfw.KeyA && key <= glfw.KeyZ && len(g.mapName) < 32:
		g.mapName += string(rune('a' + key - glfw.KeyA))
	case key >= glfw.Key0 && key <= glfw.Key9 && len(g.mapName) < 32:
		g.mapName += string(rune('0' + key - glfw.Key0))
	case key == glfw.KeyMinus && len(g.mapName) < 32:
		g.mapName += "_"
	}
}

// Save writes the map and registers it in the map list.
func (g *GameStateEditMap) Save() bool {
	if !mapNamePattern.MatchString(g.mapName) {
		g.status = "Name: use lowercase letters, digits, - and _"
		return false
	}
	meta := g.meta()
	if len(meta.SpawnPositions[0]) == 0 || len(meta.SpawnPositions[1]) == 0 {
		g.status = "Mark spawn zones for both teams before saving (tools 5 and 6)"
		return false
	}
	if err := g.engine.SaveMapAs(g.mapName); err != nil {
		g.status = "Saving failed: " + err.Error()
		return false
	}
	if err := game.RegisterMapInContent(g.mapName); err != nil {
		g.status = "Saved, but not added to the map list: " + err.Error()
		return true
	}
	g.status = fmt.Sprintf("Saved as assets/maps/%s.bin", g.mapName)
	return true
}

// ---- UI ----

// editorPanelBounds are the screen areas that belong to the editor UI instead of the map.
func (a *BattleClient) isOverEditorUI(x, y float64) bool {
	return y < 46 || x < 200 || y > float64(a.WindowHeight)-28
}

func (a *BattleClient) drawEditorUI(g *GameStateEditMap) {
	ui := a.ui
	ui.Begin(a.WindowWidth, a.WindowHeight, a.lastMousePosX, a.lastMousePosY, a.uiClicked)
	a.uiClicked = false
	width := float32(a.WindowWidth)

	// top toolbar
	ui.Fill(0, 0, width, 46, uiPanel)
	x := float32(206)
	for i, name := range editorToolNames {
		label := fmt.Sprintf("%d %s", i+1, name)
		buttonWidth := float32(len(label)*8 + 16)
		if g.tool == editorTool(i) {
			ui.Fill(x-2, 6, buttonWidth+4, 34, uiAccent)
		}
		if ui.Button(x, 8, buttonWidth, 30, label, true) {
			g.tool = editorTool(i)
		}
		x += buttonWidth + 6
	}
	right := width - 8
	for _, action := range []struct {
		label string
		do    func()
	}{
		{"Exit", func() { a.finishEditor(EditorResult{}) }},
		{"Play", func() {
			if g.Save() {
				a.finishEditor(EditorResult{PlayMap: g.mapName})
			}
		}},
		{"Save", func() { g.Save() }},
		{"Undo (U)", g.Undo},
	} {
		buttonWidth := float32(len(action.label)*8 + 20)
		right -= buttonWidth
		if ui.Button(right, 8, buttonWidth, 30, action.label, true) {
			action.do()
		}
		right -= 6
	}

	// block palette
	ui.Fill(0, 46, 200, float32(a.WindowHeight)-46, uiPanel)
	ui.Label(10, 54, "Map name (click to edit)", 1, uiTextDim)
	nameText := g.mapName
	if g.nameFocused {
		nameText += "_"
	}
	ui.Fill(6, 72, 188, 24, uiBackground)
	if g.nameFocused {
		ui.Fill(6, 94, 188, 2, uiAccent)
	}
	ui.Label(12, 77, nameText, 1, uiTextColor)
	if a.lastMousePosX < 194 && a.lastMousePosY > 72 && a.lastMousePosY < 96 && ui.clicked && !ui.ConsumedClick() {
		g.nameFocused = true
	}
	ui.Label(10, 106, "Blocks", 1, uiTextHighlight)
	library := a.GetBlockLibrary()
	var names []string
	ids := make(map[string]byte)
	for id := byte(1); id <= library.LastBlockID(); id++ {
		definition := library.GetBlockDefinition(id)
		if definition == nil || definition.UniqueName == "debug2" || definition.UniqueName == "selection" {
			continue
		}
		names = append(names, definition.UniqueName)
		ids[definition.UniqueName] = id
	}
	sort.Strings(names)
	rowHeight := float32(min(20, (a.WindowHeight-160)/max(1, (len(names)+1)/2)))
	for i, name := range names {
		bx := float32(6 + (i%2)*96)
		by := 126 + float32(i/2)*rowHeight
		if ids[name] == g.blockType {
			ui.Fill(bx-2, by-1, 94, rowHeight, uiAccent)
		}
		short := name
		if len(short) > 11 {
			short = short[:11]
		}
		if ui.Button(bx, by, 90, rowHeight-2, short, true) {
			g.blockType = ids[name]
			if g.tool == toolErase || g.tool >= toolSpawnA {
				g.tool = toolBlock
			}
		}
	}

	// status line
	ui.Fill(0, float32(a.WindowHeight)-28, width, 28, uiPanel)
	hint := editorToolHints[g.tool]
	if g.tool == toolWall {
		hint += fmt.Sprintf(" Height: %d", g.wallHeight)
	}
	status := hint + "   Z / C: levels, WASD / Q / E: camera"
	if g.status != "" {
		status = g.status + "   |   " + hint
	}
	ui.Label(206, float32(a.WindowHeight)-21, status, 1, uiTextColor)
	ui.Draw()
}

// EditorResult tells the caller what to do after the editor closes.
type EditorResult struct {
	PlayMap string // non-empty: start a test match on this map
}

func (a *BattleClient) finishEditor(result EditorResult) {
	a.editorResult = result
	a.StopRequested = true
}
