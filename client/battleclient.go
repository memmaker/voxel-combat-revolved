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

type BattleClient struct {
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
	conditionQueue        []ConditionalCall
	isBlockSelection      bool
	groundSelector        *GroundSelector
	unitSelector          *GroundSelector
	blockSelector         *util.LineMesh
	currentFactionIndex   int
	currentVisibleEnemies map[*Unit]map[*Unit]bool
	projectileTexture     *glhf.Texture
	actionbar             *gui.ActionBar
	server                *game.ServerConnection
	unitMap               map[uint64]*Unit
	mu                    sync.Mutex
	mc                    sync.Mutex
}

func (a *BattleClient) GetVisibleUnits(instance game.UnitCore) []game.UnitCore {
	result := make([]game.UnitCore, 0)
	for enemy, isVisble := range a.currentVisibleEnemies[instance.(*Unit)] {
		if isVisble {
			result = append(result, enemy)
		}
	}
	return result
}

func (a *BattleClient) CurrentVisibleEnemiesList() map[*Unit]bool {
	result := make(map[*Unit]bool)
	for observer, unitsVisible := range a.currentVisibleEnemies {
		if !observer.IsUserControlled() {
			continue
		}
		for unit, isVisible := range unitsVisible {
			if isVisible {
				result[unit] = true
			}
		}
	}
	return result
}
func (a *BattleClient) GetVoxelMap() *voxel.Map {
	//TODO implement me
	panic("implement me")
}

func (a *BattleClient) state() GameState {
	return a.stateStack[len(a.stateStack)-1]
}

func (a *BattleClient) IsOccludingBlock(x, y, z int) bool {
	if a.voxelMap.IsSolidBlockAt(int32(x), int32(y), int32(z)) {
		return !a.voxelMap.GetGlobalBlock(int32(x), int32(y), int32(z)).IsAir()
	}
	return false
}

func NewBattleGame(title string, width int, height int) *BattleClient {
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

	myApp := &BattleClient{
		GlApplication:         glApp,
		isoCamera:             util.NewISOCamera(width, height),
		fpsCamera:             util.NewFPSCamera(mgl32.Vec3{0, 10, 0}, width, height),
		timer:                 util.NewTimer(),
		unitMap:               make(map[uint64]*Unit),
		currentVisibleEnemies: make(map[*Unit]map[*Unit]bool),
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

	guiAtlas, _ := util.CreateAtlasFromDirectory("./assets/gui", []string{"walk", "ranged", "reticule", "next-turn"})
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

func (a *BattleClient) LoadModel(filename string) *util.CompoundMesh {
	compoundMesh := util.LoadGLTF(filename)
	compoundMesh.ConvertVertexData(a.modelShader)
	return compoundMesh
}

// Q: when do we add the other units?
// A: when first seen.
func (a *BattleClient) AddOwnedUnit(unitInstance *game.UnitInstance, playerID uint64) *Unit {
	unitID := unitInstance.GameUnitID
	unitDefinition := unitInstance.UnitDefinition
	spawnPos := unitInstance.SpawnPos.ToBlockCenterVec3()
	name := unitInstance.Name
	//model := a.LoadModel("./assets/model.gltf")
	model := a.LoadModel("./assets/models/Guard3.glb")
	//model.SetTexture(0, util.MustLoadTexture("./assets/mc.fire.ice.png"))
	//model.SetTexture(0, util.MustLoadTexture("./assets/Agent_47.png"))
	model.SetTexture(0, util.MustLoadTexture(unitDefinition.ClientRepresentation.TextureFile))
	model.SetAnimation("animation.idle")
	unit := NewUnit(unitID, name, spawnPos, model, &unitDefinition.CoreStats)
	unit.SetControlledBy(playerID)
	unit.SetUserControlled()
	unit.SetMap(a.voxelMap)
	a.collisionSolver.AddObject(unit)
	a.allUnits = append(a.allUnits, unit)
	a.userUnits = append(a.userUnits, unit)
	a.unitMap[unitID] = unit
	return unit
}
func (a *BattleClient) RemoveVisibleEnemyUnit(observer *Unit, unitInstance *game.UnitInstance) {
	unitID := unitInstance.GameUnitID
	if unit, ok := a.unitMap[unitID]; ok {
		if _, hasVisionMap := a.currentVisibleEnemies[observer]; !hasVisionMap {
			a.currentVisibleEnemies[observer] = make(map[*Unit]bool)
		}
		a.currentVisibleEnemies[observer][unit] = false
	}
}
func (a *BattleClient) AddVisibleEnemyUnit(observer *Unit, unitInstance *game.UnitInstance) {
	unitID := unitInstance.GameUnitID

	if unit, ok := a.unitMap[unitID]; ok {
		// already added, just change the visibility and update the position
		a.voxelMap.MoveUnitTo(unit, unit.GetFootPosition(), unitInstance.Position.ToBlockCenterVec3())
		//unit.SetFootPosition(unitInstance.Position.ToBlockCenterVec3())
		a.currentVisibleEnemies[observer][unit] = true
		return
	}

	unitDefinition := unitInstance.UnitDefinition
	spawnPos := unitInstance.SpawnPos.ToBlockCenterVec3()
	name := unitInstance.Name

	model := a.LoadModel("./assets/models/Guard3.glb")
	//model.SetTexture(0, util.MustLoadTexture("./assets/mc.fire.ice.png"))
	//model.SetTexture(0, util.MustLoadTexture("./assets/Agent_47.png"))
	model.SetTexture(0, util.MustLoadTexture(unitDefinition.ClientRepresentation.TextureFile))
	model.SetAnimation("animation.idle")
	unit := NewUnit(unitID, name, spawnPos, model, &unitDefinition.CoreStats)
	unit.SetControlledBy(unitInstance.ControlledBy())
	unit.SetMap(a.voxelMap)

	a.collisionSolver.AddObject(unit)

	a.allUnits = append(a.allUnits, unit)

	if _, ok := a.currentVisibleEnemies[observer]; !ok {
		a.currentVisibleEnemies[observer] = make(map[*Unit]bool)
	}
	a.currentVisibleEnemies[observer][unit] = true
	a.unitMap[unitID] = unit
}
func (a *BattleClient) SpawnProjectile(pos, velocity mgl32.Vec3) {
	projectile := NewProjectile(a.modelShader, a.projectileTexture, pos)
	//projectile.SetCollisionHandler(a.GetCollisionHandler())
	a.collisionSolver.AddProjectile(projectile)
	projectile.SetVelocity(velocity)
	a.projectiles = append(a.projectiles, projectile)
}

func (a *BattleClient) SetCrosshair(crosshair PositionDrawable) {
	a.crosshair = crosshair
}

func (a *BattleClient) Print(text string) {
	a.textLabel = a.textRenderer.DrawText(text)
}

func (a *BattleClient) Update(elapsed float64) {
	stopUpdateTimer := a.timer.Start("> Update()")

	a.mu.Lock()
	for _, f := range a.updateQueue {
		f(elapsed)
	}
	// what happens when we try writing to this slice exactly in between these two lines?
	a.updateQueue = a.updateQueue[:0]
	a.mu.Unlock()
	a.mc.Lock()
	for i := len(a.conditionQueue) - 1; i >= 0; i-- {
		c := a.conditionQueue[i]
		if c.condition() {
			c.function()
			a.conditionQueue = append(a.conditionQueue[:i], a.conditionQueue[i+1:]...)
		}
	}
	a.mc.Unlock()

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

func (a *BattleClient) updateUnits(deltaTime float64) {
	for i := len(a.allUnits) - 1; i >= 0; i-- {
		actor := a.allUnits[i]
		actor.Update(deltaTime)
		if actor.ShouldBeRemoved() {
			a.allUnits = append(a.allUnits[:i], a.allUnits[i+1:]...)
		}
	}
}

func (a *BattleClient) Draw(elapsed float64) {
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

func (a *BattleClient) drawWorld(cam util.Camera) {
	a.chunkShader.Begin()

	a.chunkShader.SetUniformAttr(0, cam.GetProjectionMatrix())

	a.chunkShader.SetUniformAttr(1, cam.GetViewMatrix())

	a.voxelMap.Draw(cam.GetFront(), cam.GetFrustumPlanes(cam.GetProjectionMatrix()))

	a.chunkShader.End()
}

func (a *BattleClient) drawModels(cam util.Camera) {
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

func (a *BattleClient) drawProjectiles() {
	for i := len(a.projectiles) - 1; i >= 0; i-- {
		projectile := a.projectiles[i]
		projectile.Draw()
		if projectile.IsDead() {
			a.projectiles = append(a.projectiles[:i], a.projectiles[i+1:]...)
		}
	}
}
func (a *BattleClient) drawLines(cam util.Camera) {
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

func (a *BattleClient) drawGUI() {
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

func (a *BattleClient) SwitchToUnit(unit *Unit) {
	a.stateStack = []GameState{&GameStateUnit{IsoMovementState: IsoMovementState{engine: a}, selectedUnit: unit}}
	a.state().Init(false)
}

func (a *BattleClient) SwitchToAction(unit *Unit, action game.TargetAction) {
	a.stateStack = append(a.stateStack, &GameStateAction{IsoMovementState: IsoMovementState{engine: a}, selectedUnit: unit, selectedAction: action})
	a.state().Init(false)
}

func (a *BattleClient) SwitchToFreeAim(unit *Unit, action *game.ActionShot) {
	a.stateStack = append(a.stateStack, &GameStateFreeAim{engine: a, selectedUnit: unit, selectedAction: action})
	a.state().Init(false)
}

func (a *BattleClient) SwitchToEditMap() {
	a.stateStack = append(a.stateStack, &GameStateEditMap{IsoMovementState: IsoMovementState{engine: a}})
	a.state().Init(false)
}

func (a *BattleClient) scheduleUpdate(f func(deltaTime float64)) {
	a.mu.Lock()
	a.updateQueue = append(a.updateQueue, f)
	a.mu.Unlock()
}

type ConditionalCall struct {
	condition func() bool
	function  func()
}

func (a *BattleClient) scheduleWaitForCondition(condition func() bool, function func()) {
	a.mc.Lock()
	a.conditionQueue = append(a.conditionQueue, ConditionalCall{condition: condition, function: function})
	a.mc.Unlock()
}

func (a *BattleClient) PopState() {
	if len(a.stateStack) > 1 {
		a.stateStack = a.stateStack[:len(a.stateStack)-1]
	}
	a.state().Init(true)
}
func (a *BattleClient) UpdateMousePicking(newX, newY float64) {
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

func (a *BattleClient) SwitchToGroundSelector() {
	a.selector = a.groundSelector
	a.isBlockSelection = false
}

func (a *BattleClient) SwitchToBlockSelector() {
	a.selector = a.blockSelector
	a.isBlockSelection = true
}

func (a *BattleClient) GetNextUnit(unit *Unit) (*Unit, bool) {
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
func (a *BattleClient) SwitchToFirstPerson(position mgl32.Vec3) {
	a.captureMouse()
	a.fpsCamera.SetPosition(position)
	a.cameraIsFirstPerson = true
}

func (a *BattleClient) SwitchToIsoCamera() {
	a.freeMouse()
	a.cameraIsFirstPerson = false
}

func (a *BattleClient) SetConnection(connection *game.ServerConnection) {
	a.server = connection
	scheduleOnMainthread := func(msgType, data string) {
		println(fmt.Sprintf("[BattleClient] Received message %s", msgType))
		a.scheduleUpdate(func(deltaTime float64) {
			a.OnServerMessage(msgType, data)
		})
	}
	connection.SetEventHandler(scheduleOnMainthread)
}

func (a *BattleClient) OnServerMessage(msgType, messageAsJson string) {
	switch msgType {
	case "UnitMoved":
		var msg game.VisualUnitMoved
		if util.FromJson(messageAsJson, &msg) {
			a.OnUnitMoved(msg)
		}
	case "UnitLOSUpdated":
		var msg game.VisualUnitLOSUpdated
		if util.FromJson(messageAsJson, &msg) {
			a.OnUnitsLOSUpdate(msg)
		}
	case "ProjectileFired":
		var msg game.VisualProjectileFired
		if util.FromJson(messageAsJson, &msg) {
			a.OnProjectileFired(msg)
		}
	case "TargetedUnitActionResponse":
		var msg game.ActionResponse
		if util.FromJson(messageAsJson, &msg) {
			a.OnTargetedUnitActionResponse(msg)
		}
	case "NextPlayer":
		var msg game.NextPlayerMessage
		if util.FromJson(messageAsJson, &msg) {
			a.OnNextPlayer(msg)
		}
	}

}

func (a *BattleClient) OnTargetedUnitActionResponse(msg game.ActionResponse) {
	if !msg.Success {
		println(fmt.Sprintf("[BattleClient] Action failed: %s", msg.Message))
		a.Print(fmt.Sprintf("Action failed: %s", msg.Message))
	}
}
func (a *BattleClient) OnUnitsLOSUpdate(msg game.VisualUnitLOSUpdated) {
	changeLOS := func() {
		observer := a.unitMap[msg.Observer]
		for _, unit := range msg.Spotted {
			a.AddVisibleEnemyUnit(observer, unit)
		}
		for _, unit := range msg.Lost {
			a.RemoveVisibleEnemyUnit(observer, unit)
		}
	}
	observerPositionReached := func() bool {
		observer := a.unitMap[msg.Observer]
		return voxel.ToGridInt3(observer.GetFootPosition()) == msg.ObserverPosition
	}
	a.scheduleWaitForCondition(observerPositionReached, changeLOS)
}

func (a *BattleClient) OnProjectileFired(msg game.VisualProjectileFired) {
	// TODO: animate unit firing
	a.SpawnProjectile(msg.Origin, msg.Velocity)
}

func (a *BattleClient) OnUnitMoved(msg game.VisualUnitMoved) {
	unit, known := a.unitMap[msg.UnitID]
	if !known {
		println(fmt.Sprintf("[BattleClient] Unknown unit %d", msg.UnitID))
		return
	}
	println(fmt.Sprintf("[BattleClient] Moving %d to %v", msg.UnitID, msg.Path[len(msg.Path)-1]))

	a.voxelMap.ClearHighlights()
	a.unitSelector.Hide()

	unit.SetPath(msg.Path)
	unit.EndTurn()
}

func (a *BattleClient) OnNextPlayer(msg game.NextPlayerMessage) {
	println(fmt.Sprintf("[BattleClient] NextPlayer: %v", msg))
	if msg.YourTurn {
		a.ResetUnitsForNextTurn()
		a.Print("It's your turn!")
		a.SwitchToUnit(a.FirstUnit())
	} else {
		a.Print("Waiting for other players...")
		a.SwitchToWaitForEvents()
	}
}

func (a *BattleClient) EndTurn() {
	util.MustSend(a.server.EndTurn())
	a.SwitchToWaitForEvents()
}

func (a *BattleClient) SwitchToWaitForEvents() {
	a.stateStack = []GameState{&GameStateWaitForEvents{IsoMovementState{engine: a}}}
	a.state().Init(false)
}

func (a *BattleClient) FirstUnit() *Unit {
	for _, unit := range a.userUnits {
		if unit.CanAct() {
			return unit
		}
	}
	return nil
}

func (a *BattleClient) ResetUnitsForNextTurn() {
	for _, unit := range a.userUnits {
		unit.NextTurn()
	}
}
