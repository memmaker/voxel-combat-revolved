package main

import (
	"github.com/memmaker/battleground/server"
)

// NewBattleServer creates a server with the content from assets/content.
func NewBattleServer() *server.BattleServer {
	battleServer, err := server.NewBattleServerFromAssets()
	if err != nil {
		panic(err)
	}
	return battleServer
}
