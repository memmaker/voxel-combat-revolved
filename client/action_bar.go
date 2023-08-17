package client

import (
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/memmaker/battleground/engine/gui"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/game"
)

func (a *BattleClient) UpdateActionbarFor(unit *Unit) {
	var actions []gui.ActionItem
	fireWeaponActions := []gui.ActionItem{
		{
			Name:         "Fire",
			TextureIndex: 1,
			Execute: func() {
				if !unit.CanFire() {
					println("[GameStateUnit] Unit cannot fire anymore.")
					return
				}
				a.SwitchToAction(unit, game.NewActionShot(a.GameInstance))
			},
			Hotkey: glfw.KeyR,
		},
		{
			Name:         "Free Aim",
			TextureIndex: 2,
			Execute: func() {
				if !unit.CanFire() {
					println("[GameStateUnit] Unit cannot fire anymore.")
					return
				}
				a.SwitchToFreeAim(unit, game.NewActionShot(a.GameInstance))
			},
			Hotkey: glfw.KeyF,
		},
	}
	reloadAction := gui.ActionItem{
		Name:         "Reload",
		TextureIndex: 4,
		Execute: func() {
			if !unit.CanAct() {
				println("[GameStateUnit] Unit cannot act anymore.")
				return
			}
			util.MustSend(a.server.ReloadAction(unit.UnitID()))
		},
		Hotkey: glfw.KeyR,
	}
	if unit.CanReload() {
		actions = append(actions, reloadAction)
	}
	if unit.CanFire() {
		actions = append(actions, fireWeaponActions...)
	}
	always := []gui.ActionItem{
		{
			Name:         "End Turn",
			TextureIndex: 3,
			Execute:      a.EndTurn,
			Hotkey:       glfw.KeyF8,
		},
	}
	actions = append(actions, always...)
	a.actionbar.SetActions(actions)
}
