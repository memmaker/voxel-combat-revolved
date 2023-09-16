package client

import (
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/memmaker/battleground/engine/gui"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/game"
)

func (a *BattleClient) UpdateActionbarFor(unit *Unit) {
	var actions []gui.ActionItem
	snapshot := gui.ActionItem{
		Name:         "Fire",
		TextureIndex: a.guiIcons["ranged"],
		Execute: func() {
			if !unit.CanSnapshot() {
				println("[GameStateUnit] Unit cannot snapshot anymore.")
				return
			}
			a.SwitchToBlockTarget(unit, game.NewActionShot(a.GameInstance, unit.UnitInstance))
		},
		Hotkey: glfw.KeyR,
	}
	freeAim := gui.ActionItem{
		Name:         "Free Aim",
		TextureIndex: a.guiIcons["reticule"],
		Execute: func() {
			if !unit.CanFreeAim() {
				println("[GameStateUnit] Unit cannot free aim anymore.")
				return
			}
			a.SwitchToFreeAim(unit, game.NewActionShot(a.GameInstance, unit.UnitInstance))
		},
		Hotkey: glfw.KeyF,
	}
	reloadAction := gui.ActionItem{
		Name:         "Reload",
		TextureIndex: a.guiIcons["reload"],
		Execute: func() {
			if !unit.CanAct() {
				println("[GameStateUnit] Unit cannot act anymore.")
				return
			}
			util.MustSend(a.server.ReloadAction(unit.UnitID()))
		},
		Hotkey: glfw.KeyR,
	}
	overwatch := gui.ActionItem{
		Name:         "Overwatch",
		TextureIndex: a.guiIcons["overwatch"],
		Execute: func() {
			if !unit.CanAct() {
				println("[GameStateUnit] Unit cannot act anymore.")
				return
			}
			a.SwitchToBlockTarget(unit, game.NewActionOverwatch(a.GameInstance, unit.UnitInstance))
		},
	}
	if unit.CanReload() {
		actions = append(actions, reloadAction)
	}
	for _, item := range unit.GetItems() {
		itemAction := gui.ActionItem{
			Name:         item.Definition.UniqueName,
			TextureIndex: a.guiIcons[string(item.Definition.ItemType)], // hardcoded for now
			Execute: func() {
				a.StartItemAction(unit, item)
			},
			Hotkey: glfw.KeyT, // TODO: change this to the actual hotkey
		}
		actions = append(actions, itemAction)
	}
	if unit.CanSnapshot() {
		actions = append(actions, snapshot)
	}
	if unit.CanFreeAim() {
		actions = append(actions, freeAim)
		actions = append(actions, overwatch)
	}
	always := []gui.ActionItem{
		{
			Name:         "End Turn",
			TextureIndex: a.guiIcons["next-turn"],
			Execute:      a.EndTurn,
			Hotkey:       glfw.KeyF8,
		},
	}
	actions = append(actions, always...)
	a.actionbar.SetActions(actions)
}
