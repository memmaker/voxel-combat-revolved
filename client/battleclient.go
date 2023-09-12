package client

import (
    "fmt"
    "github.com/go-gl/gl/v4.1-core/gl"
    "github.com/go-gl/mathgl/mgl32"
    "github.com/memmaker/battleground/engine/glhf"
    "github.com/memmaker/battleground/engine/gui"
    "github.com/memmaker/battleground/engine/util"
    "github.com/memmaker/battleground/engine/voxel"
    "github.com/memmaker/battleground/game"
    "os"
    "strings"

)

type BattleClient struct {
    *util.GlApplication
    *game.GameClient[*Unit]
    lastMousePosX              float64
    lastMousePosY              float64
    wireFrame                  bool
    selector                   PositionDrawable
    crosshair                  *Crosshair
    chunkShader                *glhf.Shader
    lineShader                 *glhf.Shader
    guiShader                  *glhf.Shader
    defaultShader              *glhf.Shader
    textLabel                  *util.BitmapFontMesh
    bigLabel                   *util.BitmapFontMesh
    lastHitInfo                *game.RayCastHit
    projectiles                []*Projectile
    debugObjects               []PositionDrawable
    isoCamera                  *util.ISOCamera
    fpsCamera                  *util.FPSCamera
    cameraIsFirstPerson        bool
    drawBoundingBoxes          bool
    showDebugInfo              bool
    timer                      *util.Timer
    stateStack                 []GameState
    conditionQueue             []ConditionalCall
    isBlockSelection           bool
    groundSelector             *GroundSelector
    unitSelector               *GroundSelector
    blockSelector              *util.LineMesh
    projectileTexture          *glhf.Texture
    actionbar                  *gui.ActionBar
    highlights                 *voxel.Highlights
    lines                      *LineDrawer
    server                     *game.ServerConnection
    serverChannel              chan game.StringMessage
    onSwitchToIsoCamera        func()
    bulletModel                *util.CompoundMesh
    settings                   ClientSettings
    guiIcons                   map[string]byte
    cameraAnimation            *util.CameraAnimation
    overwatchPositionsThisTurn []voxel.Int3
    selectedUnit               *Unit
    aspectRatio                float32
    debugPositions             []voxel.Int3
    oneShotParticles           *glhf.ParticleSystem
    smoker                     *Smoker
    particleProps              map[ParticleName]glhf.ParticleProperties
    selectedBlocks             []voxel.Int3
    currentlyMovingUnits       bool
}

func (a *BattleClient) state() GameState {
    return a.stateStack[len(a.stateStack)-1]
}
func (a *BattleClient) isBusyMovingUnits() bool {
    for _, unit := range a.GetAllClientUnits() {
        if unit.state.GetName() == UnitGotoWaypoint {
            return true
        }
    }
    return false
}
func (a *BattleClient) IsOccludingBlock(x, y, z int) bool {
    if a.GetVoxelMap().IsSolidBlockAt(int32(x), int32(y), int32(z)) {
        return !a.GetVoxelMap().GetGlobalBlock(int32(x), int32(y), int32(z)).IsAir()
    }
    return false
}

type ClientSettings struct {
    Width                     int
    Height                    int
    ISOCameraScrollZoomSpeed  float32
    FPSCameraInvertedMouse    bool
    EnableBulletCam           bool
    FPSCameraMouseSensitivity float32
    EnableCameraAnimations    bool
    FullScreen                bool
    Title                     string
}

func NewClientSettingsFromFile(filename string) ClientSettings {
    if util.DoesFileExist(filename) {
        var settings ClientSettings
        file, _ := os.ReadFile(filename)
        if util.FromJson(string(file), &settings) {
            return settings
        }
    }
    return ClientSettings{ // default settings
        Width:                  800,
        Height:                 600,
        FPSCameraInvertedMouse: true,
    }
}

type ParticleName int

const (
    ParticlesBlood ParticleName = iota
    ParticlesBulletImpact
    ParticlesExplosion
    ParticlesSmoke
)

func NewBattleGame(con *game.ServerConnection, initInfos game.GameStartedMessage, settings ClientSettings) *BattleClient {
    window, terminateFunc := util.InitOpenGLWindow(settings.Title, settings.Width, settings.Height, settings.FullScreen)
    usedWidth, usedHeight := window.GetSize()
    glApp := &util.GlApplication{
        WindowWidth:  usedWidth,
        WindowHeight: usedHeight,
        Window:        window,
        TerminateFunc: terminateFunc,
    }
    window.SetKeyCallback(glApp.KeyCallback)
    window.SetCursorPosCallback(glApp.MousePosCallback)
    window.SetMouseButtonCallback(glApp.MouseButtonCallback)
    window.SetScrollCallback(glApp.ScrollCallback)

    fpsCamera := util.NewFPSCamera(mgl32.Vec3{0, 10, 0}, usedWidth, usedHeight, settings.FPSCameraMouseSensitivity)
    fpsCamera.SetInvertedY(settings.FPSCameraInvertedMouse)

    myApp := &BattleClient{
        GlApplication: glApp,
        isoCamera: util.NewISOCamera(usedWidth, usedHeight),
        fpsCamera:     fpsCamera,
        timer:         util.NewTimer(),
        settings:      settings,
        particleProps: map[ParticleName]glhf.ParticleProperties{
            ParticlesBlood: {
                PositionVariation:    mgl32.Vec3{0.5, 0.5, 0.5},
                VelocityFromPosition: func(origin, pos mgl32.Vec3) mgl32.Vec3 { return mgl32.Vec3{0, 0, 0} },
                VelocityVariation:    mgl32.Vec3{0.1, 0.1, 0.1},
                SizeBegin:            0.1,
                SizeVariation:        0.05,
                SizeEnd:              0.05,
                Lifetime:             0.2,
                ColorBegin: mgl32.Vec3{0.7, 0.01, 0.01},
                ColorEnd:   mgl32.Vec3{0.4, 0.01, 0.01},
            },
            ParticlesBulletImpact: {
                PositionVariation:    mgl32.Vec3{0.5, 0.5, 0.5},
                VelocityFromPosition: func(origin, pos mgl32.Vec3) mgl32.Vec3 { return mgl32.Vec3{0, 0, 0} },
                VelocityVariation:    mgl32.Vec3{0.1, 0.1, 0.1},
                SizeBegin:            0.08,
                SizeVariation:        0.04,
                SizeEnd:              0.04,
                Lifetime:             0.2,
                ColorBegin: mgl32.Vec3{0.7, 0.7, 0.7},
                ColorEnd:   mgl32.Vec3{0.01, 0.01, 0.01},
            },
            ParticlesExplosion: {
                PositionVariation: mgl32.Vec3{0.02, 0.02, 0.02},
                VelocityFromPosition: func(origin, pos mgl32.Vec3) mgl32.Vec3 {
                    return pos.Sub(origin).Normalize().Mul(20)
                },
                VelocityVariation: mgl32.Vec3{0.1, 0.1, 0.1},
                SizeBegin:         0.1,
                SizeVariation:     0.05,
                SizeEnd:           0.5,
                Lifetime:          0.5,
                ColorBegin: mgl32.Vec3{1, 1, 1},
                ColorEnd:   mgl32.Vec3{0.01, 0.01, 0.01},
            },
        },
        aspectRatio: float32(usedWidth) / float32(usedHeight),
    }
    myApp.GameClient = game.NewGameClient[*Unit](initInfos, myApp.CreateClientUnit)
    myApp.GameClient.SetEnvironment("GL-Client")
    myApp.GameClient.SetOnExplode(myApp.OnExplode)
    myApp.GameClient.SetOnNotification(myApp.Print)
    myApp.chunkShader = myApp.loadChunkShader()
    myApp.lineShader = myApp.loadLineShader()
    myApp.guiShader = myApp.loadGuiShader()
    myApp.defaultShader = myApp.loadDefaultShader()
    myApp.projectileTexture = glhf.NewSolidColorTexture([3]uint8{255, 12, 255})
    myApp.DrawFunc = myApp.Draw
    myApp.UpdateFunc = myApp.Update
    myApp.KeyHandler = myApp.handleKeyEvents
    myApp.MousePosHandler = myApp.handleMousePosEvents
    myApp.MouseButtonHandler = myApp.handleMouseButtonEvents
    myApp.ScrollHandler = myApp.handleScrollEvents
    myApp.ResizeHandler = myApp.handleResizeEvents

    //fontTextureAtlas, atlasIndex := util.CreateAtlasFromPBMs("./assets/fonts/quadratica/", 8, 14)
    //fontTextureAtlas.SaveAsPNG("./assets/fonts/quadratica.png")

    assetLoader := myApp.GameInstance.GetAssets()

    bigFont := assetLoader.LoadBitmapFontWithoutIndex("STEEL_MV", 32, 32)
    bigFont.SetPaddingBetweenItems(0, 1)
    bigFontIndex := util.NewIndexFromDescription(util.AtlasDescription{
        PositionOfCapitalA:     [2]int{2, 3},
        PositionOfZero:         &[2]int{5, 1},
        PositionOfSpecialChain: &[2]int{0, 0},
        Cols:                   10,
        SpecialCharacterChain:  []rune{'!', 'Ö', 'Ä', '"', '§', '*', '(', ')', '+', ',', ' ', '-', '.', '/'},
    })

    /*
       bigFont := assetLoader.LoadBitmapFontWithoutIndex("CRYPGRAU", 32, 38)
       bigFontIndex := util.NewIndexFromDescription(util.AtlasDescription{
           PositionOfCapitalA:    [2]int{0, 0},
           Cols:                  10,
           SpecialCharacterChain: []rune{':', ';', '.', '-', ',', '?', '´', '(', ')', '/', '!', '*', '='},
       })

    */
    myApp.bigLabel = util.NewBitmapFontMesh(myApp.guiShader, bigFont, bigFontIndex.GetMapper())
    myApp.bigLabel.SetCenterOrigin(true)
    myApp.bigLabel.SetScale(1)
    myApp.bigLabel.SetDiscardColor(mgl32.Vec3{0, 0, 0})
    myApp.bigLabel.SetPosition(mgl32.Vec2{float32(myApp.WindowWidth) / 2, float32(myApp.WindowHeight) / 2})

    fontTextureAtlas, atlasIndex := assetLoader.LoadBitmapFont("quadratica", 8, 14)
    myApp.textLabel = util.NewBitmapFontMesh(myApp.guiShader, fontTextureAtlas, atlasIndex.GetMapper())
    myApp.textLabel.SetScale(1)
    myApp.textLabel.SetTintColor(ColorTechTeal.Vec3())

    myApp.unitSelector = NewGroundSelector(assetLoader.LoadMesh("flatselector"), myApp.defaultShader)

    selectorMesh := assetLoader.LoadMeshWithColor("selector", mgl32.Vec3{1, 1, 1})
    selectorMesh.SetTexture(0, glhf.NewSolidColorTexture([3]uint8{0, 248, 250}))
    myApp.groundSelector = NewGroundSelector(selectorMesh, myApp.defaultShader)

    myApp.blockSelector = NewBlockSelector(myApp.lineShader)

    myApp.bulletModel = assetLoader.LoadMesh("bullet")
    myApp.bulletModel.UploadVertexData(myApp.defaultShader)

    guiAtlas, guiIconIndices := util.CreateFixed256PxAtlasFromDirectory("./assets/gui", []string{"walk", "ranged", "reticule", "next-turn", "reload", "grenade", "overwatch", "shield"})
    myApp.guiIcons = guiIconIndices
    myApp.actionbar = gui.NewActionBar(myApp.guiShader, guiAtlas, glApp.WindowWidth, glApp.WindowHeight, 64, 64)

    myApp.highlights = voxel.NewHighlights(myApp.defaultShader)
    myApp.lines = NewLineDrawer(myApp.defaultShader)

    getViewFunc := func() mgl32.Mat4 { return myApp.camera().GetViewMatrix() }
    getProjectionFunc := func() mgl32.Mat4 { return myApp.camera().GetProjectionMatrix() }
    myApp.oneShotParticles = glhf.NewParticleSystem(5000, loadTransformFeedbackShader(getParticleVertexFormat()), loadParticleShader(getParticleVertexFormat()), getViewFunc, getProjectionFunc)
    myApp.smoker = NewSmoker(myApp.GetVoxelMap, glhf.NewParticleSystem(20000, loadTransformFeedbackShader(getParticleVertexFormat()), loadParticleShader(getParticleVertexFormat()), getViewFunc, getProjectionFunc))
    myApp.SwitchToBlockSelector()

    crosshair := NewCrosshair(myApp.defaultShader, myApp.fpsCamera)
    myApp.SetCrosshair(crosshair)

    myApp.SetConnection(con)

    return myApp
}

func (a *BattleClient) CreateClientUnit(currentUnit *game.UnitInstance) *Unit {
    // load model
    model := a.GetAssets().LoadMesh(currentUnit.Definition.ModelFile)
    // upload vertex data to GPU
    model.UploadVertexData(a.defaultShader)
    // select weapon model by hiding the others
    model.HideChildrenOfBoneExcept("Weapon", currentUnit.GetWeapon().Definition.Model)
    // load & set the skin texture
    if currentUnit.Definition.ClientRepresentation.TextureFile != "" {
        model.SetTexture(0, a.GetAssets().LoadSkin(currentUnit.Definition.ClientRepresentation.TextureFile))
    }
    currentUnit.SetModel(model)
    unit := NewClientUnit(currentUnit)
    util.LogGlobalUnitDebug(unit.DebugString("CreateClientUnit"))
    return unit
}

func (a *BattleClient) SpawnProjectile(pos, velocity, destination mgl32.Vec3, onArrival func()) *Projectile {
    projectile := NewProjectile(a.defaultShader, a.bulletModel, pos, velocity)
    projectile.SetDestination(destination)
    projectile.SetOnArrival(onArrival)
    a.projectiles = append(a.projectiles, projectile)
    //println(fmt.Sprintf("\n>> Projectile spawned at %v with destination %v", pos, destination))
    return projectile
}

func (a *BattleClient) SetCrosshair(crosshair *Crosshair) {
    a.crosshair = crosshair
}

func (a *BattleClient) Print(text string) {
    // split text into lines
    // to lower
    lines := strings.Split(text, "\n")
    a.textLabel.SetMultilineText(lines)
}

func (a *BattleClient) Update(elapsed float64) {
    stopUpdateTimer := a.timer.Start("> Update()")

    /*
       if a.coroutine.Running() {
           // While the coroutine runs, we call Coroutine.Update(). This allows
           // the coroutine to execute, but also gives control back to the main
           // thread when it's yielding so we can do other stuff, like take input
           // or update a game's screen.
           a.coroutine.Update()
       }

    */
    //properties := a.particleProps[ParticlesBlood].WithOrigin(a.groundSelector.GetPosition())
    //a.oneShotParticles.Emit(properties, 1)
    a.GetVoxelMap().Update(elapsed)

    waitForCameraAnimation := a.handleCameraAnimation(elapsed)
    a.currentlyMovingUnits = a.isBusyMovingUnits()

    if !a.currentlyMovingUnits && !waitForCameraAnimation { // PROBLEM: isBusy can hang. So isn't reset every time we expect it to..
        a.pollNetwork()
    }

    if len(a.conditionQueue) > 0 {
        for i := len(a.conditionQueue) - 1; i >= 0; i-- {
            c := a.conditionQueue[i]
            if c.condition(elapsed) {
                c.function(elapsed)
                a.conditionQueue = append(a.conditionQueue[:i], a.conditionQueue[i+1:]...)
            }
        }
    }

    camMoved, movementVector := a.pollInput(elapsed)
    if camMoved {
        a.state().OnDirectionKeys(elapsed, movementVector)
    }
    a.updateProjectiles(elapsed)
    a.updateUnits(elapsed)
    a.smoker.Update(elapsed)

    a.updateDebugInfo()
    stopUpdateTimer()

}

func (a *BattleClient) updateProjectiles(elapsed float64) {
    for i := len(a.projectiles) - 1; i >= 0; i-- {
        projectile := a.projectiles[i]
        projectile.Update(elapsed)
        blockPos := projectile.GetBlockPosition()
        a.smoker.ClearSmokeAt(blockPos) // TODO: not hacky solution

        if projectile.IsDead() {
            a.projectiles = append(a.projectiles[:i], a.projectiles[i+1:]...)
        }
    }
}

func (a *BattleClient) updateUnits(deltaTime float64) {
    allUnits := a.GetAllClientUnits()
    for _, unit := range allUnits {
        unit.Update(deltaTime)
    }
}
func (a *BattleClient) camera() util.Camera {
    var camera util.Camera = a.isoCamera
    if a.cameraIsFirstPerson {
        camera = a.fpsCamera
    }
    if a.cameraAnimation != nil {
        camera = a.cameraAnimation
    }
    return camera
}
func (a *BattleClient) Draw(elapsed float64) {
    stopDrawTimer := a.timer.Start("> Draw()")

    a.drawWorld(a.camera())

    a.drawLines(a.camera())

    a.oneShotParticles.Draw(elapsed)

    a.smoker.Draw(elapsed)

    a.drawDefaultShader(a.camera())

    a.drawGUI()

    stopDrawTimer()
}

func (a *BattleClient) drawWorld(cam util.Camera) {
    a.chunkShader.Begin()

    a.chunkShader.SetUniformAttr(0, cam.GetProjectionViewMatrix())

    a.GetVoxelMap().Draw(1, cam.GetFrustumPlanes())

    a.chunkShader.End()
}

func (a *BattleClient) drawDefaultShader(cam util.Camera) {
    a.defaultShader.Begin()

    a.defaultShader.SetUniformAttr(ShaderViewMatrix, cam.GetViewMatrix())
    a.defaultShader.SetUniformAttr(ShaderProjectionMatrix, cam.GetProjectionMatrix())
    a.defaultShader.SetUniformAttr(ShaderDrawMode, ShaderDrawTexturedQuads)

    for _, unit := range a.GetAllClientUnits() { // TODO: view frustum culling
        if a.UnitIsVisibleToPlayer(a.GetControllingUserID(), unit.UnitID()) {
            unit.Draw(a.defaultShader)

        }
    }

    a.drawProjectiles()

    if a.selector != nil && a.lastHitInfo != nil && !a.isBlockSelection {
        a.defaultShader.SetUniformAttr(ShaderDrawMode, ShaderDrawColoredQuads)
        a.defaultShader.SetUniformAttr(ShaderDrawColor, ColorTechTeal)
        a.selector.Draw()
    }

    if a.lines != nil && !a.currentlyMovingUnits {
        gl.Disable(gl.CULL_FACE)
        a.defaultShader.SetUniformAttr(ShaderDrawColor, a.lines.GetColor())
        a.defaultShader.SetUniformAttr(ShaderModelMatrix, mgl32.Ident4())
        a.defaultShader.SetUniformAttr(ShaderViewport, mgl32.Vec2{float32(a.WindowWidth), float32(a.WindowHeight)})
        a.defaultShader.SetUniformAttr(ShaderDrawMode, ShaderDrawLine)
        a.defaultShader.SetUniformAttr(ShaderThickness, float32(2))
        a.lines.Draw()
        gl.Enable(gl.CULL_FACE)
    }

    if a.unitSelector != nil {
        a.defaultShader.SetUniformAttr(ShaderDrawMode, ShaderDrawCircle)
        a.defaultShader.SetUniformAttr(ShaderDrawColor, ColorTechTeal)
        a.defaultShader.SetUniformAttr(ShaderThickness, float32(0.2))
        a.unitSelector.Draw()
    }

    if a.cameraIsFirstPerson && a.crosshair != nil && !a.crosshair.IsHidden() {
        gl.DepthMask(false)
        a.crosshair.Draw()
        gl.DepthMask(true)
    } else {
        a.defaultShader.SetUniformAttr(ShaderModelMatrix, mgl32.Ident4())
        a.defaultShader.SetUniformAttr(ShaderDrawMode, ShaderDrawColoredQuads)
        a.defaultShader.SetUniformAttr(ShaderDrawColor, a.highlights.GetTintColor())
        a.highlights.Draw(ShaderDrawMode, ShaderDrawColoredFadingQuads)
    }
    a.defaultShader.End()
}

func (a *BattleClient) drawProjectiles() {
    for i := len(a.projectiles) - 1; i >= 0; i-- {
        projectile := a.projectiles[i]
        projectile.Draw()
    }
}
func (a *BattleClient) drawLines(cam util.Camera) {
    a.lineShader.Begin()
    a.lineShader.SetUniformAttr(0, cam.GetProjectionMatrix())
    a.lineShader.SetUniformAttr(1, cam.GetViewMatrix())
    if a.selector != nil && a.lastHitInfo != nil && a.isBlockSelection {
        a.lineShader.SetUniformAttr(3, mgl32.Vec3{1, 1, 1})
        a.selector.Draw()
    }

    a.lineShader.SetUniformAttr(3, mgl32.Vec3{0.2, 1, 0.2})
    for _, debugPositions := range a.debugPositions {
        a.blockSelector.DrawAt(debugPositions)
    }
    for _, selectedPos := range a.selectedBlocks {
        a.blockSelector.DrawAt(selectedPos)
    }
    a.lineShader.End()
}

func (a *BattleClient) drawGUI() {
    a.guiShader.Begin()
    if a.textLabel != nil {
        a.textLabel.Draw()
    }

    if a.bigLabel != nil {
        a.bigLabel.Draw()
    }

    if a.actionbar != nil {
        a.guiShader.SetUniformAttr(2, mgl32.Vec4{0, 0, 0, 0}) // tint
        a.guiShader.SetUniformAttr(3, mgl32.Vec4{0, 0, 0, 0}) // discard
        a.actionbar.Draw()
    }

    a.guiShader.End()
}

func (a *BattleClient) SwitchToNextUnit(currentUnit *Unit) {
    nextUnit, hasNext := a.GetNextUnit(currentUnit)
    if !hasNext {
        a.SwitchToUnit(currentUnit)
        return
    }
    a.SwitchToUnit(nextUnit)
}

func (a *BattleClient) SwitchToUnit(unit *Unit) {
    a.stateStack = []GameState{NewGameStateUnit(a, unit)}
    a.state().Init(false)
}

func (a *BattleClient) SwitchToUnitNoCameraMovement(unit *Unit) {
    a.stateStack = []GameState{NewGameStateUnitNoCamMove(a, unit)}
    a.state().Init(false)
}

func (a *BattleClient) SwitchToAction(unit *Unit, action game.TargetAction) {
    a.stateStack = []GameState{
        NewGameStateUnit(a, unit),
        &GameStateAction{IsoMovementState: IsoMovementState{engine: a}, selectedUnit: unit, selectedAction: action},
    }
    a.state().Init(false)
}

func (a *BattleClient) SwitchToFreeAim(unit *Unit, action *game.ActionSnapShot) {
    a.stateStack = []GameState{
        NewGameStateUnit(a, unit),
        &GameStateFreeAim{engine: a, selectedAction: action, lockedTarget: -1},
    }
    a.state().Init(false)
}

func (a *BattleClient) SwitchToEditMap() {
    a.stateStack = append(a.stateStack, NewEditorState(a))
    a.state().Init(false)
}

func (a *BattleClient) SwitchToDeployment() {
    a.stateStack = []GameState{
        &GameStateWaitForEvents{IsoMovementState{engine: a}},
        NewGameStateDeployment(a),
    }
    a.state().Init(false)
}

func (a *BattleClient) SwitchToWaitForEvents() {
    a.stateStack = []GameState{&GameStateWaitForEvents{IsoMovementState{engine: a}}}
    a.state().Init(false)
}

func (a *BattleClient) scheduleUpdate(f func(deltaTime float64)) {
    a.scheduleWaitForCondition(func(deltaTime float64) bool { return true }, f)
}

func (a *BattleClient) scheduleUpdateIn(timeInSeconds float64, f func(deltaTime float64)) {
    timeSpent := float64(0)
    a.scheduleWaitForCondition(func(deltaTime float64) bool {
        timeSpent += deltaTime
        return timeSpent >= timeInSeconds
    }, f)
}

type ConditionalCall struct {
    condition func(deltaTime float64) bool
    function  func(deltaTime float64)
}

func (a *BattleClient) scheduleWaitForCondition(condition func(deltaTime float64) bool, function func(deltaTime float64)) {
    //a.mc.Lock()
    a.conditionQueue = append(a.conditionQueue, ConditionalCall{condition: condition, function: function})
    //a.mc.Unlock()
}

func (a *BattleClient) PopState() {
    if len(a.stateStack) > 1 {
        a.stateStack = a.stateStack[:len(a.stateStack)-1]
    }
    a.state().Init(true)
}
func (a *BattleClient) UpdateMousePicking(newX, newY float64) {
    rayStart, rayEnd := a.camera().GetPickingRayFromScreenPosition(newX, newY)

    hitInfo := a.RayCastGround(rayStart, rayEnd)

    if !hitInfo.Hit {
        return
    }

    if a.isBlockSelection {
        a.selector.SetBlockPosition(hitInfo.PreviousGridPosition)
    } else {
        a.selector.SetBlockPosition(a.GetVoxelMap().GetGroundPosition(hitInfo.PreviousGridPosition))
    }

    selectedPosition := a.selector.GetBlockPosition()
    if !a.GetVoxelMap().IsOccupied(selectedPosition) {
        return
    }
    unitHitInstance := a.GetVoxelMap().GetMapObjectAt(selectedPosition).(*game.UnitInstance)

    unitHit, _ := a.GetClientUnit(unitHitInstance.UnitID())
    pressureString := ""
    pressure := a.GetTotalPressure(unitHit.UnitID())
    if pressure > 0 {
        pressureString = fmt.Sprintf("\nPressure: %0.2f", pressure)
    }
    if unitHit.IsUserControlled() {
        a.textLabel.SetTintColor(ColorPositiveGreen.Vec3())
        a.Print(unitHit.GetFriendlyDescription() + pressureString)
    } else {
        a.textLabel.SetTintColor(ColorNegativeRed.Vec3())
        a.Print(unitHit.GetEnemyDescription() + pressureString)
    }
}

func (a *BattleClient) SwitchToGroundSelector() {
    a.selector = a.groundSelector
    a.isBlockSelection = false
}

func (a *BattleClient) SwitchToBlockSelector() {
    a.selector = a.blockSelector
    a.isBlockSelection = true
    a.unitSelector.Hide()
}

func (a *BattleClient) GetNextUnit(currentUnit *Unit) (*Unit, bool) {
    units := a.GetMyUnits()
    for i, u := range units {
        if u.UnitID() == currentUnit.UnitID() {
            for j := 1; j < len(units); j++ {
                nextUnitIndex := (i + j) % len(units)
                nextUnit := units[nextUnitIndex]
                if nextUnit.CanAct() {
                    unit, _ := a.GetClientUnit(nextUnit.UnitID())
                    return unit, true
                }
            }
            return nil, false
        }
    }
    return nil, false
}
func (a *BattleClient) SwitchToUnitFirstPerson(unit *Unit, lookAtTarget mgl32.Vec3, accuracy float64) {
    position := unit.GetEyePosition()
    a.fpsCamera.SetPosition(position)
    a.fpsCamera.SetLookTarget(lookAtTarget)

    a.SwitchToFirstPerson()

    a.crosshair.SetHidden(false)
    a.crosshair.SetSize(1.0 - accuracy)

    // attach arms of selected unit to camera
    arms, _ := unit.GetModel().GetNodeByName("Arms")
    if arms != nil {
        previousAnimation := unit.GetModel().GetAnimationName()
        arms.SetTempParent(a.fpsCamera)
        arms.SetAnimation(game.AnimationWeaponIdle.Str())
        a.onSwitchToIsoCamera = func() { // detach arms when switching back to iso camera
            arms.SetTempParent(nil)
            unit.GetModel().SetAnimation(previousAnimation, 1.0)
        }
    }
}

func (a *BattleClient) SwitchToFirstPerson() {
    a.highlights.Hide()
    a.groundSelector.Hide()
    a.actionbar.Hide()
    a.lines.Hide()

    a.captureMouse()

    a.fpsCamera.ResetFOV()

    startCam := a.isoCamera
    endCam := a.fpsCamera
    a.StartCameraTransition(startCam, endCam, 0.5)

    a.cameraIsFirstPerson = true
    a.freezeIdleAnimations()
}

func (a *BattleClient) SwitchToIsoCamera() {
    a.freeMouse()
    a.cameraIsFirstPerson = false
    a.resumeIdleAnimations()
    a.onSwitchToISO()
}

func (a *BattleClient) onSwitchToISO() {
    if a.onSwitchToIsoCamera != nil {
        a.onSwitchToIsoCamera()
        a.onSwitchToIsoCamera = nil
    }
}

func (a *BattleClient) SetConnection(connection *game.ServerConnection) {
    a.server = connection
    a.serverChannel = make(chan game.StringMessage, 100)
    connection.SetMainthreadChannel(a.serverChannel)
}

func (a *BattleClient) OnTargetedUnitActionResponse(msg game.ActionResponse) {
    if !msg.Success {
        println(fmt.Sprintf("[BattleClient] Action failed: %s", msg.Message))
        a.Print(fmt.Sprintf("Action failed: %s", msg.Message))
    }
}

func (a *BattleClient) OnServerMessage(msgType, messageAsJson string) {
    util.LogNetworkDebug(fmt.Sprintf("\n[GL-Client] FROM Server msg(%s):\n%s\n", msgType, messageAsJson))
    switch msgType {
    case "StartDeployment":
        //var msg game.StartDeploymentMessage
        a.OnStartDeployment()
    case "BeginOverwatch":
        var msg game.VisualBeginOverwatch
        if util.FromJson(messageAsJson, &msg) {
            a.OnBeginOverwatch(msg)
        }
    case "OwnUnitMoved":
        var msg game.VisualOwnUnitMoved
        if util.FromJson(messageAsJson, &msg) {
            a.OnOwnUnitMoved(msg)
        }
    case "EnemyUnitMoved":
        var msg game.VisualEnemyUnitMoved
        if util.FromJson(messageAsJson, &msg) {
            a.OnEnemyUnitMoved(msg)
        }
    case "RangedAttack":
        var msg game.VisualRangedAttack
        if util.FromJson(messageAsJson, &msg) {
            a.OnRangedAttack(msg)
        }
    case "ActionResponse":
        var msg game.ActionResponse
        if util.FromJson(messageAsJson, &msg) {
            a.OnTargetedUnitActionResponse(msg)
        }
    case "NextPlayer":
        var msg game.NextPlayerMessage
        if util.FromJson(messageAsJson, &msg) {
            a.OnNextPlayer(msg)
        }
    case "GameOver":
        var msg game.GameOverMessage
        if util.FromJson(messageAsJson, &msg) {
            a.OnGameOver(msg)
        }
    case "Reload":
        var msg game.UnitMessage
        if util.FromJson(messageAsJson, &msg) {
            a.OnReload(msg)
        }
    case "DebugResponse":
        var msg game.CompleteGameState
        if util.FromJson(messageAsJson, &msg) {
            a.OnDebugGameStateRececeivedFromServer(msg)
        }
    }
    a.state().OnServerMessage(msgType, messageAsJson)
}

func (a *BattleClient) OnBeginOverwatch(msg game.VisualBeginOverwatch) {
    unit, known := a.GetClientUnit(msg.Watcher)
    if known && a.IsMyUnit(msg.Watcher) {
        a.highlights.ClearFlat(voxel.HighlightTarget)
        a.highlights.ClearAndUpdateFlat(voxel.HighlightMove)
        a.GameClient.OnBeginOverwatch(msg)
        a.SwitchToNextUnit(unit)

        a.overwatchPositionsThisTurn = append(a.overwatchPositionsThisTurn, msg.WatchedLocations...)
        a.updateOverwatchHighlights()
    }
}

func (a *BattleClient) OnReload(msg game.UnitMessage) {
    unit, _ := a.GetClientUnit(msg.UnitID())
    unit.Reload()
    a.Print(fmt.Sprintf("%s reloaded the %s.", unit.GetName(), unit.GetWeapon().Definition.UniqueName))
    a.UpdateActionbarFor(unit)
}

func (a *BattleClient) OnRangedAttack(msg game.VisualRangedAttack) {
    // TODO: animate unit firing
    damageReport := ""

    attacker, knownAttacker := a.GetClientUnit(msg.Attacker)
    var attackerUnit *game.UnitInstance
    if knownAttacker {
        attackerUnit = attacker.UnitInstance
        attackerUnit.SetForward(voxel.DirectionToGridInt3(msg.AimDirection))
        attackerUnit.UpdateMapPosition()
        attackerUnit.GetWeapon().ConsumeAmmo(msg.AmmoCost)
        attackerUnit.ConsumeAP(msg.APCostForAttacker)
        attackerUnit.GetModel().SetAnimationLoop(game.AnimationWeaponIdle.Str(), 1.0)
        if msg.IsTurnEnding {
            attackerUnit.EndTurn()
        }
    }
    attackerIsOwnUnit := knownAttacker && a.IsMyUnit(attacker.UnitID())
    activateBulletCam := len(msg.Projectiles) == 1 && attackerIsOwnUnit && a.settings.EnableBulletCam //only for single projectiles and own units

    projectileArrivalCounter := 0
    for index, p := range msg.Projectiles {
        projectile := p
        firedProjectile := a.SpawnProjectile(projectile.Origin, projectile.Velocity, projectile.Destination, func() {
            // on arrival
            projectileArrivalCounter++
            if projectile.UnitHit >= 0 {
                unit, ok := a.GetClientUnit(uint64(projectile.UnitHit))
                if !ok {
                    println(fmt.Sprintf("[BattleClient] Projectile hit unit %d, but unit not found", projectile.UnitHit))
                    return
                }
                isLethal := a.ApplyDamage(attackerUnit, unit.UnitInstance, projectile.Damage, projectile.BodyPart)
                if isLethal {
                    unit.PlayDeathAnimation(projectile.Velocity, projectile.BodyPart)
                } else {
                    unit.PlayHitAnimation(projectile.Velocity, projectile.BodyPart)
                }

                a.AddBlood(unit, projectile.Destination, projectile.Velocity, projectile.BodyPart)

                println(fmt.Sprintf("[BattleClient] Projectile hit unit %s(%d)", unit.GetName(), unit.UnitID()))
            } else {
                a.AddBulletImpact(projectile.Destination, projectile.Velocity)
            }

            for _, damagedBlock := range projectile.BlocksHit {
                blockDef := a.GetBlockDefAt(damagedBlock)
                blockDef.OnDamageReceived(damagedBlock, projectile.Damage)
            }

            if projectileArrivalCounter == len(msg.Projectiles) {
                a.Print(damageReport)
                if activateBulletCam {
                    a.stopBulletCamAndSwitchTo(attacker)
                }
            }
        })

        if activateBulletCam {
            a.startBulletCamFor(attacker, firedProjectile)
        }
        projectileNumber := index + 1
        if projectile.UnitHit >= 0 {
            hitUnit, knownUnit := a.GetClientUnit(uint64(projectile.UnitHit))
            if !knownUnit {
                damageReport += fmt.Sprintf("%d. something was hit\n", projectileNumber)
            } else if projectile.IsLethal {
                damageReport += fmt.Sprintf("%d. lethal hit on %s (%s)\n", projectileNumber, hitUnit.GetName(), projectile.BodyPart)
            } else {
                damageReport += fmt.Sprintf("%d. hit on %s (%s) for %d damage\n", projectileNumber, hitUnit.GetName(), projectile.BodyPart, projectile.Damage)
            }
        } else {
            damageReport += fmt.Sprintf("%d. missed\n", projectileNumber)
        }
    }
}

func (a *BattleClient) startBulletCamFor(attacker *Unit, firedProjectile *Projectile) {
    if !a.cameraIsFirstPerson {
        a.SwitchToFirstPerson()
    } else {
        a.onSwitchToISO()
    }
    a.onSwitchToIsoCamera = func() {
        a.fpsCamera.Detach()
    }
    a.fpsCamera.SetForward(firedProjectile.GetForward())
    a.fpsCamera.AttachTo(firedProjectile)
    a.crosshair.SetHidden(true)
}

func (a *BattleClient) stopBulletCamAndSwitchTo(unit *Unit) {
    a.fpsCamera.Detach()
    a.SwitchToUnit(unit)
}

func (a *BattleClient) OnEnemyUnitMoved(msg game.VisualEnemyUnitMoved) {
    // When an enemy unit is leaving the LOS of a player owned unit,
    // the space where the unit was standing is cleared on the client side map.
    movingUnit, _ := a.GetClientUnit(msg.MovingUnit)
    if movingUnit == nil && msg.UpdatedUnit == nil {
        util.LogGameError(fmt.Sprintf("[BattleClient] Received LOS update for unknown unit %d", msg.MovingUnit))
        return
    }
    if msg.UpdatedUnit != nil { // we lost LOS, so no update is sent
        a.AddOrUpdateUnit(msg.UpdatedUnit)
        movingUnit, _ = a.GetClientUnit(msg.MovingUnit)
        //println(fmt.Sprintf("[BattleClient] Received LOS update for unit %s(%d) at %s facing %s", movingUnit.GetName(), movingUnit.UnitID(), movingUnit.GetBlockPosition().ToString(), movingUnit.GetForward2DCardinal().ToString()))
    }
    //println(fmt.Sprintf("[BattleClient] Enemy unit %s(%d) moving", movingUnit.GetName(), movingUnit.UnitID()))
    /*
    	for i, path := range msg.PathParts {
    		//println(fmt.Sprintf("[BattleClient] Path %d", i))
    		for _, pathPos := range path {
    			//println(fmt.Sprintf("[BattleClient] --> %s", pathPos.ToString()))
    		}
    	}

    */
    hasPath := len(msg.PathParts) > 0 && len(msg.PathParts[0]) > 0
    changeLOS := func() {
        a.SetLOSAndPressure(msg.LOSMatrix, msg.PressureMatrix)
        if msg.UpdatedUnit != nil { // we lost LOS, so no update is sent
            movingUnit.SetForward(msg.UpdatedUnit.GetForward())
        }
        if hasPath && a.UnitIsVisibleToPlayer(a.GetControllingUserID(), movingUnit.UnitID()) { // if the unit has actually moved further, but we lost LOS, this will set a wrong position
            // even worse: if we lost the LOS, the unit was removed from the map, but this will add it again.
            movingUnit.SetBlockPositionAndUpdateStance(msg.PathParts[len(msg.PathParts)-1][len(msg.PathParts[len(msg.PathParts)-1])-1])
        }
    }
    currentPathPart := 0
    observerPositionReached := func(deltaTime float64) bool { return false }
    observerPositionReached = func(deltaTime float64) bool {
        reachedLastWaypoint := voxel.PositionToGridInt3(movingUnit.GetPosition()) == movingUnit.GetLastWaypoint()
        if !reachedLastWaypoint { // not yet at last waypoint
            return false
        }
        // we reached the last waypoint of the current path part
        if currentPathPart < len(msg.PathParts)-1 && len(msg.PathParts[currentPathPart+1]) > 0 { // not yet at last path part
            // it's not the last path part, so we can continue with the next one
            // TODO: add a delay here with invisibility
            // NOTE: We really instantly teleport to the next path part.
            currentPathPart++
            startPos := msg.PathParts[currentPathPart][0]
            movingUnit.SetBlockPosition(startPos)
            movingUnit.SetPath(msg.PathParts[currentPathPart])
            return false
        }
        if movingUnit.IsIdle() {
            return true
        }

        return false
    }

    if !hasPath {
        if msg.UpdatedUnit != nil {
            movingUnit.SetBlockPositionAndUpdateStance(msg.UpdatedUnit.GetBlockPosition())
        }
        changeLOS()
        return
    }
    firstPath := msg.PathParts[currentPathPart]
    startPos := firstPath[0]
    currentPos := movingUnit.GetBlockPosition()
    if voxel.ManhattanDistance2(currentPos, startPos) > 1 {
        movingUnit.SetBlockPositionAndUpdateStance(startPos)
        currentPos = startPos
    }
    destination := firstPath[len(firstPath)-1]
    if currentPos == destination {
        changeLOS()
    } else {
        movingUnit.SetPath(firstPath)
        a.scheduleWaitForCondition(observerPositionReached, func(deltaTime float64) {
            changeLOS()
        })
    }
}

func (a *BattleClient) OnOwnUnitMoved(msg game.VisualOwnUnitMoved) {
    unit, _ := a.GetClientUnit(msg.UnitID)
    if unit == nil {
        util.LogGraphicalClientGameError(fmt.Sprintf("[BattleClient] Unknown unit %d", msg.UnitID))
        return
    }

    util.LogGraphicalClientGameInfo(fmt.Sprintf("[BattleClient] Moving %s(%d): %v -> %v", unit.GetName(), unit.UnitID(), unit.GetBlockPosition(), msg.Path[len(msg.Path)-1]))

    a.highlights.ClearFlat(voxel.HighlightMove)
    a.unitSelector.Hide()
    a.lines.Clear()

    destination := msg.Path[len(msg.Path)-1]

    changeLOS := func(deltaTime float64) {
        a.SetLOSAndPressure(msg.LOSMatrix, msg.PressureMatrix)
        for _, acquiredLOSUnit := range msg.Spotted {
            a.AddOrUpdateUnit(acquiredLOSUnit)
        }
        unit.SetBlockPositionAndUpdateStance(destination)
        if a.selectedUnit == unit {
            a.SwitchToUnit(unit)
        }
    }
    destinationReached := func(deltaTime float64) bool { // problem: we are hanging here..
        return unit.IsAtLocation(destination)
    }
    a.scheduleWaitForCondition(destinationReached, changeLOS)

    // HMM, maybe the ending the turn could interfer with the movement animations?
    // but then, it doesn't appear until we fired at someone
    // the scenario seems to imply, that the unit does not receive the NewWayPoint Event or doesn't act on it
    // apparently, we don't leave the hit animation state beforehand
    // because we received a NewPath event, before the AnimationFinished event.
    // NewPath is not a transition from the HitState -> Deadlock

    unit.UseMovement(msg.Cost)
    unit.SetPath(msg.Path)
}

func (a *BattleClient) OnNextPlayer(msg game.NextPlayerMessage) {
    util.LogGraphicalClientGameDebug(fmt.Sprintf("[BattleClient] NextPlayer: %v", msg))
    //println("[BattleClient] Map State:")
    //a.GetVoxelMap().PrintArea2D(16, 16)
    /*
    for _, unit := range a.GetAllUnits() {
        println(fmt.Sprintf("[BattleClient] > Unit %s(%d): %v", unit.GetName(), unit.UnitID(), unit.GetBlockPosition()))
    }

    */

    if msg.YourTurn {
        a.smoker.NextTurn()

        a.ResetOverwatch()
        a.ResetUnitsForNextTurn()
        //a.Print("It's your turn!")
        if a.GetMissionDetails().Scenario == game.MissionScenarioDefend {
            if a.GameInstance.IndexOfPlayer(a.GetControllingUserID()) == 0 {
                a.FlashText("DEFEND!", 3)
            } else {
                a.FlashText("DESTROY!", 3)
            }
        } else {
            a.FlashText("FIGHT!", 3)
        }
        a.SwitchToUnitNoCameraMovement(a.FirstUnit())
    } else {
        a.FlashText("WAIT!", 3)
        a.SwitchToWaitForEvents()
    }
}

func (a *BattleClient) EndTurn() {
    util.MustSend(a.server.EndTurn())
    a.SwitchToWaitForEvents()
}

func (a *BattleClient) FirstUnit() *Unit {
    for _, unit := range a.GetPlayerUnits(a.GetControllingUserID()) {
        if unit.CanAct() {
            clientUnit, _ := a.GetClientUnit(unit.UnitID())
            return clientUnit
        }
    }
    return nil
}

func (a *BattleClient) freezeIdleAnimations() {
    for _, unit := range a.GetAllUnits() {
        if unit.IsActive() {
            clientUnit, _ := a.GetClientUnit(unit.UnitID())
            clientUnit.FreezeStanceAnimation()
        }
    }
}

func (a *BattleClient) resumeIdleAnimations() {
    for _, unit := range a.GetAllUnits() {
        if unit.IsActive() && unit.IsPlayingIdleAnimation() {
            clientUnit, _ := a.GetClientUnit(unit.UnitID())
            clientUnit.StartStanceAnimation()
        }
    }
}

func (a *BattleClient) pollNetwork() {
    select {
    case msg := <-a.serverChannel:
        a.OnServerMessage(msg.MessageType, msg.Message)
    default:
    }
}

func (a *BattleClient) OnGameOver(msg game.GameOverMessage) {
    a.GameClient.OnGameOver(msg)
    a.SwitchToWaitForEvents()
    var printedMessage string
    if msg.YouWon {
        printedMessage = "VICTORY!"
    } else {
        printedMessage = "DEFEAT!"
    }
    a.FlashText(printedMessage, 3)
}

func (a *BattleClient) IsUnitOwnedByClient(unitID uint64) bool {
    unit, _ := a.GetClientUnit(unitID)
    return unit != nil && unit.IsUserControlled()
}

func (a *BattleClient) AddBlood(unitHit *Unit, entryWoundPosition mgl32.Vec3, bulletVelocity mgl32.Vec3, partHit util.DamageZone) {
    // TODO: UpdateMapPosition blood oneShotParticles
    // TODO: AddFlat blood decals on unit skin
    bloodProps := a.particleProps[ParticlesBlood].WithOrigin(entryWoundPosition)
    a.oneShotParticles.Emit(bloodProps, 10)
}
func (a *BattleClient) AddBulletImpact(worldPosition mgl32.Vec3, bulletVelocity mgl32.Vec3) {
    bloodProps := a.particleProps[ParticlesBulletImpact].WithOrigin(worldPosition)
    a.oneShotParticles.Emit(bloodProps, 10)
}
func (a *BattleClient) SetHighlightsForMovement(action *game.ActionMove, unit *Unit, targets []voxel.Int3) {
    var snapRange []voxel.Int3  // can snap fire from here
    var aimedRange []voxel.Int3 // can free aim from here
    var farRange []voxel.Int3   // can't fire from here

    apNeededForFiring := unit.GetWeapon().Definition.BaseAPForShot

    apAvailable := unit.GetExactAP()

    budgetForSnap := apAvailable - float64(apNeededForFiring)
    budgetForAimed := apAvailable - float64(apNeededForFiring+1)

    movesPerAp := unit.Definition.CoreStats.MovementPerAP

    budgetInBlocksForSnap := budgetForSnap * movesPerAp
    budgetInBlocksForAimed := budgetForAimed * movesPerAp

    for _, target := range targets {
        cost := float64(action.GetCost(target))
        if cost <= budgetInBlocksForAimed {
            aimedRange = append(aimedRange, target)
        } else if cost <= budgetInBlocksForSnap {
            snapRange = append(snapRange, target)
        } else {
            farRange = append(farRange, target)
        }
    }
    // we want a scifi techy x-com blue for the near range
    // we want a corresponding scify techy orange (blade runner) for the far range
    a.highlights.ClearFlat(voxel.HighlightMove)
    a.highlights.AddFlat(voxel.HighlightMove, aimedRange, mgl32.Vec3{0.412, 0.922, 1})
    a.highlights.AddFlat(voxel.HighlightMove, snapRange, mgl32.Vec3{0.361, 0.714, 1})
    a.highlights.AddFlat(voxel.HighlightMove, farRange, mgl32.Vec3{1, 0.537, 0.2})
    a.highlights.ShowAsFlat(voxel.HighlightMove)
}

func (a *BattleClient) updateOverwatchHighlights() {
    a.highlights.ClearFancy(voxel.HighlightOverwatch)
    a.highlights.AddFancy(voxel.HighlightOverwatch, a.overwatchPositionsThisTurn, mgl32.Vec3{1, 0.2, 0.2})
    a.highlights.ShowAsFancy(voxel.HighlightOverwatch)
}

func (a *BattleClient) ResetOverwatch() {
    a.overwatchPositionsThisTurn = nil
    a.highlights.ClearAndUpdateFancy(voxel.HighlightOverwatch)
}

func (a *BattleClient) SetSelectedUnit(unit *Unit) {
    a.selectedUnit = unit

    a.SwitchToGroundSelector()
    a.unitSelector.SetBlockPosition(unit.GetBlockPosition())

    a.UpdateActionbarFor(unit)
}

func (a *BattleClient) OnDebugGameStateRececeivedFromServer(msg game.CompleteGameState) {
    serverUnitsState := msg.AllUnits
    anyDifferences := false
    clientUnitsState := a.DebugGetCompleteState().AllUnits
    println(fmt.Sprintf("[BattleClient] Client state with %d units.", len(clientUnitsState)))
    for _, unitState := range clientUnitsState {
        unitID := unitState.ID
        serverUnitState, ok := serverUnitsState[unitID]
        if !ok {
            println(fmt.Sprintf("[BattleClient] Unit %d not found in server state", unitID))
            continue
        }
        isTheSame, reasonForDifference := unitState.Equals(serverUnitState)
        if !isTheSame {
            println(fmt.Sprintf("[BattleClient] Unit %d has non-matching state:\n%s", unitID, reasonForDifference))
            println(unitState.Diff(serverUnitState))
            anyDifferences = true
        }
    }

    if anyDifferences {
        println("[BattleClient] Client state DIFFERS from server state")
    } else {
        println("[BattleClient] Client state MATCHES server state")
    }
}

func (a *BattleClient) handleResizeEvents(width int, height int) {
    a.WindowWidth = width // PROBLEM: This is the value we restore when switching back..
    a.WindowHeight = height
    a.aspectRatio = float32(width) / float32(height)

    a.isoCamera.SetScreenSize(width, height)
    a.fpsCamera.SetScreenSize(width, height)

    a.actionbar.SetScreenSize(width, height)

    a.guiShader.Begin()
    a.guiShader.SetUniformAttr(0, util.Get2DPixelCoordOrthographicProjectionMatrix(a.WindowWidth, a.WindowHeight))
    a.guiShader.End()
}

func (a *BattleClient) OnStartDeployment() {
    // show the spawn markers on the map
    // attach the current unit to the cursor
    a.SwitchToDeployment()
}

func (a *BattleClient) FlashText(text string, delayInSeconds float64) {
    a.bigLabel.SetText(text)
    a.scheduleUpdateIn(delayInSeconds, func(deltaTime float64) {
        if a.bigLabel.GetText() == text {
            a.bigLabel.Hide()
        }
    })
}

func (a *BattleClient) OnExplode(origin voxel.Int3, radius float64) {
    properties := a.particleProps[ParticlesExplosion].WithOrigin(origin.ToBlockCenterVec3())
    a.oneShotParticles.Emit(properties, 100)
}

func (a *BattleClient) SetSelectedBlocks(selection []voxel.Int3) {
    a.selectedBlocks = selection
}


