package game

import "encoding/json"

type Message interface {
	MessageType() string
}

type BufferedMessage struct {
	MessageType   []byte
	MessageAsJSON []byte
}

type MessageBuffer struct {
	buffer  map[uint64][]*BufferedMessage
	userIDs []uint64
	send    func(userID uint64, messageType, response []byte)
}

func NewMessageBuffer(userIDs []uint64, writer func(userID uint64, messageType, response []byte)) *MessageBuffer {
	return &MessageBuffer{make(map[uint64][]*BufferedMessage), userIDs, writer}
}
func (mb *MessageBuffer) AddMessageFor(userID uint64, response Message) {
	messageAsJSON, _ := json.Marshal(response)
	messageType := []byte(response.MessageType())
	if _, ok := mb.buffer[userID]; !ok {
		mb.buffer[userID] = make([]*BufferedMessage, 0)
	}
	mb.buffer[userID] = append(mb.buffer[userID], &BufferedMessage{messageType, messageAsJSON})
}

func (mb *MessageBuffer) AddMessageForAll(response Message) {
	messageAsJSON, _ := json.Marshal(response)
	messageType := []byte(response.MessageType())
	for _, userID := range mb.userIDs {
		if _, ok := mb.buffer[userID]; !ok {
			mb.buffer[userID] = make([]*BufferedMessage, 0)
		}
		mb.buffer[userID] = append(mb.buffer[userID], &BufferedMessage{messageType, messageAsJSON})
	}
}

func (mb *MessageBuffer) AddMessageForAllExcept(userID uint64, response Message) {
	messageAsJSON, _ := json.Marshal(response)
	messageType := []byte(response.MessageType())
	for i := uint64(0); i < uint64(len(mb.userIDs)); i++ {
		if i != userID {
			if _, ok := mb.buffer[userID]; !ok {
				mb.buffer[userID] = make([]*BufferedMessage, 0)
			}
			mb.buffer[userID] = append(mb.buffer[userID], &BufferedMessage{messageType, messageAsJSON})
		}
	}
}

func (mb *MessageBuffer) SendAll() {
	for userID, messages := range mb.buffer {
		for _, message := range messages {
			mb.send(userID, message.MessageType, message.MessageAsJSON)
		}
	}
	mb.buffer = make(map[uint64][]*BufferedMessage)
}

func (mb *MessageBuffer) UserIDs() []uint64 {
	return mb.userIDs
}
