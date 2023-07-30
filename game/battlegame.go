package game

import (
	"fmt"
	"github.com/faiface/mainthread"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/etxt"
	"github.com/memmaker/battleground/engine/glhf"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
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
	lastHitInfo           *RayCastHit
	units                 []*Unit
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
	factions              []*Faction
	currentFactionIndex   int
	currentVisibleEnemies map[*Unit][]*Unit
	projectileTexture     *glhf.Texture
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

	mainthread.Call(func() {
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
	})

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

func (a *BattleGame) SpawnUnit(spawnPos mgl32.Vec3, skinFile, name string) *Unit {
	//model := a.LoadModel("./assets/model.gltf")
	model := a.LoadModel("./assets/models/Guard3.glb")
	//model.SetTexture(0, util.MustLoadTexture("./assets/mc.fire.ice.png"))
	//model.SetTexture(0, util.MustLoadTexture("./assets/Agent_47.png"))
	model.SetTexture(0, util.MustLoadTexture(skinFile))
	model.SetAnimation("animation.walk")
	unit := NewUnit(model, spawnPos, name)
	unit.SetMap(a.voxelMap)
	a.collisionSolver.AddObject(unit)
	a.units = append(a.units, unit)
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

	for _, f := range a.updateQueue {
		f(elapsed)
	}
	a.updateQueue = a.updateQueue[:0]

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
	for i := len(a.units) - 1; i >= 0; i-- {
		actor := a.units[i]
		actor.Update(deltaTime)
		if actor.ShouldBeRemoved() {
			a.units = append(a.units[:i], a.units[i+1:]...)
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
	for _, unit := range a.CurrentFaction().units {
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
		for _, actor := range a.units {
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

	if a.crosshair != nil {
		a.crosshair.Draw()
	}
	a.guiShader.End()
}

func (a *BattleGame) SwitchToUnit(unit *Unit) {
	a.stateStack = []GameState{&GameStateUnit{engine: a, selectedUnit: unit}}
	a.state().Init(false)
}

func (a *BattleGame) SwitchToAction(unit *Unit, action Action) {
	a.stateStack = append(a.stateStack, &GameStateAction{engine: a, selectedUnit: unit, selectedAction: action})
	a.state().Init(false)
}

func (a *BattleGame) SwitchToFreeAim(unit *Unit, action *ActionAttack) {
	a.stateStack = append(a.stateStack, &GameStateFreeAim{engine: a, selectedUnit: unit, selectedAction: action})
	a.state().Init(false)
}

func (a *BattleGame) SwitchToEditMap() {
	a.stateStack = append(a.stateStack, &GameStateEditMap{engine: a})
	a.state().Init(false)
}

func (a *BattleGame) scheduleUpdate(f func(deltaTime float64)) {
	a.updateQueue = append(a.updateQueue, f)
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
	if hitInfo != nil {
		if hitInfo.UnitHit != nil {
			a.selector.SetPosition(util.ToGrid(hitInfo.UnitHit.GetFootPosition()))
		} else if hitInfo.Hit {
			if a.isBlockSelection {
				a.selector.SetPosition(hitInfo.PreviousGridPosition.ToVec3())
			} else {
				a.selector.SetPosition(a.voxelMap.GetGroundPosition(hitInfo.PreviousGridPosition).ToVec3())
			}
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
	}
	mesh.ConvertVertexData(shader)
	return groundSelector
}

func (a *BattleGame) SwitchToBlockSelector() {
	a.selector = a.blockSelector
	a.isBlockSelection = true
}

func (a *BattleGame) FirstTurn(factionName string) {
	for index, faction := range a.factions {
		if faction.name == factionName {
			a.currentFactionIndex = index
			a.NextTurn(faction)
		}
	}
}

func (a *BattleGame) NextTurn(faction *Faction) {
	a.UpdateVisibleEnemies()
	println(fmt.Sprintf("[Next Turn] Starting turn for %s", faction.name))
	for _, unit := range faction.units {
		if unit.IsDead() || unit.IsDying() {
			continue
		}
		unit.canAct = true
	}
	for _, unit := range faction.units {
		if unit.IsDead() || unit.IsDying() {
			continue
		}
		a.SwitchToUnit(unit)
		break
	}
}

func (a *BattleGame) EndTurn() {
	println(fmt.Sprintf("[End Turn] Ending turn for %s", a.CurrentFaction().name))
	a.currentFactionIndex = (a.currentFactionIndex + 1) % len(a.factions)
	a.NextTurn(a.CurrentFaction())
}

func (a *BattleGame) CurrentFaction() *Faction {
	return a.factions[a.currentFactionIndex]
}

func (a *BattleGame) GetNextUnit(unit *Unit) (*Unit, bool) {
	faction := a.CurrentFaction()
	for i, u := range faction.units {
		if u == unit {
			for j := 1; j < len(faction.units); j++ {
				nextUnitIndex := (i + j) % len(faction.units)
				nextUnit := faction.units[nextUnitIndex]
				if nextUnit.CanAct() {
					return nextUnit, true
				}
			}
			return nil, false
		}
	}
	return nil, false
}

func (a *BattleGame) UpdateVisibleEnemies() {
	own := a.CurrentFaction()
	visibleEnemies := make(map[*Unit][]*Unit)
	for _, ownUnit := range own.units {
		for _, unit := range a.units {
			if unit.faction == own {
				continue
			}
			if a.CanSee(ownUnit, unit) {
				if _, ok := visibleEnemies[ownUnit]; !ok {
					visibleEnemies[ownUnit] = make([]*Unit, 0)
				}
				visibleEnemies[ownUnit] = append(visibleEnemies[ownUnit], unit)
			}
		}
	}
	a.currentVisibleEnemies = visibleEnemies
}

func (a *BattleGame) CanSee(one *Unit, another *Unit) bool {
	source := one.GetEyePosition()
	targetOne := another.GetEyePosition()
	targetTwo := another.GetFootPosition()

	rayOne := a.RayCastUnits(source, targetOne, one, another)
	rayTwo := a.RayCastUnits(source, targetTwo, one, another)

	return rayOne.UnitHit == another || rayTwo.UnitHit == another
}

func (a *BattleGame) OnUnitMoved(unitMapObject voxel.MapObject, pos mgl32.Vec3, pos2 mgl32.Vec3) {
	unit := unitMapObject.(*Unit)
	own := a.CurrentFaction()
	if unit.faction == own {
		a.currentVisibleEnemies[unit] = make([]*Unit, 0)
		for _, enemy := range a.units {
			if enemy.faction == own {
				continue
			}
			if a.CanSee(unit, enemy) {
				a.currentVisibleEnemies[unit] = append(a.currentVisibleEnemies[unit], enemy)
			}
		}
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
