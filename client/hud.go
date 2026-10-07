package client

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/glhf"
	"github.com/memmaker/battleground/engine/gui"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/game"
)

// palette indices for the HUD rect batch
const (
	hudBlack = iota
	hudDarkGrey
	hudHealthGood
	hudHealthHurt
	hudHealthCritical
	hudFriendly
	hudEnemy
	hudAPFull
	hudAPEmpty
	hudPanel
)

var hudPalette = []mgl32.Vec3{
	{0, 0, 0},
	{0.15, 0.15, 0.15},
	{0.18, 0.80, 0.30},
	{0.95, 0.75, 0.10},
	{0.90, 0.15, 0.10},
	game.ColorTechTeal.Vec3(),
	game.ColorNegativeRed.Vec3(),
	{0.36, 0.71, 1.0},
	{0.12, 0.20, 0.30},
	{0.05, 0.08, 0.10},
}

type floatingText struct {
	label     *util.BitmapFontMesh
	world     mgl32.Vec3
	age       float64
	lifetime  float64
	riseSpeed float32
}

// HUD draws all screen-space information about the battle: unit bars, the selected unit panel,
// the turn banner and floating combat text.
type HUD struct {
	engine       *BattleClient
	rects        *gui.RectBatch
	shader       *glhf.Shader
	newLabel     func() *util.BitmapFontMesh
	unitPanel    *util.BitmapFontMesh
	turnBanner   *util.BitmapFontMesh
	floating     []*floatingText
	panelText    string
	bannerText   string
	round        int
	isMyTurn     bool
	panelBounds  [4]float32
	hidden       bool
	labelCache   map[string]*util.BitmapFontMesh
	frameLabels  []*util.BitmapFontMesh
	unitList     []unitListEntry
	turnDeadline time.Time // zero: no time limit
	charW, charH float32
}

func NewHUD(engine *BattleClient, shader *glhf.Shader, newLabel func() *util.BitmapFontMesh, charWidth, charHeight int) *HUD {
	h := &HUD{
		engine:     engine,
		shader:     shader,
		rects:      gui.NewRectBatch(shader, hudPalette),
		newLabel:   newLabel,
		unitPanel:  newLabel(),
		turnBanner: newLabel(),
		charW:      float32(charWidth),
		charH:      float32(charHeight),
	}
	h.unitPanel.SetTintColor(game.ColorTechTeal.Vec3())
	h.turnBanner.SetTintColor(mgl32.Vec3{1, 1, 1})
	return h
}

func (h *HUD) SetHidden(hidden bool) {
	h.hidden = hidden
}

// OnTurnStarted is called whenever the active player changes.
func (h *HUD) OnTurnStarted(isMyTurn bool, turnSeconds int) {
	h.turnDeadline = time.Time{}
	if turnSeconds > 0 {
		h.turnDeadline = time.Now().Add(time.Duration(turnSeconds) * time.Second)
	}
	if isMyTurn {
		h.round++
	}
	h.isMyTurn = isMyTurn
}

func (h *HUD) Round() int {
	return h.round
}

// AddFloatingText shows a short text at a world position that rises and fades.
func (h *HUD) AddFloatingText(text string, world mgl32.Vec3, color mgl32.Vec3) {
	label := h.newLabel()
	label.SetScale(2)
	label.SetCenterOrigin(true)
	label.SetTintColor(color)
	label.SetText(text)
	h.floating = append(h.floating, &floatingText{label: label, world: world, lifetime: 1.6, riseSpeed: 0.8})
}

func (h *HUD) Update(elapsed float64) {
	for i := len(h.floating) - 1; i >= 0; i-- {
		f := h.floating[i]
		f.age += elapsed
		f.world = f.world.Add(mgl32.Vec3{0, f.riseSpeed * float32(elapsed), 0})
		if f.age >= f.lifetime {
			h.floating = append(h.floating[:i], h.floating[i+1:]...)
		}
	}
}

// worldToScreen projects a world position into window pixel coordinates (origin top-left).
func (h *HUD) worldToScreen(cam util.Camera, world mgl32.Vec3) (mgl32.Vec2, bool) {
	clip := cam.GetProjectionViewMatrix().Mul4x1(world.Vec4(1))
	if clip.W() <= 0.01 {
		return mgl32.Vec2{}, false
	}
	ndc := clip.Vec3().Mul(1 / clip.W())
	if ndc.X() < -1.1 || ndc.X() > 1.1 || ndc.Y() < -1.1 || ndc.Y() > 1.1 {
		return mgl32.Vec2{}, false
	}
	width, height := float32(h.engine.WindowWidth), float32(h.engine.WindowHeight)
	return mgl32.Vec2{(ndc.X() + 1) * 0.5 * width, (1 - ndc.Y()) * 0.5 * height}, true
}

func healthColor(fraction float32) int {
	if fraction > 0.6 {
		return hudHealthGood
	} else if fraction > 0.3 {
		return hudHealthHurt
	}
	return hudHealthCritical
}

func (h *HUD) Draw(cam util.Camera, firstPerson bool) {
	if h.hidden {
		return
	}
	h.rects.Clear()
	if !firstPerson {
		h.queueUnitBars(cam)
	}
	h.queuePanel()
	h.queueTurnBanner()
	h.queueUnitList()

	hotkeyLabels := append(h.queueHotkeys(), h.frameLabels...)
	h.frameLabels = h.frameLabels[:0]

	gl.Disable(gl.DEPTH_TEST)
	h.rects.Draw()
	h.unitPanel.Draw()
	h.turnBanner.Draw()
	for _, label := range hotkeyLabels {
		label.Draw()
	}
	for _, f := range h.floating {
		screenPos, onScreen := h.worldToScreen(cam, f.world)
		if !onScreen {
			continue
		}
		f.label.SetPosition(screenPos)
		f.label.Draw()
	}
	gl.Enable(gl.DEPTH_TEST)
}

func (h *HUD) queueUnitBars(cam util.Camera) {
	myID := h.engine.GetControllingUserID()
	for _, unit := range h.engine.GetAllClientUnits() {
		if !unit.IsActive() || !h.engine.UnitIsVisibleToPlayer(myID, unit.UnitID()) {
			continue
		}
		head := unit.GetPosition().Add(mgl32.Vec3{0, 2.4, 0})
		screenPos, onScreen := h.worldToScreen(cam, head)
		if !onScreen {
			continue
		}
		maxHealth := unit.Definition.CoreStats.Health
		health := max(unit.Health, 0)
		// one segment per health point, so damage reads at a glance
		segmentWidth := float32(4)
		barWidth := float32(maxHealth)*segmentWidth + 2
		barHeight := float32(7)
		x := screenPos.X() - barWidth/2
		y := screenPos.Y()

		ownerColor := hudEnemy
		if unit.ControlledBy() == myID {
			ownerColor = hudFriendly
		}
		fraction := float32(health) / float32(maxHealth)
		h.rects.Add(x-2, y-2, barWidth+4, barHeight+4, ownerColor)
		h.rects.Add(x, y, barWidth, barHeight, hudBlack)
		fill := healthColor(fraction)
		for i := 0; i < health; i++ {
			h.rects.Add(x+1+float32(i)*segmentWidth, y+1, segmentWidth-1, barHeight-2, fill)
		}

		if unit.ControlledBy() == myID {
			h.queueAPPips(unit.UnitInstance, screenPos.X(), y+barHeight+3)
			if unit.Reaction != game.ReactionNone {
				text := string(unit.Reaction)
				h.frameLabels = append(h.frameLabels, h.cachedLabelKeyed(fmt.Sprintf("%s#%d", text, unit.UnitID()), text, mgl32.Vec2{screenPos.X() - float32(len(text))*h.charW/2, y - h.charH - 4}))
			}
		}
	}
}

func (h *HUD) queueAPPips(unit *game.UnitInstance, centerX, y float32) {
	maxAP := int(unit.Definition.CoreStats.MaxActionPoints)
	currentAP := unit.GetIntegerAP()
	pipWidth, pipHeight, gap := float32(8), float32(4), float32(2)
	totalWidth := float32(maxAP)*(pipWidth+gap) - gap
	x := centerX - totalWidth/2
	for i := 0; i < maxAP; i++ {
		color := hudAPEmpty
		if i < currentAP {
			color = hudAPFull
		}
		h.rects.Add(x+float32(i)*(pipWidth+gap), y, pipWidth, pipHeight, color)
	}
}

func (h *HUD) queuePanel() {
	unit := h.engine.selectedUnit
	if unit == nil || !unit.IsActive() || !h.engine.IsMyUnit(unit.UnitID()) {
		h.panelText = ""
		return
	}
	weapon := unit.GetWeapon()
	lines := []string{
		unit.GetName(),
		fmt.Sprintf("HP   %d/%d", max(unit.Health, 0), unit.Definition.CoreStats.Health),
		fmt.Sprintf("AP   %d/%d", unit.GetIntegerAP(), int(unit.Definition.CoreStats.MaxActionPoints)),
	}
	if weapon != nil {
		lines = append(lines,
			weapon.Definition.UniqueName,
			fmt.Sprintf("AMMO %d/%d", weapon.AmmoCount, weapon.Definition.MagazineSize),
			fmt.Sprintf("SHOT %d AP  RELOAD %d AP", weapon.Definition.BaseAPForShot, weapon.Definition.BaseAPForReload),
		)
	}
	if len(unit.GetItems()) > 0 {
		names := make([]string, len(unit.GetItems()))
		for i, item := range unit.GetItems() {
			names[i] = item.Definition.UniqueName
		}
		lines = append(lines, "ITEMS "+strings.Join(names, ", "))
	}
	if unit.MovementPenalty > 0 || unit.AimPenalty > 0 {
		lines = append(lines, fmt.Sprintf("WOUNDS move +%.1f AP  aim -%d%%", unit.MovementPenalty, int(math.Round(unit.AimPenalty*100))))
	}

	text := strings.Join(lines, "\n")
	textWidth, textHeight := h.unitPanel.CalculateSize(lines)
	padding := float32(8)
	width := float32(textWidth) + padding*2
	height := float32(textHeight) + padding*2
	x := padding
	y := float32(h.engine.WindowHeight) - height - padding
	if text != h.panelText || h.panelBounds != [4]float32{x, y, width, height} {
		h.panelText = text
		h.panelBounds = [4]float32{x, y, width, height}
		h.unitPanel.SetPosition(mgl32.Vec2{x + padding, y + padding})
		h.unitPanel.SetMultilineText(lines)
	}
	h.rects.Add(x, y, width, height, hudPanel)
	h.rects.AddFrame(x, y, width, height, 1, hudFriendly)
}

// queueHotkeys labels the action bar slots with their number keys and shows the fast-forward state.
func (h *HUD) queueHotkeys() []*util.BitmapFontMesh {
	var labels []*util.BitmapFontMesh
	bar := h.engine.actionbar
	for i := 0; i < bar.ItemCount() && i < 9; i++ {
		bounds := bar.ItemBounds(i)
		h.rects.Add(bounds.TopLeft.X(), bounds.TopLeft.Y()-16, 14, 15, hudPanel)
		labels = append(labels, h.cachedLabel(fmt.Sprintf("%d", i+1), mgl32.Vec2{bounds.TopLeft.X() + 3, bounds.TopLeft.Y() - 15}))
	}
	if h.engine.SpeedMultiplier > 1 {
		text := fmt.Sprintf("FAST FORWARD x%.0f (Space)", h.engine.SpeedMultiplier)
		labels = append(labels, h.cachedLabel(text, mgl32.Vec2{float32(h.engine.WindowWidth) - float32(len(text))*h.charW - 12, 12}))
	}
	labels = append(labels, h.cachedLabel("F1 Help", mgl32.Vec2{float32(h.engine.WindowWidth) - 7*h.charW - 12, float32(h.engine.WindowHeight) - h.charH - 8}))
	return labels
}

func (h *HUD) cachedLabel(text string, position mgl32.Vec2) *util.BitmapFontMesh {
	return h.cachedLabelKeyed(text, text, position)
}

// cachedLabelKeyed caches one label per key, so the same text can be shown at several places.
func (h *HUD) cachedLabelKeyed(key, text string, position mgl32.Vec2) *util.BitmapFontMesh {
	if h.labelCache == nil {
		h.labelCache = make(map[string]*util.BitmapFontMesh)
	}
	label, cached := h.labelCache[key]
	if !cached {
		label = h.newLabel()
		label.SetTintColor(game.ColorTechTeal.Vec3())
		label.SetText(text)
		h.labelCache[key] = label
	}
	label.SetPosition(position)
	return label
}

func (h *HUD) queueTurnBanner() {
	if h.round == 0 {
		return
	}
	turnText := "ENEMY TURN"
	if h.isMyTurn {
		turnText = "YOUR TURN"
	}
	if !h.turnDeadline.IsZero() {
		left := max(0, int(time.Until(h.turnDeadline).Seconds()))
		turnText += fmt.Sprintf("  %d:%02d", left/60, left%60)
	}
	text := fmt.Sprintf("ROUND %d  -  %s", h.round, turnText)
	details := h.engine.GetMissionDetails()
	if details != nil && details.TurnLimit > 0 {
		text = fmt.Sprintf("ROUND %d/%d  -  %s", h.round, details.TurnLimit/2, turnText)
	}
	textWidth, textHeight := h.turnBanner.CalculateSize([]string{text})
	width := float32(textWidth) + 24
	height := float32(textHeight) + 12
	x := (float32(h.engine.WindowWidth) - width) / 2
	y := float32(8)
	if text != h.bannerText {
		h.bannerText = text
		h.turnBanner.SetText(text)
	}
	h.turnBanner.SetPosition(mgl32.Vec2{x + 12, y + 6})
	border := hudEnemy
	if h.isMyTurn {
		border = hudFriendly
	}
	h.rects.Add(x, y, width, height, hudPanel)
	h.rects.AddFrame(x, y, width, height, 1, border)
}

type unitListEntry struct {
	unitID uint64
	x, y   float32
	w, h   float32
}

// queueUnitList shows one card per own soldier at the right edge: name, health and AP.
// Clicking a card selects the soldier (see UnitAt).
func (h *HUD) queueUnitList() {
	h.unitList = h.unitList[:0]
	width, height := float32(150), float32(34)
	x := float32(h.engine.WindowWidth) - width - 8
	y := float32(48)
	for _, unit := range h.engine.GetMyUnits() {
		if !unit.IsActive() {
			continue
		}
		border := hudDarkGrey
		if h.engine.selectedUnit != nil && h.engine.selectedUnit.UnitID() == unit.UnitID() {
			border = hudFriendly
		}
		h.rects.Add(x, y, width, height, hudPanel)
		h.rects.AddFrame(x, y, width, height, 1, border)
		color := game.ColorTechTeal.Vec3()
		if !unit.CanAct() {
			color = mgl32.Vec3{0.4, 0.5, 0.5} // done for this turn
		}
		label := h.cachedLabelKeyed(fmt.Sprintf("list#%d#%s#%t", unit.UnitID(), unit.GetName(), unit.CanAct()), unit.GetName(), mgl32.Vec2{x + 6, y + 4})
		label.SetTintColor(color)
		h.frameLabels = append(h.frameLabels, label)
		fraction := float32(max(unit.Health, 0)) / float32(unit.Definition.CoreStats.Health)
		h.rects.AddBar(x+6, y+21, width-60, 7, fraction, hudDarkGrey, healthColor(fraction))
		h.queueAPPips(unit, x+width-28, y+22)
		h.unitList = append(h.unitList, unitListEntry{unitID: unit.UnitID(), x: x, y: y, w: width, h: height})
		y += height + 4
	}
}

// UnitAt returns the soldier whose card is at a screen position.
func (h *HUD) UnitAt(x, y float64) (uint64, bool) {
	for _, entry := range h.unitList {
		if float32(x) >= entry.x && float32(x) <= entry.x+entry.w && float32(y) >= entry.y && float32(y) <= entry.y+entry.h {
			return entry.unitID, true
		}
	}
	return 0, false
}
