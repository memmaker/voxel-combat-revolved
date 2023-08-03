package client

import (
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/etxt"
	"github.com/memmaker/battleground/engine/glhf"
	"github.com/memmaker/battleground/engine/gui"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
	"github.com/memmaker/battleground/game"
	"sync"
)

type BattleGame struct {
	*util.GlApplication
	lastMousePosX         float64
	lastMousePosY         float64
	voxelMap              *voxel.Map
	wireFrame             bool
	selector              PositionDrawable
	crosshair             PositionDrawable
	modelShader           *glhf.Shader
	chunkShader           *glhf.Shader
	lineShader            *glhf.Shader
	guiShader             *glhf.Shader
	highlightShader       *glhf.Shader
	textLabel             *etxt.TextMesh
	textRenderer          *etxt.OpenGLTextRenderer
	lastHitInfo           *game.RayCastHit
	allUnits              []*Unit
	userUnits             []*Unit
	projectiles           []*Projectile
	debugObjects          []PositionDrawable
	blockTypeToPlace      byte
	isoCamera             *util.ISOCamera
	fpsCamera             *util.FPSCamera
	cameraIsFirstPerson   bool
	drawBoundingBoxes     bool
	showDebugInfo         bool
	timer                 *util.Timer
	collisionSolver       *util.CollisionSolver
	stateStack            []GameState
	updateQueue           []func(deltaTime float64)
	isBlockSelection      bool
	groundSelector        *GroundSelector
	unitSelector          *GroundSelector
	blockSelector         *util.LineMesh
	currentFactionIndex   int
	currentVisibleEnemies map[*Unit][]*Unit
	projectileTexture     *glhf.Texture
	actionbar             *gui.ActionBar
	server                *game.ServerConnection
	unitMap               map[uint64]*Unit

	mu sync.Mutex
}

func (a *BattleGame) state() GameState {
	return a.stateStack[len(a.stateStack)-1]
}

func (a *BattleGame) IsOccludingBlock(x, y, z int) bool {
	if a.voxelMap.IsSolidBlockAt(int32(x), int32(y), int32(z)) {
		return !a.voxelMap.GetGlobalBlock(int32(x), int32(y), int32(z)).IsAir()
	}
	return false
}

func NewBattleGame(title string, width int, height int) *BattleGame {
	window, terminateFunc := util.InitOpenGL(title, width, height)
	glApp := &util.GlApplication{
		WindowWidth:   width,
		WindowHeight:  height,
		Window:        window,
		TerminateFunc: terminateFunc,
	}
	window.SetKeyCallback(glApp.KeyCallback)
	window.SetCursorPosCallback(glApp.MousePosCallback)
	window.SetMouseButtonCallback(glApp.MouseButtonCallback)
	window.SetScrollCallback(glApp.ScrollCallback)

	myApp := &BattleGame{
		GlApplication: glApp,
		isoCamera:     util.NewISOCamera(width, height),
		fpsCamera:     util.NewFPSCamera(mgl32.Vec3{0, 10, 0}, width, height),
		timer:         util.NewTimer(),
		unitMap:       make(map[uint64]*Unit),
	}
	myApp.modelShader = myApp.loadModelShader()
	myApp.chunkShader = myApp.loadChunkShader()
	myApp.highlightShader = myApp.loadHighlightShader()
	myApp.lineShader = myApp.loadLineShader()
	myApp.guiShader = myApp.loadGuiShader()
	myApp.projectileTexture = glhf.NewSolidColorTexture([3]uint8{255, 12, 255})
	myApp.textRenderer = etxt.NewOpenGLTextRenderer(myApp.guiShader)
	myApp.DrawFunc = myApp.Draw
	myApp.UpdateFunc = myApp.Update
	myApp.KeyHandler = myApp.handleKeyEvents
	myApp.MousePosHandler = myApp.handleMousePosEvents
	myApp.MouseButtonHandler = myApp.handleMouseButtonEvents
	myApp.ScrollHandler = myApp.handleScrollEvents

	myApp.unitSelector = NewGroundSelector(util.LoadGLTF("./assets/models/flatselector.glb"), myApp.modelShader)

	selectorMesh := util.LoadGLTF("./assets/models/selector.glb")
	selectorMesh.SetTexture(0, glhf.NewSolidColorTexture([3]uint8{0, 248, 250}))
	myApp.groundSelector = NewGroundSelector(selectorMesh, myApp.modelShader)
	myApp.blockSelector = NewBlockSelector(myApp.lineShader)

	guiAtlas, _ := util.CreateAtlasFromDirectory("./assets/gui", []string{"walk", "reticule"})
	myApp.actionbar = gui.NewActionBar(myApp.guiShader, guiAtlas, glApp.WindowWidth, glApp.WindowHeight, 64, 64)

	myApp.SwitchToBlockSelector()
	myApp.textRenderer.SetAlign(etxt.YCenter, etxt.XCenter)
	crosshair := myApp.textRenderer.DrawText("+")
	crosshair.SetPosition(mgl32.Vec3{float32(glApp.WindowWidth) / 2, float32(glApp.WindowHeight) / 2, 0})
	myApp.textRenderer.SetAlign(etxt.Top, etxt.Left)
	myApp.SetCrosshair(crosshair)

	isSolid := func(x, y, z int32) bool {
		currentMap := myApp.voxelMap
		if !currentMap.Contains(x, y, z) {
			return true
		}
		block := currentMap.GetGlobalBlock(x, y, z)
		return block != nil && !block.IsAir()
	}
	myApp.collisionSolver = util.NewCollisionSolver(isSolid)

	return myApp
}

func NewBlockSelector(shader *glhf.Shader) *util.LineMesh {
	blockSelector := util.NewLineMesh(shader, [][2]mgl32.Vec3{
		// we need to draw 12 lines, each line has 2 points, should be a wireframe cube
		// bottom
		{mgl32.Vec3{0, 0, 0}, mgl32.Vec3{1, 0, 0}},
		{mgl32.Vec3{1, 0, 0}, mgl32.Vec3{1, 0, 1}},
		{mgl32.Vec3{1, 0, 1}, mgl32.Vec3{0, 0, 1}},
		{mgl32.Vec3{0, 0, 1}, mgl32.Vec3{0, 0, 0}},
		// top
		{mgl32.Vec3{0, 1, 0}, mgl32.Vec3{1, 1, 0}},
		{mgl32.Vec3{1, 1, 0}, mgl32.Vec3{1, 1, 1}},
		{mgl32.Vec3{1, 1, 1}, mgl32.Vec3{0, 1, 1}},
		{mgl32.Vec3{0, 1, 1}, mgl32.Vec3{0, 1, 0}},

		// sides
		{mgl32.Vec3{0, 0, 0}, mgl32.Vec3{0, 1, 0}},
		{mgl32.Vec3{1, 0, 0}, mgl32.Vec3{1, 1, 0}},
		{mgl32.Vec3{1, 0, 1}, mgl32.Vec3{1, 1, 1}},
		{mgl32.Vec3{0, 0, 1}, mgl32.Vec3{0, 1, 1}},
	})

	return blockSelector
}

type PositionDrawable interface {
	SetPosition(pos mgl32.Vec3)
	Draw()
}

func (a *BattleGame) LoadModel(filename string) *util.CompoundMesh {
	compoundMesh := util.LoadGLTF(filename)
	compoundMesh.ConvertVertexData(a.modelShader)
	return compoundMesh
}

// Q: when do we add the other units?
// A: when first seen.
func (a *BattleGame) AddOwnedUnit(spawnPos mgl32.Vec3, unitID uint64, unitDefinition *game.UnitDefinition, name string) *Unit {
	//model := a.LoadModel("./assets/model.gltf")
	model := a.LoadModel("./assets/models/Guard3.glb")
	//model.SetTexture(0, util.MustLoadTexture("./assets/mc.fire.ice.png"))
	//model.SetTexture(0, util.MustLoadTexture("./assets/Agent_47.png"))
	model.SetTexture(0, util.MustLoadTexture(unitDefinition.ClientRepresentation.TextureFile))
	model.SetAnimation("animation.idle")
	unit := NewUnit(unitID, name, spawnPos, model, &unitDefinition.CoreStats)
	unit.SetMap(a.voxelMap)
	a.collisionSolver.AddObject(unit)
	a.allUnits = append(a.allUnits, unit)
	a.userUnits = append(a.userUnits, unit)
	a.unitMap[unitID] = unit
	return unit
}

func (a *BattleGame) SpawnProjectile(pos, velocity mgl32.Vec3) {
	projectile := NewProjectile(a.modelShader, a.projectileTexture, pos)
	//projectile.SetCollisionHandler(a.GetCollisionHandler())
	a.collisionSolver.AddProjectile(projectile)
	projectile.SetVelocity(velocity)
	a.projectiles = append(a.projectiles, projectile)
}

func (a *BattleGame) SetCrosshair(crosshair PositionDrawable) {
	a.crosshair = crosshair
}

func (a *BattleGame) Print(text string) {
	a.textLabel = a.textRenderer.DrawText(text)
}

func (a *BattleGame) Update(elapsed float64) {
	stopUpdateTimer := a.timer.Start("> Update()")

	a.mu.Lock()
	for _, f := range a.updateQueue {
		f(elapsed)
	}
	// what happens when we try writing to this slice exactly in between these two lines?
	a.updateQueue = a.updateQueue[:0]
	a.mu.Unlock()

	camMoved, movementVector := a.pollInput(elapsed)
	if camMoved {
		//a.HandlePlayerCollision()
		a.state().OnDirectionKeys(elapsed, movementVector)
	}
	a.updateUnits(elapsed)
	//a.updateProjectiles(elapsed)
	a.collisionSolver.Update(elapsed)

	a.updateDebugInfo()
	stopUpdateTimer()

}

func (a *BattleGame) updateUnits(deltaTime float64) {
	for i := len(a.allUnits) - 1; i >= 0; i-- {
		actor := a.allUnits[i]
		actor.Update(deltaTime)
		if actor.ShouldBeRemoved() {
			a.allUnits = append(a.allUnits[:i], a.allUnits[i+1:]...)
		}
	}
}

func (a *BattleGame) Draw(elapsed float64) {
	stopDrawTimer := a.timer.Start("> Draw()")

	var camera util.Camera = a.isoCamera
	if a.cameraIsFirstPerson {
		camera = a.fpsCamera
	}

	a.drawWorld(camera)

	a.drawModels(camera)

	a.drawLines(camera)

	a.drawGUI()
	stopDrawTimer()
}

func (a *BattleGame) drawWorld(cam util.Camera) {
	a.chunkShader.Begin()

	a.chunkShader.SetUniformAttr(0, cam.GetProjectionMatrix())

	a.chunkShader.SetUniformAttr(1, cam.GetViewMatrix())

	a.voxelMap.Draw(cam.GetFront(), cam.GetFrustumPlanes(cam.GetProjectionMatrix()))

	a.chunkShader.End()
}

func (a *BattleGame) drawModels(cam util.Camera) {
	a.modelShader.Begin()

	a.modelShader.SetUniformAttr(1, cam.GetViewMatrix())

	if a.selector != nil && a.lastHitInfo != nil && !a.isBlockSelection {
		a.modelShader.SetUniformAttr(0, cam.GetProjectionMatrix())
		a.modelShader.SetUniformAttr(1, cam.GetViewMatrix())
		a.selector.Draw()
	}
	if a.unitSelector != nil {
		a.unitSelector.Draw()
	}
	for _, unit := range a.userUnits {
		unit.Draw(a.modelShader)
	}
	for unit, _ := range a.CurrentVisibleEnemiesList() {
		unit.Draw(a.modelShader)
	}

	a.drawProjectiles()

	a.modelShader.End()
}

func (a *BattleGame) drawProjectiles() {
	for i := len(a.projectiles) - 1; i >= 0; i-- {
		projectile := a.projectiles[i]
		projectile.Draw()
		if projectile.IsDead() {
			a.projectiles = append(a.projectiles[:i], a.projectiles[i+1:]...)
		}
	}
}
func (a *BattleGame) drawLines(cam util.Camera) {
	a.lineShader.Begin()
	a.lineShader.SetUniformAttr(3, mgl32.Vec3{0, 0, 0})
	if a.selector != nil && a.lastHitInfo != nil && a.isBlockSelection {
		a.lineShader.SetUniformAttr(0, cam.GetProjectionMatrix())
		a.lineShader.SetUniformAttr(1, cam.GetViewMatrix())
		a.selector.Draw()
	}
	for _, drawable := range a.debugObjects {
		drawable.Draw()
	}
	if a.drawBoundingBoxes { // TODO: SOMETHING IS VERY WRONG WITH DRAWING BOUNDING BOXES
		a.lineShader.SetUniformAttr(2, mgl32.Ident4())
		a.lineShader.SetUniformAttr(3, mgl32.Vec3{0.1, 1.0, 0.1})
		//a.player.GetAABB().Draw(a.lineShader)
		for _, actor := range a.allUnits {
			actor.GetAABB().Draw(a.lineShader)
		}
		for _, projectile := range a.projectiles {
			projectile.GetAABB().Draw(a.lineShader)
		}
	}
	a.lineShader.End()
}

func (a *BattleGame) drawGUI() {
	a.guiShader.Begin()

	if a.textLabel != nil {
		a.textLabel.Draw()
	}

	if a.actionbar != nil {
		a.actionbar.Draw()
	}
	if a.crosshair != nil {
		a.crosshair.Draw()
	}
	a.guiShader.End()
}

func (a *BattleGame) SwitchToUnit(unit *Unit) {
	a.stateStack = []GameState{&GameStateUnit{IsoMovementState: IsoMovementState{engine: a}, selectedUnit: unit}}
	a.state().Init(false)
}

func (a *BattleGame) SwitchToAction(unit *Unit, action game.Action) {
	a.stateStack = append(a.stateStack, &GameStateAction{IsoMovementState: IsoMovementState{engine: a}, selectedUnit: unit, selectedAction: action})
	a.state().Init(false)
}

func (a *BattleGame) SwitchToFreeAim(unit *Unit, action *ActionShot) {
	a.stateStack = append(a.stateStack, &GameStateFreeAim{engine: a, selectedUnit: unit, selectedAction: action})
	a.state().Init(false)
}

func (a *BattleGame) SwitchToEditMap() {
	a.stateStack = append(a.stateStack, &GameStateEditMap{IsoMovementState: IsoMovementState{engine: a}})
	a.state().Init(false)
}

func (a *BattleGame) scheduleUpdate(f func(deltaTime float64)) {
	a.mu.Lock()
	a.updateQueue = append(a.updateQueue, f)
	a.mu.Unlock()
}

func (a *BattleGame) PopState() {
	if len(a.stateStack) > 1 {
		a.stateStack = a.stateStack[:len(a.stateStack)-1]
	}
	a.state().Init(true)
}
func (a *BattleGame) UpdateMousePicking(newX, newY float64) {
	rayStart, rayEnd := a.isoCamera.GetPickingRayFromScreenPosition(newX, newY)
	//a.placeDebugLine([2]mgl32.Vec3{rayStart, rayEnd})
	hitInfo := a.RayCast(rayStart, rayEnd)
	if hitInfo.HitUnit() {
		unitHit := hitInfo.UnitHit.(*Unit)
		a.selector.SetPosition(util.ToGrid(unitHit.GetFootPosition()))
	} else if hitInfo.Hit {
		if a.isBlockSelection {
			a.selector.SetPosition(hitInfo.PreviousGridPosition.ToVec3())
		} else {
			a.selector.SetPosition(a.voxelMap.GetGroundPosition(hitInfo.PreviousGridPosition).ToVec3())
		}
	}
}

func (a *BattleGame) SwitchToGroundSelector() {
	a.selector = a.groundSelector
	a.isBlockSelection = false
}

type GroundSelector struct {
	mesh   *util.CompoundMesh
	shader *glhf.Shader
	hide   bool
}

func (g *GroundSelector) SetPosition(pos mgl32.Vec3) {
	offset := mgl32.Vec3{0.5, 0.025, 0.5}
	g.mesh.SetPosition(pos.Add(offset))
	g.hide = false
}

func (g *GroundSelector) Hide() {
	g.hide = true
}

func (g *GroundSelector) Draw() {
	if g.hide {
		return
	}
	g.mesh.Draw(g.shader)
}

func (g *GroundSelector) GetBlockPosition() voxel.Int3 {
	return voxel.ToGridInt3(g.mesh.GetPosition())
}

func NewGroundSelector(mesh *util.CompoundMesh, shader *glhf.Shader) *GroundSelector {
	groundSelector := &GroundSelector{
		mesh:   mesh,
		shader: shader,
		hide:   true,
	}
	mesh.ConvertVertexData(shader)
	return groundSelector
}

func (a *BattleGame) SwitchToBlockSelector() {
	a.selector = a.blockSelector
	a.isBlockSelection = true
}

func (a *BattleGame) GetNextUnit(unit *Unit) (*Unit, bool) {
	units := a.userUnits
	for i, u := range units {
		if u == unit {
			for j := 1; j < len(units); j++ {
				nextUnitIndex := (i + j) % len(units)
				nextUnit := units[nextUnitIndex]
				if nextUnit.CanAct() {
					return nextUnit, true
				}
			}
			return nil, false
		}
	}
	return nil, false
}

func (a *BattleGame) GetVisibleUnits(unit *Unit) []*Unit {
	if _, ok := a.currentVisibleEnemies[unit]; ok {
		return a.currentVisibleEnemies[unit]
	}
	return make([]*Unit, 0)
}

func (a *BattleGame) SwitchToFirstPerson(position mgl32.Vec3) {
	a.captureMouse()
	a.fpsCamera.SetPosition(position)
	a.cameraIsFirstPerson = true
}

func (a *BattleGame) SwitchToIsoCamera() {
	a.freeMouse()
	a.cameraIsFirstPerson = false
}

func (a *BattleGame) SetConnection(connection *game.ServerConnection) {
	a.server = connection
	scheduleOnMainthread := func(msgType, data string) {
		println(fmt.Sprintf("[BattleGame] Received message %s", msgType))
		a.scheduleUpdate(func(deltaTime float64) {
			a.OnServerMessage(msgType, data)
		})
	}
	connection.SetEventHandler(scheduleOnMainthread)
}

func (a *BattleGame) OnServerMessage(msgType, messageAsJson string) {
	switch msgType {
	case "ConfirmedTargetedUnitAction":
		var msg game.TargetedUnitActionMessage
		if util.FromJson(messageAsJson, &msg) {
			a.OnConfirmedTargetedUnitAction(msg)
		}
	case "NextPlayer":
		var msg game.NextPlayerMessage
		if util.FromJson(messageAsJson, &msg) {
			a.OnNextPlayer(msg)
		}
	}

}

func (a *BattleGame) OnConfirmedTargetedUnitAction(msg game.TargetedUnitActionMessage) {
	unit, known := a.unitMap[msg.GameUnitID]
	if !known {
		println(fmt.Sprintf("[BattleGame] Unknown unit %d", msg.GameUnitID))
		return
	}
	clientAction := a.animateAction(unit, msg.Action, msg.Target)
	if clientAction != nil {
		if clientAction.IsValidTarget(unit, msg.Target) {
			clientAction.Execute(unit, msg.Target)
		}
	}
}

func (a *BattleGame) OnNextPlayer(msg game.NextPlayerMessage) {
	println(fmt.Sprintf("[BattleGame] NextPlayer: %v", msg))
	if msg.YourTurn {
		a.ResetUnitsForNextTurn()
		a.Print("It's your turn!")
		a.SwitchToUnit(a.FirstUnit())
	} else {
		a.Print("Waiting for other players...")
		a.SwitchToWaitForEvents()
	}
}

func (a *BattleGame) CurrentVisibleEnemiesList() map[*Unit]bool {
	result := make(map[*Unit]bool)
	for _, visibles := range a.currentVisibleEnemies {
		for _, visible := range visibles {
			result[visible] = true
		}
	}
	return result
}

func (a *BattleGame) EndTurn() {
	util.MustSend(a.server.EndTurn())
	a.SwitchToWaitForEvents()
}

func (a *BattleGame) SwitchToWaitForEvents() {
	a.stateStack = []GameState{&GameStateWaitForEvents{IsoMovementState{engine: a}}}
	a.state().Init(false)
}

func (a *BattleGame) FirstUnit() *Unit {
	for _, unit := range a.userUnits {
		if unit.CanAct() {
			return unit
		}
	}
	return nil
}

// concept: actions
// actions are split between client and server
// the server action contains the actual game logic that will alter the world state
// the corresponding client action contains the animation logic that will be played on the client
// so server changes state and client visualizes it

func (a *BattleGame) animateAction(unit *Unit, action string, target voxel.Int3) game.Action {
	switch action {
	case "Move":
		return game.NewActionMove(a.voxelMap)
	}
	return nil
}

func (a *BattleGame) ResetUnitsForNextTurn() {
	for _, unit := range a.userUnits {
		unit.NextTurn()
	}
}
