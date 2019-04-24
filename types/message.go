package types

import (
	"bytes"
	"encoding/binary"

	uuid "github.com/satori/go.uuid"
)

type from uint8

const (
	// Thrower 来自丢瓶方
	Thrower from = 0
	// Replier 来自回应方
	Replier from = 1
)

// Message 消息，每个漂流瓶里有至少一条消息
type Message struct {
	ID       uint16
	BottleID uuid.UUID
	Data     []byte
	From     from
}

// GetKey 获取消息键
func (msg *Message) GetKey() []byte {
	return GetMessageKey(msg.BottleID, msg.ID)
}

// GetMessageKey 获取消息建
func GetMessageKey(bottleID uuid.UUID, msgID uint16) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, msgID)
	key := append(bottleID.Bytes(), buf.Bytes()...)
	return key
}

func (msg *Message) getSignBytes() []byte {
	return msg.GetKey()
}
