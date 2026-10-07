package client

import (
	"fmt"
	"os"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
	"github.com/memmaker/battleground/game"
)

// autoPlayer lets the AI play the local player's side in the graphical client (BT_AUTOPLAY=1).
// It is used to watch AI matches and to exercise the renderer without manual input.
type autoPlayer struct {
	engine   *BattleClient
	brain    *game.AIBrain
	isMyTurn bool
	// readyForStep is set once the result of the last action arrived; the next step is taken
	// when all animations caused by it have finished.
	readyForStep bool
	blockedFor   float64
}

func newAutoPlayerFromEnv(engine *BattleClient) *autoPlayer {
	if os.Getenv("BT_AUTOPLAY") == "" {
		return nil
	}
	return &autoPlayer{engine: engine}
}

func (p *autoPlayer) onTurnStarted(isMyTurn bool) {
	if p.brain == nil {
		p.brain = game.NewAIBrain(p.engine.GameInstance, p.engine.GetControllingUserID(), int(p.engine.GetSpawnIndex()), game.AIDifficultyFromEnv())
	}
	p.isMyTurn = isMyTurn
	if isMyTurn {
		p.brain.StartTurn()
		p.readyForStep = true
	}
}

func (p *autoPlayer) onOwnActionFinished() {
	p.readyForStep = true
}

func (p *autoPlayer) onActionFailed() {
	if p.brain != nil {
		p.brain.OnActionFailed()
	}
	p.readyForStep = true
}

func (p *autoPlayer) onGameOver() {
	p.isMyTurn = false
}

func (p *autoPlayer) update(elapsed float64) {
	if !p.isMyTurn || !p.readyForStep {
		return
	}
	e := p.engine
	if e.currentlyMovingUnits || len(e.flyingObjects) > 0 || e.scriptedAnimation.Running() || len(e.conditionQueue) > 0 || len(e.serverChannel) > 0 {
		p.blockedFor += elapsed
		if p.blockedFor > 5 {
			p.blockedFor = 0
			states := ""
			for _, unit := range e.GetAllClientUnits() {
				states += fmt.Sprintf(" %s:%s@%s", unit.GetName(), unit.state.GetName().ToString(), unit.GetBlockPosition().ToString())
			}
			println(fmt.Sprintf("[AutoPlay] waiting: moving=%t flying=%d script=%t conditions=%d network=%d units:%s", e.currentlyMovingUnits, len(e.flyingObjects), e.scriptedAnimation.Running(), len(e.conditionQueue), len(e.serverChannel), states))
		}
		return
	}
	p.blockedFor = 0
	p.readyForStep = false
	p.brain.Step(p)
}

// game.AICommander

func (p *autoPlayer) Move(unitID uint64, target voxel.Int3) {
	util.MustSend(p.engine.server.TargetedUnitAction(unitID, "Move", []voxel.Int3{target}))
}

func (p *autoPlayer) Shoot(unitID uint64, target voxel.Int3) {
	util.MustSend(p.engine.server.TargetedUnitAction(unitID, "Shot", []voxel.Int3{target}))
}

func (p *autoPlayer) Throw(unitID uint64, itemName string, target mgl32.Vec3) {
	util.MustSend(p.engine.server.ThrownUnitAction(unitID, "Throw", itemName, []mgl32.Vec3{target}))
}

func (p *autoPlayer) Overwatch(unitID uint64, targets []voxel.Int3) {
	util.MustSend(p.engine.server.TargetedUnitAction(unitID, "Overwatch", targets))
}

func (p *autoPlayer) Heal(unitID uint64, target voxel.Int3) {
	util.MustSend(p.engine.server.TargetedUnitAction(unitID, "Heal", []voxel.Int3{target}))
}

func (p *autoPlayer) Melee(unitID uint64, target voxel.Int3) {
	util.MustSend(p.engine.server.TargetedUnitAction(unitID, "Melee", []voxel.Int3{target}))
}

func (p *autoPlayer) React(unitID uint64, reaction game.Reaction) {
	util.MustSend(p.engine.server.TargetedUnitAction(unitID, string(reaction), nil))
}

func (p *autoPlayer) Crouch(unitID uint64) {
	util.MustSend(p.engine.server.TargetedUnitAction(unitID, "Crouch", nil))
}

func (p *autoPlayer) Reload(unitID uint64) {
	util.MustSend(p.engine.server.ReloadAction(unitID))
}

func (p *autoPlayer) EndTurn() {
	p.isMyTurn = false
	p.engine.EndTurn()
}
