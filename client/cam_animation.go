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

func (a *BattleClient) StartCameraAnimation(start, end util.Transform, duration float64) {
	if a.settings.EnableCameraAnimations {
		a.cameraAnimation = util.NewCameraAnimation(start, end, duration, a.WindowWidth, a.WindowHeight)
	}
}
