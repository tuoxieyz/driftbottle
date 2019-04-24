package types

import (
	"time"

	"github.com/tendermint/tendermint/crypto"
)

// IPayload 接口
type IPayload interface {
	getSignBytes() []byte
	GetKey() []byte
}

// Transx 事务基类
type Transx struct {
	Signature  []byte //发送方对这个消息的私钥签名
	SendTime   *time.Time
	SignPubKey crypto.PubKey
	Payload    IPayload
}

/*Sign 给消息签名
privKey:发送方私钥
*/
func (cmu *Transx) Sign(privKey crypto.PrivKey) error {
	bz := cmu.Payload.getSignBytes()
	sig, err := privKey.Sign(bz)
	cmu.Signature = sig
	return err
}

// Verify 校验消息是否未被篡改
func (cmu *Transx) Verify() bool {
	data := cmu.Payload.getSignBytes()
	sig := cmu.Signature
	rslt := cmu.SignPubKey.VerifyBytes(data, sig)
	return rslt
}
