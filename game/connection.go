package game

import (
	"bufio"
	"encoding/json"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/engine/voxel"
	"log"
	"net"
	"strings"
)

type ServerConnection struct {
	connection   util.VirtualInterface
	eventHandler func(msgType, data string)
}

func NewTCPConnection() *ServerConnection {
	con, err := net.Dial("tcp", "0.0.0.0:9999")
	if err != nil {
		log.Fatalln(err)
	}
	println("Connected to server")
	s := &ServerConnection{connection: con}
	go s.readLoop(bufio.NewReader(con))
	return s
}

func NewChannelConnection(serverChannel *util.ChannelWrapper) *ServerConnection {
	// create a string channel
	s := &ServerConnection{connection: serverChannel}
	go s.readLoop(bufio.NewReader(s.connection))
	return s
}

func (c *ServerConnection) Login(username string) error {
	message := LoginMessage{Username: username}
	return c.send("Login", message)
}

func (c *ServerConnection) SelectFaction(factionName string) error {
	message := SelectFactionMessage{FactionName: factionName}
	return c.send("SelectFaction", message)
}

func (c *ServerConnection) CreateGame(mapName string, gameID string, isPublic bool) error {
	message := CreateGameMessage{Map: mapName, GameIdentifier: gameID, IsPublic: isPublic}
	return c.send("CreateGame", message)
}

func (c *ServerConnection) JoinGame(gameID string) error {
	message := JoinGameMessage{GameID: gameID}
	return c.send("JoinGame", message)
}

func (c *ServerConnection) send(messageType string, message any) error {
	dataAsJson, err := json.Marshal(message)
	if err != nil {
		return err
	}
	_, err = c.connection.Write(append([]byte(messageType), '\n'))
	_, err = c.connection.Write(append(dataAsJson, '\n'))
	return err
}

func (c *ServerConnection) readLoop(serverReader *bufio.Reader) {
	for {
		// our protocol is: messageType (string) + \n + data as json (string) + \n
		messageType, err := serverReader.ReadString('\n')
		if err != nil {
			log.Println(err)
			return
		}
		message, err := serverReader.ReadString('\n')
		if err != nil {
			log.Println(err)
			return
		}
		message = strings.TrimSpace(message)
		messageType = strings.TrimSpace(messageType)
		if c.eventHandler != nil {
			c.eventHandler(messageType, message)
		}
	}
}

func (c *ServerConnection) SetEventHandler(handler func(msgType, data string)) {
	c.eventHandler = handler
}

func (c *ServerConnection) SelectUnits(choices []UnitChoices) error {
	message := SelectUnitsMessage{Units: choices}
	return c.send("SelectUnits", message)
}

func (c *ServerConnection) TargetedUnitAction(gameUnitID uint64, action string, target voxel.Int3) error {
	message := TargetedUnitActionMessage{GameUnitID: gameUnitID, Action: action, Target: target}
	return c.send("TargetedUnitAction", message)
}

type NoData struct {
}

func (c *ServerConnection) EndTurn() error {
	return c.send("EndTurn", NoData{})
}
