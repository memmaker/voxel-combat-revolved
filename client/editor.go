package client

import (
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/memmaker/battleground/game"
)

// RunEditor opens the map editor on a map (a saved map name or a generated "gen:<style>:<seed>")
// and returns when the player exits or starts a test match.
func RunEditor(window *glfw.Window, settings ClientSettings, mapName string) (EditorResult, bool) {
	// the editor needs no server: messages go into a pipe nobody reads
	clientEnd, _ := game.MemPipe()
	connection := game.NewServerConnectionFromConn(clientEnd, nil)
	info := game.GameStartedMessage{GameID: "editor", MapFile: mapName, MissionDetails: game.NewRandomDeathmatch()}
	editor := NewBattleGame(window, connection, info, settings)
	editor.LoadMapBlocks()
	editor.stateStack = []GameState{NewEditorState(editor)}
	editor.state().Init(false)
	editor.Run()
	return editor.editorResult, !window.ShouldClose()
}
