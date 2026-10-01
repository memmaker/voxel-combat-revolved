package game

import (
	"bufio"
	"encoding/json"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/voxel"
	"log"
	"net"
	"strings"
)

type ServerConnection struct {
	connection        net.Conn
	eventHandler      func(msg StringMessage)
	mainthreadChannel chan StringMessage
}

type StringMessage struct {
	Message     string
	MessageType string
}

func NewTCPConnection(endpoint string) *ServerConnection {
	con, err := Dial(endpoint)
	if err != nil {
		log.Fatalln(err)
	}
	println("Connected to server")
	s := &ServerConnection{connection: con}
	go s.readLoop(bufio.NewReader(con))
	return s
}

func NewTCPConnectionWithHandler(endpoint string, handler func(msg StringMessage)) *ServerConnection {
	con, err := Dial(endpoint)
	if err != nil {
		log.Fatalln(err)
	}
	println("Connected to server")
	s := &ServerConnection{connection: con}
	s.SetEventHandler(handler)
	go s.readLoop(bufio.NewReader(con))
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

func (c *ServerConnection) CreateGame(mapName string, gameID string, details *MissionDetails, isPublic bool) error {
	message := CreateGameMessage{Map: mapName, GameIdentifier: gameID, IsPublic: isPublic, MissionDetails: details}
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
	//println(fmt.Sprintf("[ServerConnection] Sending message: %s", messageType))
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
		//println(fmt.Sprintf("[ServerConnection] Received message: %s", messageType))
		msg := StringMessage{MessageType: messageType, Message: message}
		if c.eventHandler != nil {
			c.eventHandler(msg)
		} else if c.mainthreadChannel != nil {
			c.mainthreadChannel <- msg
		} else {
			log.Println("No event handler or mainthread channel set")
		}
	}
}

func (c *ServerConnection) SetEventHandler(handler func(msg StringMessage)) {
	c.eventHandler = handler
	c.mainthreadChannel = nil
}

func (c *ServerConnection) SetMainthreadChannel(channel chan StringMessage) {
	c.mainthreadChannel = channel
	c.eventHandler = nil
}

func (c *ServerConnection) SelectUnits(choices []UnitChoice) error {
	message := SelectUnitsMessage{Units: choices}
	return c.send("SelectUnits", message)
}

func (c *ServerConnection) TargetedUnitAction(gameUnitID uint64, action string, target []voxel.Int3) error {
	message := TargetedUnitActionMessage{UnitMessage: UnitMessage{GameUnitID: gameUnitID}, Action: action, Targets: target}
	return c.send("UnitAction", message)
}

func (c *ServerConnection) ThrownUnitAction(gameUnitID uint64, action string, itemName string, target []mgl32.Vec3) error {
	message := ThrownUnitActionMessage{UnitMessage: UnitMessage{GameUnitID: gameUnitID}, Action: action, Targets: target, ItemName: itemName}
	return c.send("ThrownUnitAction", message)
}

func (c *ServerConnection) FreeAimAction(gameUnitID uint64, action string, camPos mgl32.Vec3, targetAngles [][2]float32) error {
	message := FreeAimActionMessage{UnitMessage: UnitMessage{GameUnitID: gameUnitID}, Action: action, CamPos: camPos, TargetAngles: targetAngles}
	return c.send("FreeAimAction", message)
}

type NoData struct {
}

func (c *ServerConnection) EndTurn() error {
	return c.send("EndTurn", NoData{})
}

func (c *ServerConnection) MapLoaded() error {
	return c.send("MapLoaded", MapLoadedMessage{})
}

func (c *ServerConnection) ReloadAction(unitID uint64) error {
	return c.send("Reload", UnitMessage{GameUnitID: unitID})
}

func (c *ServerConnection) SelectDeployment(deployment map[uint64]voxel.Int3) error {
	return c.send("SelectDeployment", DeploymentMessage{
		Deployment: deployment,
	})
}
func (c *ServerConnection) DebugRequest(command string) error {
	message := DebugRequest{Command: command}
	return c.send("DebugRequest", message)
}

// Dial opens the client connection; the browser build swaps it for an in-memory pipe to an in-process server.
var Dial = func(endpoint string) (net.Conn, error) { return net.Dial("tcp", endpoint) }

// MemPipe is a buffered in-memory net.Conn pair (net.Pipe is unbuffered and would deadlock request/response handlers).
func MemPipe() (net.Conn, net.Conn) {
	a, b := make(chan []byte, 4096), make(chan []byte, 4096)
	return &memConn{in: a, out: b}, &memConn{in: b, out: a}
}

type memConn struct {
	net.Conn // unused methods panic
	in, out  chan []byte
	buf      []byte
}

func (c *memConn) Write(p []byte) (int, error) {
	c.out <- append([]byte(nil), p...)
	return len(p), nil
}

func (c *memConn) Read(p []byte) (int, error) {
	if len(c.buf) == 0 {
		c.buf = <-c.in
	}
	n := copy(p, c.buf)
	c.buf = c.buf[n:]
	return n, nil
}

func (c *memConn) Close() error { return nil }
