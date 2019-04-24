package types

import (
	uuid "github.com/satori/go.uuid"
)

// Bottle 漂流瓶
type Bottle struct {
	ID      uuid.UUID
	Thrower [32]byte //扔瓶者的加密公钥
	//Messages         []Message // leveldb的k-v模式（顺序读效率不高）不适合relationship
	Title       string   //塞入的第一条消息
	MessagesNum uint16   //瓶中消息数量
	Replier     [32]byte //回应者的加密公钥
}

// GetKey 获取实体键
func (bottle *Bottle) GetKey() []byte {
	return bottle.ID.Bytes()
}

func (bottle *Bottle) getSignBytes() []byte {
	return bottle.ID[:]
}