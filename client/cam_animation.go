package client

import "github.com/memmaker/battleground/engine/util"

// handleCameraTransition returns true if currently animating
func (a *BattleClient) handleCameraTransition(elapsed float64) bool {
    if a.camTransition == nil {
		return false
	}
    if a.camTransition.IsFinished() {
        a.camTransition = nil
		return false
	}
    a.camTransition.Update(elapsed)
	return true
}

func (a *BattleClient) StartCameraTransition(start, end util.Camera, duration float64) {
	if a.settings.EnableCameraAnimations {
        a.camTransition = util.NewCameraTransition(start, end, duration, a.WindowWidth, a.WindowHeight)
	}
}
func (a *BattleClient) StartCameraLookAnimation(start util.Transform, end util.Camera, duration float64) {
	if a.settings.EnableCameraAnimations {
        a.camTransition = util.NewCameraLookAnimation(start, end, duration, a.WindowWidth, a.WindowHeight)
	}
}
