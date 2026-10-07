package server

import (
	"fmt"
	"time"

	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/game"
)

// ReconnectGracePeriod is how long a dropped player's seat is kept before the opponent wins.
const ReconnectGracePeriod = 60 * time.Second

type offlinePlayer struct {
	gameID string
	name   string
	since  time.Time
}

// resolvePlayer maps the id of a new connection to the player it took over after a rejoin.
func (b *BattleServer) resolvePlayer(connectionID uint64) uint64 {
	if playerID, isAlias := b.aliases[connectionID]; isAlias {
		return playerID
	}
	return connectionID
}

func (b *BattleServer) otherPlayerConnected(gameInstance *game.GameInstance, userID uint64) bool {
	for _, playerID := range gameInstance.GetPlayerIDs() {
		if playerID == userID {
			continue
		}
		if _, connected := b.connectedClients[playerID]; connected && b.offline[playerID] == nil {
			return true
		}
	}
	return false
}

// holdSeat keeps a disconnected player in the match for the grace period.
func (b *BattleServer) holdSeat(gameInstance *game.GameInstance, user *UserConnection) {
	gameID := gameInstance.GetID()
	b.offline[user.id] = &offlinePlayer{gameID: gameID, name: user.name, since: time.Now()}
	user.isReady = false
	util.LogGameInfo(fmt.Sprintf("[BattleServer] %s dropped out of '%s', holding the seat", user.name, gameID))
	b.notifyOthers(gameInstance, user.id, game.PlayerConnectionMessage{Name: user.name, Connected: false, Seconds: int(ReconnectGracePeriod.Seconds())})

	playerID := user.id
	time.AfterFunc(ReconnectGracePeriod, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		info, stillOffline := b.offline[playerID]
		if !stillOffline || info.gameID != gameID {
			return
		}
		delete(b.offline, playerID)
		delete(b.connectedClients, playerID)
		running, exists := b.runningGames[gameID]
		if !exists || running != gameInstance {
			return
		}
		for _, other := range gameInstance.GetPlayerIDs() {
			if other != playerID {
				b.SendGameOver(gameInstance, other)
				return
			}
		}
	})
}

func (b *BattleServer) notifyOthers(gameInstance *game.GameInstance, userID uint64, message game.Message) {
	for _, playerID := range gameInstance.GetPlayerIDs() {
		if playerID == userID || b.offline[playerID] != nil {
			continue
		}
		if user, connected := b.connectedClients[playerID]; connected {
			b.respondWithMessage(user, message)
		}
	}
}

// tryRejoin lets a player with the same name take back a held seat. It returns true when it did.
func (b *BattleServer) tryRejoin(connectionID uint64, user *UserConnection, gameInstance *game.GameInstance) bool {
	for playerID, info := range b.offline {
		if info.gameID != gameInstance.GetID() || info.name != user.name {
			continue
		}
		seat := b.connectedClients[playerID]
		seat.raw = user.raw
		seat.isReady = false
		delete(b.offline, playerID)
		delete(b.connectedClients, connectionID)
		b.aliases[connectionID] = playerID
		util.LogGameInfo(fmt.Sprintf("[BattleServer] %s rejoined '%s'", info.name, gameInstance.GetID()))

		b.respond(seat, "JoinGameResponse", game.ActionResponse{Success: true, Message: "Rejoined the match"})
		state := b.gameStateFor(gameInstance, playerID)
		state.Rejoin = true
		state.DestroyedBlocks = gameInstance.GetDestroyedBlocks()
		state.Round = gameInstance.GetTurnCounter()/2 + 1
		b.respond(seat, "GameStarted", state)
		b.notifyOthers(gameInstance, playerID, game.PlayerConnectionMessage{Name: info.name, Connected: true})
		return true
	}
	return false
}

// resumeTurnFor tells a rejoined player whose turn it is, without resetting anyone's AP.
func (b *BattleServer) resumeTurnFor(gameInstance *game.GameInstance, user *UserConnection) {
	current := gameInstance.GetCurrentPlayer()
	b.respondWithMessage(user, game.NextPlayerMessage{
		CurrentPlayer: current,
		YourTurn:      current == user.id,
		Resume:        true,
	})
}
