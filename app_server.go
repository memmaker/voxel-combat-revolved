//go:build server
// +build server

package main

func main() {
	server := NewBattleServer()
	server.ListenTCP("0.0.0.0:9999")
}
