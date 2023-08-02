//go:build server
// +build server

package main

func main() {
	NewBattleServer().ListenTCP()
}
