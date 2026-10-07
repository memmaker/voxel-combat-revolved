package server

import "github.com/memmaker/battleground/game"

func (b *BattleServer) AddContent(content game.Content) {
	for _, entry := range content.Maps {
		b.AddMap(entry.Name, entry.File)
	}
	for _, faction := range content.Factions {
		b.AddFaction(faction)
	}
	for _, weapon := range content.Weapons {
		b.AddWeapon(weapon)
	}
	for _, item := range content.Items {
		b.AddItem(item)
	}
	b.content = content
}

// GetContent returns the content this server was configured with (used by menus and the AI).
func (b *BattleServer) GetContent() game.Content {
	return b.content
}

// NewBattleServerFromAssets creates a server and loads the content files from assets/content.
func NewBattleServerFromAssets() (*BattleServer, error) {
	content, err := game.LoadContent()
	if err != nil {
		return nil, err
	}
	battleServer := NewBattleServer()
	battleServer.AddContent(content)
	return battleServer, nil
}
