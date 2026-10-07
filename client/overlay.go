package client

import (
	"fmt"

	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/game"
)

type MatchOutcome int

const (
	OutcomeMenu MatchOutcome = iota
	OutcomeRematch
	OutcomeQuit
)

type overlayKind int

const (
	overlayNone overlayKind = iota
	overlayPause
	overlayGameOver
	overlayHelp
)

// MatchStats is the local player's view of how the match went.
type MatchStats struct {
	ShotsFired  int
	Hits        int
	DamageDealt int
	DamageTaken int
	Kills       int
	Losses      int
}

func (s MatchStats) Accuracy() float64 {
	if s.ShotsFired == 0 {
		return 0
	}
	return float64(s.Hits) / float64(s.ShotsFired)
}

func (a *BattleClient) recordShots(msg game.VisualRangedAttack) {
	if !a.IsMyUnit(msg.Attacker) {
		return
	}
	for _, projectile := range msg.Projectiles {
		a.stats.ShotsFired++
		if projectile.UnitHit >= 0 && !a.IsMyUnit(uint64(projectile.UnitHit)) {
			a.stats.Hits++
		}
	}
}

func (a *BattleClient) recordDamage(unit *game.UnitInstance, damage int, lethal bool) {
	if a.IsMyUnit(unit.UnitID()) {
		a.stats.DamageTaken += damage
		if lethal {
			a.stats.Losses++
		}
	} else {
		a.stats.DamageDealt += damage
		if lethal {
			a.stats.Kills++
		}
	}
}

// Outcome is what the player chose after the match (or Quit when the window was closed).
func (a *BattleClient) Outcome() MatchOutcome {
	if a.Window.ShouldClose() {
		return OutcomeQuit
	}
	return a.outcome
}

func (a *BattleClient) finishMatch(outcome MatchOutcome) {
	a.outcome = outcome
	a.StopRequested = true
}

func (a *BattleClient) openPauseMenu() {
	if a.overlay != overlayNone {
		return
	}
	a.overlay = overlayPause
	a.freeMouse()
}

func (a *BattleClient) closeOverlay() {
	a.overlay = overlayNone
	if a.cameraIsFirstPerson {
		a.captureMouse()
	}
}

func (a *BattleClient) drawOverlay() {
	if a.overlay == overlayNone {
		return
	}
	a.ui.Begin(a.WindowWidth, a.WindowHeight, a.lastMousePosX, a.lastMousePosY, a.uiClicked)
	a.uiClicked = false
	switch a.overlay {
	case overlayPause:
		a.pauseOverlay()
	case overlayGameOver:
		a.gameOverOverlay()
	case overlayHelp:
		a.helpOverlay()
	}
	a.ui.Draw()
	a.guiShader.SetUniformAttr(0, util.Get2DPixelCoordOrthographicProjectionMatrix(a.WindowWidth, a.WindowHeight))
}

func (a *BattleClient) pauseOverlay() {
	a.ui.Shade()
	width, height := float32(300), float32(260)
	x := (float32(a.WindowWidth) - width) / 2
	y := (float32(a.WindowHeight) - height) / 2
	a.ui.Panel(x, y, width, height)
	a.ui.CenteredLabel(x+width/2, y+18, "PAUSED", 2, uiTextHighlight)
	buttonX, buttonWidth, buttonHeight := x+30, width-60, float32(34)
	buttonY := y + 70
	if a.ui.Button(buttonX, buttonY, buttonWidth, buttonHeight, "Resume", true) {
		a.closeOverlay()
	}
	buttonY += buttonHeight + 12
	if a.ui.Button(buttonX, buttonY, buttonWidth, buttonHeight, "Surrender & main menu", true) {
		_ = a.server.Close() // the server awards the match to the opponent
		a.finishMatch(OutcomeMenu)
	}
	buttonY += buttonHeight + 12
	if a.ui.Button(buttonX, buttonY, buttonWidth, buttonHeight, "Quit game", true) {
		_ = a.server.Close()
		a.finishMatch(OutcomeQuit)
	}
	a.ui.CenteredLabel(x+width/2, y+height-24, "Esc to resume", 1, uiTextDim)
}

func (a *BattleClient) gameOverOverlay() {
	a.ui.Shade()
	width, height := float32(420), float32(380)
	x := (float32(a.WindowWidth) - width) / 2
	y := (float32(a.WindowHeight) - height) / 2
	a.ui.Panel(x, y, width, height)
	title, color := "DEFEAT", game.ColorNegativeRed.Vec3()
	if a.wonMatch {
		title, color = "VICTORY", uiTextHighlight
	}
	a.ui.CenteredLabel(x+width/2, y+20, title, 3, color)
	rows := [][2]string{
		{"Rounds played", fmt.Sprintf("%d", a.hud.Round())},
		{"Enemies killed", fmt.Sprintf("%d", a.stats.Kills)},
		{"Soldiers lost", fmt.Sprintf("%d", a.stats.Losses)},
		{"Damage dealt", fmt.Sprintf("%d", a.stats.DamageDealt)},
		{"Damage taken", fmt.Sprintf("%d", a.stats.DamageTaken)},
		{"Projectiles fired", fmt.Sprintf("%d", a.stats.ShotsFired)},
		{"Hit rate", percent(a.stats.Accuracy())},
	}
	rowY := y + 80
	for _, row := range rows {
		a.ui.Label(x+40, rowY, row[0], 1, uiTextDim)
		valueWidth, _ := a.ui.TextSize(row[1], 1)
		a.ui.Label(x+width-40-valueWidth, rowY, row[1], 1, uiTextColor)
		rowY += 22
	}
	buttonY := y + height - 60
	buttonWidth := (width - 80) / 3
	if a.ui.Button(x+20, buttonY, buttonWidth, 36, "Rematch", a.canRematch) {
		a.finishMatch(OutcomeRematch)
	}
	if a.ui.Button(x+40+buttonWidth, buttonY, buttonWidth, 36, "Main menu", true) {
		a.finishMatch(OutcomeMenu)
	}
	if a.ui.Button(x+60+buttonWidth*2, buttonY, buttonWidth, 36, "Quit", true) {
		a.finishMatch(OutcomeQuit)
	}
}

// SetRematchAvailable enables the rematch button (only for matches against the AI).
func (a *BattleClient) SetRematchAvailable(available bool) {
	a.canRematch = available
}

func (a *BattleClient) showGameOver(youWon bool) {
	a.wonMatch = youWon
	if youWon {
		a.audio.Play(SoundVictory)
	} else {
		a.audio.Play(SoundDefeat)
	}
	a.overlay = overlayGameOver
	a.hud.SetHidden(false)
	a.freeMouse()
}

func (a *BattleClient) toggleFastForward() {
	if a.SpeedMultiplier > 1 {
		a.SpeedMultiplier = 1
	} else {
		a.SpeedMultiplier = 3
	}
}

var helpSections = []struct {
	title string
	lines []string
}{
	{"Camera", []string{
		"W A S D   pan          Q / E   rotate",
		"Mouse wheel   zoom     Z / C   show fewer / more floors",
	}},
	{"Your turn", []string{
		"Click a highlighted block to move. Blue: you can still fire",
		"afterwards, orange: the move uses up your shot.",
		"1-9   actions on the bar       Tab   next soldier",
		"Enter / F8   end turn          Space   fast forward",
	}},
	{"Shooting", []string{
		"Fire: quick shot at a visible enemy (cheaper).",
		"Free aim: aim yourself in first person, +1 AP. Every bullet",
		"is simulated: aim at the head (double damage) or around cover.",
		"In free aim: mouse to aim, click to fire, N / M zoom, Esc back.",
		"Weapon hits cost no health but reduce accuracy.",
		"Arm wounds reduce accuracy, leg wounds slow movement.",
	}},
	{"Tactics", []string{
		"Pressure: enemies close by and in sight make snipers less accurate.",
		"Overwatch: pick blocks to watch; enemies stepping on them get shot",
		"(-20% accuracy, +10% damage).",
		"Grenades blow holes in walls; smoke blocks lines of sight.",
	}},
	{"Reactions (secret until they trigger)", []string{
		"Dodge: bullets that would hit you miss half the time. Useless vs grenades.",
		"Brace: -1 damage per bullet, half damage from explosions.",
		"Parry: melee attacks fail and the attacker takes 3 damage.",
		"Melee (2 AP): strike an adjacent enemy, 85% for 4 damage.",
		"Guess what the enemy will do: shoot, throw or advance.",
	}},
}

func (a *BattleClient) helpOverlay() {
	a.ui.Shade()
	width := min(float32(680), float32(a.WindowWidth)-40)
	height := min(float32(640), float32(a.WindowHeight)-20)
	x := (float32(a.WindowWidth) - width) / 2
	y := (float32(a.WindowHeight) - height) / 2
	a.ui.Panel(x, y, width, height)
	a.ui.CenteredLabel(x+width/2, y+14, "HOW TO PLAY", 2, uiTextHighlight)
	rowY := y + 56
	for _, section := range helpSections {
		a.ui.Label(x+24, rowY, section.title, 1, uiTextHighlight)
		rowY += 20
		for _, line := range section.lines {
			a.ui.Label(x+36, rowY, line, 1, uiTextColor)
			rowY += 17
		}
		rowY += 10
	}
	if a.ui.Button(x+width/2-70, y+height-46, 140, 32, "Close (F1)", true) {
		a.closeOverlay()
	}
}
