package game

import (
	"github.com/faiface/mainthread"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/etxt"
	"github.com/memmaker/battleground/engine/glhf"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
)

type BattleGame struct {
	*util.GlApplication
	mousePosX         float64
	mousePosY         float64
	lastMousePosX     float64
	lastMousePosY     float64
	voxelMap          *voxel.Map
	wireFrame         bool
	blockSelector     PositionDrawable
	crosshair         PositionDrawable
	modelShader       *glhf.Shader
	chunkShader       *glhf.Shader
	lineShader        *glhf.Shader
	guiShader         *glhf.Shader
	highlightShader   *glhf.Shader
	textLabel         *etxt.TextMesh
	textRenderer      *etxt.OpenGLTextRenderer
	lastHitInfo       *RayCastHit
	actors            []*Unit
	projectiles       []*Projectile
	debugObjects      []PositionDrawable
	blockTypeToPlace  byte
	camera            *util.ISOCamera
	drawBoundingBoxes bool
	showDebugInfo     bool
	timer             *util.Timer
	collisionSolver   *util.CollisionSolver
	state             GameState
	updateQueue       []func(deltaTime float64)
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
		camera:        util.NewISOCamera(width, height),
		timer:         util.NewTimer(),
	}
	myApp.modelShader = myApp.loadModelShader()
	myApp.chunkShader = myApp.loadChunkShader()
	myApp.highlightShader = myApp.loadHighlightShader()
	myApp.lineShader = myApp.loadLineShader()
	myApp.guiShader = myApp.loadGuiShader()
	myApp.textRenderer = etxt.NewOpenGLTextRenderer(myApp.guiShader)
	myApp.DrawFunc = myApp.Draw
	myApp.UpdateFunc = myApp.Update
	myApp.KeyHandler = myApp.handleKeyEvents
	myApp.MousePosHandler = myApp.handleMousePosEvents
	myApp.MouseButtonHandler = myApp.handleMouseButtonEvents
	myApp.ScrollHandler = myApp.handleScrollEvents
	mainthread.Call(func() {
		blockSelector := util.NewLineMesh(myApp.lineShader, [][2]mgl32.Vec3{
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
		myApp.textRenderer.SetAlign(etxt.YCenter, etxt.XCenter)
		crosshair := myApp.textRenderer.DrawText("+")
		crosshair.SetPosition(mgl32.Vec3{float32(glApp.WindowWidth) / 2, float32(glApp.WindowHeight) / 2, 0})
		myApp.textRenderer.SetAlign(etxt.Top, etxt.Left)

		myApp.SetBlockSelector(blockSelector)
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

type PositionDrawable interface {
	SetPosition(pos mgl32.Vec3)
	Draw()
}

func (a *BattleGame) LoadModel(filename string) *util.CompoundMesh {
	compoundMesh := util.LoadGLTF(filename)
	compoundMesh.ConvertVertexData(a.modelShader)
	return compoundMesh
}

func (a *BattleGame) SpawnUnit(spawnPos mgl32.Vec3) *Unit {
	//model := a.LoadModel("./assets/model.gltf")
	model := a.LoadModel("./assets/models/Guard2.glb")
	//model.SetTexture(0, util.MustLoadTexture("./assets/mc.fire.ice.png"))
	//model.SetTexture(0, util.MustLoadTexture("./assets/Agent_47.png"))
	model.SetTexture(0, util.MustLoadTexture("./assets/textures/skins/police_officer.png"))
	model.SetAnimation("animation.walk")
	unit := NewUnit(model, spawnPos, "Policeman")
	a.collisionSolver.AddObject(unit)
	a.voxelMap.MoveUnitTo(unit, spawnPos)
	a.actors = append(a.actors, unit)
	return unit
}

func (a *BattleGame) SpawnProjectile(pos, velocity mgl32.Vec3) {
	projectile := NewProjectile(a.modelShader, pos)
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
		a.state.OnDirectionKeys(elapsed, movementVector)
		a.updateDebugInfo()
	}
	a.updateActors(elapsed)
	//a.updateProjectiles(elapsed)
	a.collisionSolver.Update(elapsed)
	stopUpdateTimer()
}

func (a *BattleGame) updateActors(deltaTime float64) {
	for i := len(a.actors) - 1; i >= 0; i-- {
		actor := a.actors[i]
		actor.Update(deltaTime)
		if actor.ShouldBeRemoved() {
			a.actors = append(a.actors[:i], a.actors[i+1:]...)
		}
	}
}

func (a *BattleGame) Draw(elapsed float64) {
	stopDrawTimer := a.timer.Start("> Draw()")
	a.drawWorld(a.camera)

	a.drawModels()

	a.drawLines()

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

func (a *BattleGame) drawModels() {
	a.modelShader.Begin()

	a.modelShader.SetUniformAttr(1, a.camera.GetViewMatrix())

	for _, actor := range a.actors {
		actor.Draw(a.modelShader, a.camera.GetPosition())
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
func (a *BattleGame) drawLines() {
	a.lineShader.Begin()
	a.lineShader.SetUniformAttr(3, mgl32.Vec3{0, 0, 0})
	if a.blockSelector != nil && a.lastHitInfo != nil {
		a.lineShader.SetUniformAttr(0, a.camera.GetProjectionMatrix())
		a.lineShader.SetUniformAttr(1, a.camera.GetViewMatrix())
		a.blockSelector.Draw()
	}
	for _, drawable := range a.debugObjects {
		drawable.Draw()
	}
	if a.drawBoundingBoxes { // TODO: SOMETHING IS VERY WRONG WITH DRAWING BOUNDING BOXES
		a.lineShader.SetUniformAttr(2, mgl32.Ident4())
		a.lineShader.SetUniformAttr(3, mgl32.Vec3{0.1, 1.0, 0.1})
		//a.player.GetAABB().Draw(a.lineShader)
		for _, actor := range a.actors {
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
	a.state = &GameStateUnit{engine: a, selectedUnit: unit}
	a.state.Init()
}

func (a *BattleGame) SwitchToAction(unit *Unit, action Action) {
	a.state = &GameStateAction{engine: a, selectedUnit: unit, selectedAction: action}
	a.state.Init()
}

func (a *BattleGame) SwitchToEditMap() {
	a.state = &GameStateEditMap{engine: a}
	a.state.Init()
}

func (a *BattleGame) scheduleUpdate(f func(deltaTime float64)) {
	a.updateQueue = append(a.updateQueue, f)
}
