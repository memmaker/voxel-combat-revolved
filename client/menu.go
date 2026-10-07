package client

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"os"
	"runtime"
	"time"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/glhf"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/game"
)

type MatchMode int

const (
	ModeVsAI MatchMode = iota
	ModeHost
	ModeJoin
	ModeQuit
	ModeEditor
)

// MatchConfig is everything the player chose in the menu before a match starts.
type MatchConfig struct {
	Mode            MatchMode
	PlayerName      string
	Address         string // join: server address
	GameID          string // join: chosen game
	Connection      *game.ServerConnection
	MapFile         string
	Scenario        game.MissionScenario
	Placement       game.PlacementMode
	FactionName     string
	Squad           []game.UnitChoice
	OpponentFaction string
	Difficulty      game.AIDifficulty
	EditorMap       string // ModeEditor: map to open
	TurnSeconds     int
}

// Mission builds the mission details for the chosen scenario and deployment mode.
func (c MatchConfig) Mission() *game.MissionDetails {
	var details *game.MissionDetails
	if c.Scenario == game.MissionScenarioDefend {
		details = game.NewRandomDefend()
	} else {
		details = game.NewRandomDeathmatch()
	}
	details.Placement = c.Placement
	details.TurnSeconds = c.TurnSeconds
	return details
}

type menuScreen int

const (
	screenMain menuScreen = iota
	screenSetup
	screenJoin
	screenSettings
	screenEditor
)

type squadSlot struct {
	unitType int // index into the faction's playable unit types
	name     string
	weapon   int
	item     int // 0 = none
}

// Menu is the front end of the game: mode selection, match setup with squad editor, joining and settings.
type Menu struct {
	window   *glfw.Window
	settings *ClientSettings
	content  game.Content
	ui       *UI
	shader   *glhf.Shader
	title    *util.BitmapFontMesh

	width, height  int
	mouseX, mouseY float64
	clicked        bool
	typedKeys      []glfw.Key
	typedShift     bool

	screen       menuScreen
	mode         MatchMode
	result       *MatchConfig
	mapIndex     int
	scenario     int
	placement    int
	faction      int
	difficulty   int
	turnTimer    int
	squad        []squadSlot
	address      string
	addressFocus bool
	status       string
	lobby        *lobbyConnection
	lastConfig   *MatchConfig
}

var scenarioOptions = []game.MissionScenario{game.MissionScenarioDeathmatch, game.MissionScenarioDefend}
var placementOptions = []game.PlacementMode{game.PlacementModeRandom, game.PlacementModeManual}
var difficultyOptions = []game.AIDifficulty{game.AIEasy, game.AINormal, game.AIHard}

const maxSquadSize = 4

func NewMenu(window *glfw.Window, settings *ClientSettings, content game.Content) *Menu {
	width, height := window.GetSize()
	assets := game.NewAssets()
	shader := newGuiShader(width, height)
	fontTexture, fontIndex := assets.LoadBitmapFont("quadratica", 8, 14)
	bigFont := assets.LoadBitmapFontWithoutIndex("STEEL_MV", 32, 32)
	bigFont.SetPaddingBetweenItems(0, 1)
	bigFontIndex := util.NewIndexFromDescription(util.AtlasDescription{
		PositionOfCapitalA:     [2]int{2, 3},
		PositionOfZero:         &[2]int{5, 1},
		PositionOfSpecialChain: &[2]int{0, 0},
		Cols:                   10,
		SpecialCharacterChain:  []rune{'!', 'Ö', 'Ä', '"', '§', '*', '(', ')', '+', ',', ' ', '-', '.', '/'},
	})
	title := util.NewBitmapFontMesh(shader, bigFont, bigFontIndex.GetMapper())
	title.SetCenterOrigin(true)
	title.SetDiscardColor(mgl32.Vec3{0, 0, 0})
	title.SetTintColor(game.ColorTechTeal.Vec3())
	title.SetText("BLOCK TACTICS")

	m := &Menu{
		window:     window,
		settings:   settings,
		content:    content,
		ui:         NewUI(shader, fontTexture, fontIndex.GetMapper(), 8, 14),
		shader:     shader,
		title:      title,
		width:      width,
		height:     height,
		difficulty: 1,
		address:    "127.0.0.1:9999",
	}
	m.resetSquad()
	return m
}

func (m *Menu) installCallbacks() {
	m.window.SetCursorPosCallback(func(w *glfw.Window, x, y float64) { m.mouseX, m.mouseY = x, y })
	m.window.SetMouseButtonCallback(func(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
		if button == glfw.MouseButtonLeft && action == glfw.Release {
			m.clicked = true
		}
	})
	m.window.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		if action == glfw.Press || action == glfw.Repeat {
			m.typedKeys = append(m.typedKeys, key)
			m.typedShift = mods&glfw.ModShift != 0
		}
	})
	m.window.SetScrollCallback(func(w *glfw.Window, x, y float64) {})
	m.window.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
}

// Run shows the menu until the player starts a match or quits.
func (m *Menu) Run() MatchConfig {
	m.installCallbacks()
	GetAudio().PlayMusic(m.settings.MusicVolume)
	m.result = nil
	m.screen = screenMain
	switch os.Getenv("BT_MENU_SCREEN") { // development aid for screenshots
	case "setup":
		m.screen = screenSetup
	case "settings":
		m.screen = screenSettings
	case "join":
		m.screen = screenJoin
	}
	screenshots := util.NewAutoScreenshot()
	lastFrame := glfw.GetTime()
	for m.result == nil {
		if m.window.ShouldClose() {
			return MatchConfig{Mode: ModeQuit}
		}
		m.frame()
		now := glfw.GetTime()
		if screenshots != nil {
			fbWidth, fbHeight := m.window.GetFramebufferSize()
			screenshots.Update(now-lastFrame, fbWidth, fbHeight)
		}
		lastFrame = now
		m.window.SwapBuffers()
		glfw.PollEvents()
	}
	m.lastConfig = m.result
	return *m.result
}

// ShowStatus renders a message screen until done returns true or the player cancels (returns false).
func (m *Menu) ShowStatus(message string, done func() bool) bool {
	m.installCallbacks()
	for !done() {
		if m.window.ShouldClose() {
			return false
		}
		m.beginFrame()
		m.ui.CenteredLabel(float32(m.width)/2, float32(m.height)/2-30, message, 1, uiTextColor)
		cancel := m.ui.Button(float32(m.width)/2-80, float32(m.height)/2+10, 160, 32, "Cancel", true)
		m.endFrame()
		m.window.SwapBuffers()
		glfw.PollEvents()
		if cancel {
			return false
		}
	}
	return true
}

func (m *Menu) beginFrame() {
	gl.ClearColor(0.02, 0.03, 0.04, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	m.ui.Begin(m.width, m.height, m.mouseX, m.mouseY, m.clicked)
	m.clicked = false
}

func (m *Menu) endFrame() {
	m.shader.Begin()
	m.ui.Draw()
	m.title.SetPosition(mgl32.Vec2{float32(m.width) / 2, 70})
	gl.Disable(gl.DEPTH_TEST)
	m.title.Draw()
	gl.Enable(gl.DEPTH_TEST)
	m.shader.End()
	m.typedKeys = m.typedKeys[:0]
}

func (m *Menu) frame() {
	m.beginFrame()
	switch m.screen {
	case screenMain:
		m.mainScreen()
	case screenSetup:
		m.setupScreen()
	case screenJoin:
		m.joinScreen()
	case screenSettings:
		m.settingsScreen()
	case screenEditor:
		m.editorScreen()
	}
	if m.status != "" {
		m.ui.CenteredLabel(float32(m.width)/2, float32(m.height)-30, m.status, 1, game.ColorNegativeRed.Vec3())
	}
	m.endFrame()
}

func (m *Menu) mainScreen() {
	buttonWidth, buttonHeight := float32(260), float32(36)
	x := (float32(m.width) - buttonWidth) / 2
	y := float32(150)
	m.ui.CenteredLabel(float32(m.width)/2, 110, "Turn-based tactics in a destructible world", 1, uiTextDim)
	if m.lastConfig != nil && m.ui.Button(x, y, buttonWidth, buttonHeight, "Play again", true) {
		again := *m.lastConfig
		again.Connection = nil
		if again.Mode == ModeJoin {
			m.screen = screenJoin
		} else {
			m.result = &again
		}
		return
	}
	if m.lastConfig != nil {
		y += buttonHeight + 12
	}
	if m.ui.Button(x, y, buttonWidth, buttonHeight, "Play against the AI", true) {
		m.mode, m.screen, m.status = ModeVsAI, screenSetup, ""
	}
	y += buttonHeight + 12
	networkAvailable := runtime.GOOS != "js"
	if m.ui.Button(x, y, buttonWidth, buttonHeight, "Host a multiplayer game", networkAvailable) {
		m.mode, m.screen, m.status = ModeHost, screenSetup, ""
	}
	y += buttonHeight + 12
	if m.ui.Button(x, y, buttonWidth, buttonHeight, "Join a multiplayer game", networkAvailable) {
		m.mode, m.screen, m.status = ModeJoin, screenJoin, ""
	}
	y += buttonHeight + 12
	if m.ui.Button(x, y, buttonWidth, buttonHeight, "Map editor", runtime.GOOS != "js") {
		m.screen, m.status = screenEditor, ""
	}
	y += buttonHeight + 12
	if m.ui.Button(x, y, buttonWidth, buttonHeight, "Settings", true) {
		m.screen, m.status = screenSettings, ""
	}
	y += buttonHeight + 12
	if runtime.GOOS != "js" && m.ui.Button(x, y, buttonWidth, buttonHeight, "Quit", true) {
		m.result = &MatchConfig{Mode: ModeQuit}
	}
}

// playableUnits are the unit types of a faction that have animations (the big 3x3 units do not, yet).
func (m *Menu) playableUnits(factionIndex int) []game.UnitDefinition {
	var result []game.UnitDefinition
	for _, unit := range m.content.Factions[factionIndex].Units {
		if len(unit.AnimationMap) > 0 {
			result = append(result, unit)
		}
	}
	return result
}

var unitNames = []string{"Ash", "Brick", "Cole", "Dex", "Echo", "Fitz", "Gray", "Hale", "Ivo", "Jax", "Kade", "Lux", "Mara", "Nox", "Oren", "Pike", "Quinn", "Rook", "Sage", "Tate", "Vale", "Wren", "Yara", "Zane"}

func (m *Menu) resetSquad() {
	names := rand.Perm(len(unitNames))
	defaultWeapons := []string{"M16 Rifle", "Steyr SSG 69", "Mossberg 500", "M1911 Pistol"}
	defaultItems := []string{"Frag Grenade", "Smoke Grenade", "Frag Grenade", "Smoke Grenade"}
	m.squad = nil
	for i := 0; i < 3; i++ {
		m.squad = append(m.squad, squadSlot{
			name:     unitNames[names[i]],
			unitType: []int{0, 2, 1}[i], // rifleman, marksman, scout (order of the content file)
			weapon:   m.weaponIndex(defaultWeapons[i]),
			item:     m.itemIndex(defaultItems[i]) + 1,
		})
	}
}

func (m *Menu) weaponIndex(name string) int {
	for i, weapon := range m.content.Weapons {
		if weapon.UniqueName == name {
			return i
		}
	}
	return 0
}

func (m *Menu) itemIndex(name string) int {
	for i, item := range m.content.Items {
		if item.UniqueName == name {
			return i
		}
	}
	return 0
}

func (m *Menu) setupScreen() {
	columnWidth := min(float32(360), (float32(m.width)-80)/2)
	left := float32(m.width)/2 - columnWidth - 20
	y := float32(110)
	rowHeight, rowGap := float32(26), float32(8)

	heading := "Play against the AI"
	if m.mode == ModeHost {
		heading = "Host a multiplayer game"
	} else if m.mode == ModeJoin {
		heading = "Join: choose your squad"
	}
	m.ui.Label(left, y, heading, 2, uiTextHighlight)
	y += 44

	// match options (left column)
	optionsY := y
	if m.mode != ModeJoin {
		mapNames := make([]string, len(m.content.Maps))
		for i, entry := range m.content.Maps {
			mapNames[i] = entry.Name
		}
		m.mapIndex = m.ui.Selector(left, optionsY, columnWidth, rowHeight, "Map", mapNames, m.mapIndex)
		optionsY += rowHeight + rowGap
		m.scenario = m.ui.Selector(left, optionsY, columnWidth, rowHeight, "Mission", []string{"Deathmatch", "Defend"}, m.scenario)
		optionsY += rowHeight + rowGap
		m.placement = m.ui.Selector(left, optionsY, columnWidth, rowHeight, "Deployment", []string{"Random", "Manual"}, m.placement)
		optionsY += rowHeight + rowGap
	}
	factionNames := make([]string, len(m.content.Factions))
	for i, faction := range m.content.Factions {
		factionNames[i] = faction.Name
	}
	previousFaction := m.faction
	m.faction = m.ui.Selector(left, optionsY, columnWidth, rowHeight, "Your faction", factionNames, m.faction)
	if m.faction != previousFaction {
		for i := range m.squad {
			m.squad[i].unitType = 0
		}
	}
	optionsY += rowHeight + rowGap
	if m.mode == ModeHost {
		m.turnTimer = m.ui.Selector(left, optionsY, columnWidth, rowHeight, "Turn timer", []string{"Off", "60 s", "90 s", "120 s"}, m.turnTimer)
		optionsY += rowHeight + rowGap
	}
	if m.mode == ModeVsAI {
		m.difficulty = m.ui.Selector(left, optionsY, columnWidth, rowHeight, "AI difficulty", []string{"Easy", "Normal", "Hard"}, m.difficulty)
		optionsY += rowHeight + rowGap
	}
	if m.scenario == 1 && m.mode != ModeJoin {
		m.ui.Label(left, optionsY+6, "Defend: the host defends the objective,", 1, uiTextDim)
		m.ui.Label(left, optionsY+24, "the other side has to destroy it.", 1, uiTextDim)
	}

	// squad editor (right column)
	squadX := left + columnWidth + 40
	_ = rowGap
	squadY := y
	m.ui.Label(squadX, squadY, fmt.Sprintf("Squad (%d/%d)", len(m.squad), maxSquadSize), 1, uiTextColor)
	squadY += 24
	units := m.playableUnits(m.faction)
	unitTypeNames := make([]string, len(units))
	for i, unit := range units {
		name := unit.ClassName
		if name == "" {
			name = unit.ModelFile
		}
		unitTypeNames[i] = fmt.Sprintf("%s (%d HP)", name, unit.CoreStats.Health)
	}
	weaponNames := make([]string, len(m.content.Weapons))
	for i, weapon := range m.content.Weapons {
		weaponNames[i] = weapon.UniqueName
	}
	itemNames := []string{"None"}
	for _, item := range m.content.Items {
		itemNames = append(itemNames, item.UniqueName)
	}
	remove := -1
	cardRow := float32(22)
	for i := range m.squad {
		slot := &m.squad[i]
		cardHeight := cardRow*3 + 14
		if len(units) > 1 {
			cardHeight += cardRow + 4
		}
		m.ui.Panel(squadX, squadY, columnWidth, cardHeight)
		m.ui.Label(squadX+8, squadY+6, slot.name, 1, uiTextHighlight)
		if len(m.squad) > 1 && m.ui.Button(squadX+columnWidth-26, squadY+4, 20, 18, "x", true) {
			remove = i
		}
		rowY := squadY + cardRow + 6
		if len(units) > 1 {
			slot.unitType = m.ui.Selector(squadX+8, rowY, columnWidth-16, cardRow, "Type", unitTypeNames, slot.unitType)
			rowY += cardRow + 4
		}
		slot.weapon = m.ui.Selector(squadX+8, rowY, columnWidth-16, cardRow, "Weapon", weaponNames, slot.weapon)
		rowY += cardRow + 4
		slot.item = m.ui.Selector(squadX+8, rowY, columnWidth-16, cardRow, "Item", itemNames, slot.item)
		squadY += cardHeight + 8
	}
	if remove >= 0 {
		m.squad = append(m.squad[:remove], m.squad[remove+1:]...)
	}
	if len(m.squad) < maxSquadSize && m.ui.Button(squadX, squadY, columnWidth, 26, "+ Add soldier", true) {
		used := make(map[string]bool)
		for _, slot := range m.squad {
			used[slot.name] = true
		}
		name := unitNames[rand.Intn(len(unitNames))]
		for used[name] {
			name = unitNames[rand.Intn(len(unitNames))]
		}
		m.squad = append(m.squad, squadSlot{name: name, weapon: m.weaponIndex("M16 Rifle"), item: m.itemIndex("Frag Grenade") + 1})
	}

	// start / back below the match options
	buttonsY := optionsY + rowHeight + 50
	startLabel := "Start match"
	if m.mode == ModeHost {
		startLabel = "Host game"
	} else if m.mode == ModeJoin {
		startLabel = "Join game"
	}
	if m.ui.Button(left, buttonsY, columnWidth, 40, startLabel, len(m.squad) > 0) {
		m.result = m.buildConfig()
	}
	if m.ui.Button(left, buttonsY+52, columnWidth, 32, "Back", true) {
		if m.lobby != nil {
			m.lobby.close()
			m.lobby = nil
		}
		m.screen = screenMain
	}
}

func (m *Menu) buildConfig() *MatchConfig {
	units := m.playableUnits(m.faction)
	config := &MatchConfig{
		Mode:        m.mode,
		PlayerName:  m.settings.PlayerName,
		Scenario:    scenarioOptions[m.scenario],
		Placement:   placementOptions[m.placement],
		FactionName: m.content.Factions[m.faction].Name,
		Difficulty:  difficultyOptions[m.difficulty],
		Address:     m.address,
		TurnSeconds: []int{0, 60, 90, 120}[m.turnTimer],
	}
	if len(m.content.Maps) > 0 {
		config.MapFile = game.WithRandomSeed(m.content.Maps[m.mapIndex].File, rand.New(rand.NewSource(time.Now().UnixNano())))
	}
	config.OpponentFaction = m.content.Factions[(m.faction+1)%len(m.content.Factions)].Name
	for _, slot := range m.squad {
		choice := game.UnitChoice{
			UnitTypeID: units[min(slot.unitType, len(units)-1)].ID,
			Name:       slot.name,
			Weapon:     m.content.Weapons[slot.weapon].UniqueName,
		}
		if slot.item > 0 {
			choice.Items = []string{m.content.Items[slot.item-1].UniqueName}
		}
		config.Squad = append(config.Squad, choice)
	}
	if m.mode == ModeJoin && m.lobby != nil {
		config.Connection = m.lobby.connection
		config.GameID = m.lobby.selectedGame
		m.lobby.detach()
		m.lobby = nil
	}
	return config
}

// OpponentSquad creates an AI squad of the same size for the opponent faction.
func (c MatchConfig) OpponentSquad(content game.Content) []game.UnitChoice {
	var faction game.FactionDefinition
	for _, f := range content.Factions {
		if f.Name == c.OpponentFaction {
			faction = f
		}
	}
	classID := func(className string) uint64 {
		for _, unit := range faction.Units {
			if unit.ClassName == className {
				return unit.ID
			}
		}
		for _, unit := range faction.Units {
			if len(unit.AnimationMap) > 0 {
				return unit.ID
			}
		}
		return 0
	}
	loadouts := []struct{ class, weapon, item string }{
		{"Rifleman", "M16 Rifle", "Frag Grenade"},
		{"Marksman", "Steyr SSG 69", "Smoke Grenade"},
		{"Scout", "Mossberg 500", "Frag Grenade"},
		{"Medic", "M1911 Pistol", "Smoke Grenade"},
	}
	names := []string{"Gnarg", "Gorn", "Grimbel", "Gluk"}
	var squad []game.UnitChoice
	for i := range c.Squad {
		loadout := loadouts[i%len(loadouts)]
		squad = append(squad, game.UnitChoice{UnitTypeID: classID(loadout.class), Name: names[i%len(names)], Weapon: loadout.weapon, Items: []string{loadout.item}})
	}
	return squad
}

func (m *Menu) handleTyping() {
	for _, key := range m.typedKeys {
		switch {
		case key == glfw.KeyBackspace || key == glfw.KeyDelete:
			if len(m.address) > 0 {
				m.address = m.address[:len(m.address)-1]
			}
		case key >= glfw.Key0 && key <= glfw.Key9:
			if m.typedShift && key == glfw.Key0+0 {
				continue
			}
			m.address += string(rune('0' + key - glfw.Key0))
		case key == glfw.KeyPeriod:
			m.address += "."
		case key == glfw.KeySemicolon && m.typedShift:
			m.address += ":"
		case key >= glfw.KeyA && key <= glfw.KeyZ:
			m.address += string(rune('a' + key - glfw.KeyA))
		case key == glfw.KeyMinus:
			m.address += "-"
		case key == glfw.KeyEnter:
			m.addressFocus = false
		}
	}
}

func (m *Menu) joinScreen() {
	left := float32(m.width)/2 - 300
	y := float32(140)
	m.ui.Label(left, y, "Join a multiplayer game", 2, uiTextHighlight)
	y += 50
	if m.addressFocus {
		m.handleTyping()
	}
	if m.ui.TextField(left, y, 600, 30, "Server address", m.address, m.addressFocus) {
		m.addressFocus = true
	} else if m.clickedElsewhere() {
		m.addressFocus = false
	}
	y += 44
	connecting := m.lobby != nil && m.lobby.state() == lobbyConnecting
	if m.ui.Button(left, y, 200, 32, "Connect & refresh", !connecting) {
		if m.lobby != nil {
			m.lobby.close()
		}
		m.lobby = newLobbyConnection(m.address, m.settings.PlayerName)
	}
	y += 50
	if m.lobby != nil {
		state := m.lobby.state()
		switch state {
		case lobbyConnecting:
			m.ui.Label(left, y, "Connecting...", 1, uiTextDim)
		case lobbyFailed:
			m.ui.Label(left, y, "Could not connect: "+m.lobby.errorMessage(), 1, game.ColorNegativeRed.Vec3())
		case lobbyConnected:
			games := m.lobby.games()
			if len(games) == 0 {
				m.ui.Label(left, y, "No open games on this server. Ask your friend to host one.", 1, uiTextDim)
			}
			for _, listing := range games {
				label := fmt.Sprintf("%s  -  %s on %s  (%d/%d players)", listing.GameID, listing.Scenario, listing.MapFile, listing.Players, listing.PlayersNeeded)
				if listing.Rejoinable {
					label = fmt.Sprintf("REJOIN %s  -  waiting for %s", listing.GameID, listing.Absent)
				}
				if m.ui.Button(left, y, 600, 30, label, true) {
					m.lobby.selectedGame = listing.GameID
					if listing.Rejoinable { // the squad is already on the battlefield: go straight in
						m.result = m.buildConfig()
					} else {
						m.screen = screenSetup
					}
				}
				y += 36
			}
		}
	}
	if m.ui.Button(left, float32(m.height)-80, 160, 36, "Back", true) {
		if m.lobby != nil {
			m.lobby.close()
			m.lobby = nil
		}
		m.screen = screenMain
	}
}

func (m *Menu) clickedElsewhere() bool {
	return m.ui.clicked && !m.ui.ConsumedClick()
}

var resolutionOptions = [][2]int{{1280, 720}, {1600, 900}, {1920, 1080}, {1024, 768}, {800, 600}}

func (m *Menu) settingsScreen() {
	left := float32(m.width)/2 - 260
	y := float32(140)
	width, rowHeight := float32(520), float32(28)
	s := m.settings
	m.ui.Label(left, y, "Settings", 2, uiTextHighlight)
	y += 50

	resolutionNames := make([]string, len(resolutionOptions))
	current := 0
	for i, r := range resolutionOptions {
		resolutionNames[i] = fmt.Sprintf("%d x %d", r[0], r[1])
		if r[0] == s.Width && r[1] == s.Height {
			current = i
		}
	}
	chosen := m.ui.Selector(left, y, width, rowHeight, "Resolution (restart)", resolutionNames, current)
	s.Width, s.Height = resolutionOptions[chosen][0], resolutionOptions[chosen][1]
	y += rowHeight + 8
	s.FullScreen = m.ui.Toggle(left, y, width, rowHeight, "Fullscreen (restart, F6)", s.FullScreen)
	y += rowHeight + 8
	s.FPSCameraInvertedMouse = m.ui.Toggle(left, y, width, rowHeight, "Invert mouse in aim mode", s.FPSCameraInvertedMouse)
	y += rowHeight + 8
	sensitivities := []float32{0.04, 0.06, 0.08, 0.1, 0.12, 0.15, 0.2}
	sensitivityNames := make([]string, len(sensitivities))
	sensitivityIndex := 2
	for i, value := range sensitivities {
		sensitivityNames[i] = fmt.Sprintf("%.2f", value)
		if mgl32.Abs(value-s.FPSCameraMouseSensitivity) < 0.005 {
			sensitivityIndex = i
		}
	}
	s.FPSCameraMouseSensitivity = sensitivities[m.ui.Selector(left, y, width, rowHeight, "Aim sensitivity", sensitivityNames, sensitivityIndex)]
	y += rowHeight + 8
	s.EnableCameraAnimations = m.ui.Toggle(left, y, width, rowHeight, "Camera animations", s.EnableCameraAnimations)
	y += rowHeight + 8
	s.EnableActionCam = m.ui.Toggle(left, y, width, rowHeight, "Action camera", s.EnableActionCam)
	y += rowHeight + 8
	s.EnableBulletCam = m.ui.Toggle(left, y, width, rowHeight, "Bullet camera (single shots)", s.EnableBulletCam)
	y += rowHeight + 8
	s.AutoSwitchToIsoCameraAfterFiring = m.ui.Toggle(left, y, width, rowHeight, "Leave aim view after firing", s.AutoSwitchToIsoCameraAfterFiring)
	y += rowHeight + 8
	s.SoundVolume = float32(m.ui.Selector(left, y, width, rowHeight, "Sound volume", []string{"Off", "10%", "20%", "30%", "40%", "50%", "60%", "70%", "80%", "90%", "100%"}, int(s.SoundVolume*10+0.5))) / 10
	y += rowHeight + 8
	previousMusic := s.MusicVolume
	s.MusicVolume = float32(m.ui.Selector(left, y, width, rowHeight, "Music volume", []string{"Off", "10%", "20%", "30%", "40%", "50%", "60%", "70%", "80%", "90%", "100%"}, int(s.MusicVolume*10+0.5))) / 10
	if s.MusicVolume != previousMusic {
		GetAudio().PlayMusic(s.MusicVolume)
	}
	y += rowHeight + 8

	if m.ui.Button(left, float32(m.height)-80, 160, 36, "Back", true) {
		m.screen = screenMain
		if err := s.SaveToFile("settings.json"); err != nil {
			m.status = "Could not save settings: " + err.Error()
		}
	}
}

// SaveToFile writes the settings as JSON (no-op in the browser).
func (s ClientSettings) SaveToFile(filename string) error {
	if runtime.GOOS == "js" {
		return nil
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, append(data, '\n'), 0o644)
}

func dialWithTimeout(address string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout("tcp", address, timeout)
}

// ---- lobby connection used by the join screen ----

type lobbyState int

const (
	lobbyConnecting lobbyState = iota
	lobbyConnected
	lobbyFailed
)

type lobbyConnection struct {
	connection   *game.ServerConnection
	updates      chan func(l *lobbyConnection)
	current      lobbyState
	err          string
	listing      []game.GameListing
	selectedGame string
	detached     bool
}

func newLobbyConnection(address, name string) *lobbyConnection {
	l := &lobbyConnection{updates: make(chan func(l *lobbyConnection), 16)}
	go func() {
		conn, err := dialWithTimeout(address, 4*time.Second)
		if err != nil {
			l.updates <- func(l *lobbyConnection) { l.current, l.err = lobbyFailed, err.Error() }
			return
		}
		connection := game.NewServerConnectionFromConn(conn, func(msg game.StringMessage) {
			switch msg.MessageType {
			case "LoginResponse":
				l.updates <- func(l *lobbyConnection) { util.MustSend(l.connection.ListGames()) }
			case "GameList":
				var list game.GameListMessage
				if util.FromJson(msg.Message, &list) {
					l.updates <- func(l *lobbyConnection) { l.listing, l.current = list.Games, lobbyConnected }
				}
			}
		})
		l.updates <- func(l *lobbyConnection) {
			l.connection = connection
			util.MustSend(connection.Login(name))
		}
	}()
	return l
}

func (l *lobbyConnection) poll() {
	for {
		select {
		case update := <-l.updates:
			update(l)
		default:
			return
		}
	}
}

func (l *lobbyConnection) state() lobbyState {
	l.poll()
	return l.current
}

func (l *lobbyConnection) games() []game.GameListing {
	return l.listing
}

func (l *lobbyConnection) errorMessage() string {
	return l.err
}

// detach hands the connection over to the match; the lobby stops listening.
func (l *lobbyConnection) detach() {
	l.detached = true
}

func (l *lobbyConnection) close() {
	if l.connection != nil && !l.detached {
		_ = l.connection.Close()
	}
}

func (m *Menu) editorScreen() {
	left := float32(m.width)/2 - 260
	y := float32(140)
	m.ui.Label(left, y, "Map editor", 2, uiTextHighlight)
	y += 50
	open := func(mapName string) {
		m.result = &MatchConfig{Mode: ModeEditor, EditorMap: mapName}
	}
	m.ui.Label(left, y, "Start a new map", 1, uiTextDim)
	y += 22
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for _, start := range []struct{ label, name string }{
		{"Empty field", "gen:empty:0"},
		{"Generated city", "gen:urban:random"},
		{"Generated ruins", "gen:ruins:random"},
		{"Generated outpost", "gen:outpost:random"},
	} {
		if m.ui.Button(left, y, 520, 30, start.label, true) {
			open(game.WithRandomSeed(start.name, rng))
		}
		y += 36
	}
	y += 12
	m.ui.Label(left, y, "Edit a saved map", 1, uiTextDim)
	y += 22
	for _, entry := range m.content.Maps {
		if game.IsGeneratedMap(entry.File) {
			continue
		}
		if m.ui.Button(left, y, 520, 30, entry.Name, true) {
			open(entry.File)
		}
		y += 36
	}
	if m.ui.Button(left, float32(m.height)-80, 160, 36, "Back", true) {
		m.screen = screenMain
	}
}

// SetContent replaces the content shown in the menus, e.g. after the editor saved a new map.
func (m *Menu) SetContent(content game.Content) {
	m.content = content
	if m.mapIndex >= len(content.Maps) {
		m.mapIndex = 0
	}
}
