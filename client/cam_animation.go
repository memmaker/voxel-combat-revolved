package client

import "github.com/memmaker/battleground/engine/util"

// handleCameraAnimation returns true if currently animating
func (a *BattleClient) handleCameraAnimation(elapsed float64) bool {
	if a.cameraAnimation == nil {
		return false
	}
	if a.cameraAnimation.IsFinished() {
		a.cameraAnimation = nil
		return false
	}
	a.cameraAnimation.Update(elapsed)
	return true
}

func (a *BattleClient) StartCameraTransition(start, end util.Camera, duration float64) {
	if a.settings.EnableCameraAnimations {
		a.cameraAnimation = util.NewCameraTransition(start, end, duration, a.WindowWidth, a.WindowHeight)
	}
}
func (a *BattleClient) StartCameraLookAnimation(start util.Transform, end util.Camera, duration float64) {
	if a.settings.EnableCameraAnimations {
		a.cameraAnimation = util.NewCameraLookAnimation(start, end, duration, a.WindowWidth, a.WindowHeight)
	}
}
