package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"time"
	"tuoxie/driftbottle"
	"tuoxie/driftbottle/types"

	cfg "github.com/tendermint/tendermint/config"
	cmn "github.com/tendermint/tendermint/libs/common"
	rpcclient "github.com/tendermint/tendermint/rpc/client"

	crypto_rand "crypto/rand"

	uuid "github.com/satori/go.uuid"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"golang.org/x/crypto/nacl/box"
)

// KEYFILENAME 私钥文件名
const KEYFILENAME string = ".userkey"

var (
	cli *rpcclient.HTTP
	cdc = driftbottle.AminoCdc
)

func init() {
	addr := cfg.DefaultRPCConfig().ListenAddress
	cli = rpcclient.NewHTTP(addr, "/websocket")
}

type cryptoPair struct {
	PrivKey *[32]byte
	PubKey  *[32]byte
}

type user struct {
	SignKey    crypto.PrivKey `json:"sign_key"` // 节点私钥，用户签名
	CryptoPair cryptoPair     // 密钥协商使用
	bottleIDs  []string       // 投放的所有漂流瓶id集合
}

func loadOrGenUserKey() (*user, error) {
	if cmn.FileExists(KEYFILENAME) {
		uk, err := loadUserKey()
		if err != nil {
			return nil, err
		}
		return uk, nil
	}
	//fmt.Println("userkey file not exists")
	uk := new(user)
	uk.SignKey = ed25519.GenPrivKey()
	pubKey, priKey, err := box.GenerateKey(crypto_rand.Reader)
	if err != nil {
		return nil, err
	}
	uk.CryptoPair = cryptoPair{PrivKey: priKey, PubKey: pubKey}
	jsonBytes, err := cdc.MarshalJSON(uk)
	if err != nil {
		return nil, err
	}
	err = ioutil.WriteFile(KEYFILENAME, jsonBytes, 0644)
	if err != nil {
		return nil, err
	}
	return uk, nil
}

func loadUserKey() (*user, error) {
	//copy(privKey[:], bz)
	jsonBytes, err := ioutil.ReadFile(KEYFILENAME)
	if err != nil {
		return nil, err
	}
	uk := new(user)
	err = cdc.UnmarshalJSON(jsonBytes, uk)
	if err != nil {
		return nil, fmt.Errorf("Error reading UserKey from %v: %v", KEYFILENAME, err)
	}
	return uk, nil
}

func (me *user) throwBottle(content string) error {
	now := time.Now()
	tx := new(types.Transx)
	tx.SendTime = &now

	bottle := new(types.Bottle)
	bottle.ID = uuid.NewV4()
	bottle.Thrower = *me.CryptoPair.PubKey
	bottle.MessagesNum = 0
	bottle.Title = content

	tx.Payload = bottle

	tx.Sign(me.SignKey)
	tx.SignPubKey = me.SignKey.PubKey()
	
	bz, err := cdc.MarshalJSON(&tx)
	if err != nil {
		return err
	}

	ret, err := cli.BroadcastTxSync(bz)
	if err != nil {
		return err
	}
	fmt.Printf("throw bottle => %+v\n", ret)
	return nil
}

// getMyBottles 获取由我投放的所有漂流瓶Id
func (me *user) getMyBottles() error {
	addr, _ := cdc.MarshalJSON(*me.CryptoPair.PubKey)

	//addr = addr[1 : len(addr)-1] // 移除两边的引号
	var buf bytes.Buffer
	buf.WriteString("/")
	buf.Write(addr)
	buf.WriteString("/bottles")
	//获得拼接后的字符串
	path := buf.String()
	rsp, _ := cli.ABCIQuery(path, nil)

	me.bottleIDs = me.bottleIDs[0:0] // 清空
	data := rsp.Response.Value
	for i := 0; i < len(data); i += 16 {
		id := uuid.FromBytesOrNil(data[i : i+16])
		fmt.Println(id.String())
		me.bottleIDs = append(me.bottleIDs, id.String())
	}

	return nil
}

func (me *user) getBottle(id string) *types.Bottle {
	addr, _ := cdc.MarshalJSON(*me.CryptoPair.PubKey)
	var buf bytes.Buffer
	buf.WriteString("/")
	buf.Write(addr)
	buf.WriteString("/bottles/")
	buf.WriteString(id)
	//获得拼接后的字符串
	path := buf.String()
	rsp, _ := cli.ABCIQuery(path, nil)

	data := rsp.Response.Value
	var tx types.Transx
	cdc.UnmarshalJSON(data, &tx)

	//fmt.Printf("tx=>%+v\n", tx)
	fmt.Printf("bottle=>%+v", tx.Payload)

	bottle, ok := tx.Payload.(*types.Bottle)
	if ok {
		return bottle
	}
	return nil
}

// 捞瓶子
func (me *user) salvage() {
	path := "/salvage"
	rsp, _ := cli.ABCIQuery(path, nil)
	data := rsp.Response.Value

	var tx types.Transx
	cdc.UnmarshalJSON(data, &tx)
	bottle, _ := tx.Payload.(*types.Bottle)
	bottle.Replier = *me.CryptoPair.PubKey // 标记Replier即表示该bottle被人打捞了

	bz, err := cdc.MarshalJSON(&tx)
	if err != nil {
		fmt.Println(err)
	}

	_, err = cli.BroadcastTxCommit(bz)
	if err != nil {
		fmt.Println(err)
	} else {
		//fmt.Printf("tx=>%+v\n", tx)
		fmt.Printf("bottle=>%+v\n", tx.Payload)
	}
}

func (me *user) getMessageOfBottle(bottleID string, mid uint16) {
	bottle := me.getBottle(bottleID)
	if bottle == nil {
		fmt.Println("wrong bottle ~")
		return
	}

	if bottle.Thrower != *me.CryptoPair.PubKey && bottle.Replier != *me.CryptoPair.PubKey {
		fmt.Println("这个瓶子和你无缘~")
		return
	}

	addr, _ := cdc.MarshalJSON(*me.CryptoPair.PubKey)
	var buf bytes.Buffer
	buf.WriteString("/")
	buf.Write(addr)
	buf.WriteString("/bottles/")
	buf.WriteString(bottleID)
	buf.WriteString("/messages/")
	//mkey := types.GetMessageKey(uuid.FromStringOrNil(bottleID), uint16(mid))
	msid := strconv.Itoa(int(mid))
	buf.WriteString(msid)
	path := buf.String()

	rsp, _ := cli.ABCIQuery(path, nil)

	data := rsp.Response.Value
	var tx types.Transx
	cdc.UnmarshalJSON(data, &tx)
	//fmt.Printf("%v,%s", tx, path)
	msg, ok := tx.Payload.(*types.Message)
	if ok {
		var decryptKey, publicKey [32]byte
		if bottle.Thrower == *me.CryptoPair.PubKey {
			publicKey = bottle.Replier
		} else {
			publicKey = bottle.Thrower
		}

		box.Precompute(&decryptKey, &publicKey, me.CryptoPair.PrivKey)
		var decryptNonce [24]byte
		copy(decryptNonce[:], msg.Data[:24])
		//fmt.Printf("msg.Data=>%v,decryptNonce=>%v,decryptKey=>%v\n", msg.Data[24:], decryptNonce, decryptKey)
		decrypted, ok := box.OpenAfterPrecomputation(nil, msg.Data[24:], &decryptNonce, &decryptKey)
		if !ok {
			panic("decryption error")
		}
		msg.Data = decrypted
		fmt.Printf("message=>%s", string(msg.Data))
	}
}

func (me *user) reply(bottleID, msg string) {
	bottle := me.getBottle(bottleID)
	if bottle == nil {
		fmt.Println("wrong bottle ~")
		return
	}
	if bottle.Replier == [32]byte{} {
		fmt.Println("请先把这个瓶子打捞上来再说~")
		return
	}
	if bottle.Thrower != *me.CryptoPair.PubKey && bottle.Replier != *me.CryptoPair.PubKey {
		fmt.Println("这个瓶子和你无缘~")
		return
	}

	sharedEncryptKey := new([32]byte)
	box.Precompute(sharedEncryptKey, &bottle.Thrower, me.CryptoPair.PrivKey)

	var nonce [24]byte
	if _, err := io.ReadFull(crypto_rand.Reader, nonce[:]); err != nil {
		panic(err)
	}
	//fmt.Printf("msg=>%v,nonce=>%v,sharedEncryptKey=>%v\n", msg, nonce, *sharedEncryptKey)
	encrypted := box.SealAfterPrecomputation(nonce[:], []byte(msg), &nonce, sharedEncryptKey)

	now := time.Now()
	tx := new(types.Transx)
	tx.SendTime = &now

	message := types.Message{
		//ID:       0, //放到server端赋值
		Data:     encrypted,
		From:     types.Replier,
		BottleID: uuid.FromStringOrNil(bottleID),
	}

	tx.Payload = &message

	tx.Sign(me.SignKey)
	tx.SignPubKey = me.SignKey.PubKey()
	// broadcast this tx
	bz, err := cdc.MarshalJSON(&tx)
	if err != nil {
		fmt.Println(err)
	}

	ret, err := cli.BroadcastTxAsync(bz)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("reply bottle => %+v\n", ret)
}
