package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"time"
	"tuoxie/driftbottle"
	"tuoxie/driftbottle/types"

	uuid "github.com/satori/go.uuid"

	tmtypes "github.com/tendermint/tendermint/abci/types"
	dbm "github.com/tendermint/tendermint/libs/db"
)

const (
	// 匹配如下格式
	// /salvage
	// /userpubkey/bottles
	// /userpubkey/bottles/bottleid
	// /userpubkey/bottles/bottleid/messages/
	// /userpubkey/bottles/bottleid/messages/msgstartidx //这里的userpubkey并不要求与后面的bottle和message有关系
	// /userpubkey/bottles/bottleid/messages/msgstartidx-msgendidx //业务暂未实现
	// go不支持后向引用（分组可用于表达式其它地方），:(
	// 如`^/bottles/(?P<bid>\w+)(?P<msgs>/messages/)?(?(msgs)(\d+(-\d+)?))?$`会引发运行时错误
	queryPathPattern string = `^/(salvage|(?P<uk>\S+)/bottles(/(?P<bid>[\w\-]+)((?P<msgs>/messages/)((?P<msid>\d+)(-(?P<meid>\d+))?)?)?)?)$`
	appKEY           string = "driftbottleappkey"
)

var (
	cdc           = driftbottle.AminoCdc
	dbDir         = "data"
	throwerPrefix = []byte("thrower:")
	replerPrefix  = []byte("repler:")
	bottlePoolKey = []byte("bottlepool")
	//bottleMsgsPrefix = []byte("bottle:")

	salvagedBottles []uuid.UUID // 本轮被打捞的瓶子
)

type driftBottleApplication struct {
	tmtypes.BaseApplication
	db          dbm.DB
	BlockHeight int64  `json:"height"`
	AppHash     []byte `json:"app_hash"`
}

func newDriftBottleApplication() *driftBottleApplication {
	name := "driftbottleapp"
	db, err := dbm.NewGoLevelDB(name, dbDir)
	if err != nil {
		panic(err)
	}

	appBytes := db.Get([]byte(appKEY))
	var app driftBottleApplication
	if len(appBytes) != 0 {
		err := json.Unmarshal(appBytes, &app)
		if err != nil {
			panic(err)
		}
	}
	app.db = db
	return &app
}

func (app *driftBottleApplication) Info(req tmtypes.RequestInfo) tmtypes.ResponseInfo {
	res := tmtypes.ResponseInfo{LastBlockHeight: app.BlockHeight}
	return res
}

func (app *driftBottleApplication) CheckTx(raw []byte) (rsp tmtypes.ResponseCheckTx) {
	//fmt.Println("begin checktx")
	var tx types.Transx
	err := cdc.UnmarshalJSON(raw, &tx)
	// value, ok := tx.(*types.Bottle) // true
	// fmt.Printf("%v,%t", value, ok)
	if err != nil {
		rsp.Code = 1
		rsp.Log = "error occured in decoding when CheckTx"
		return
	}
	if !tx.Verify() {
		rsp.Code = 2
		rsp.Log = "CheckTx failed"
		return
	}

	// 业务校验
	bottle, ok := tx.Payload.(*types.Bottle)
	if ok {
		if bottle.Replier != [32]byte{} { //若不是默认值则表明这是salvage操作，否则是throw操作
			for _, v := range salvagedBottles {
				if uuid.Equal(v, bottle.ID) {
					rsp.Code = 3
					rsp.Log = "这个瓶子被别人捞走了，去打捞下一个吧~"
					return
				}
			}
			salvagedBottles = append(salvagedBottles, bottle.ID)
		}
	} else {
		msg, ok := tx.Payload.(*types.Message)
		if ok {
			btBytes := app.db.Get(msg.BottleID.Bytes())
			if btBytes == nil {
				rsp.Code = 3
				rsp.Log = "there is no bottle associated with the msg's bottleID"
				return
			}
		}
	}

	fmt.Println("CheckTx successed")
	return
}

func (app *driftBottleApplication) DeliverTx(raw []byte) (rsp tmtypes.ResponseDeliverTx) {
	var tx types.Transx
	cdc.UnmarshalJSON(raw, &tx) //由于之前CheckTx中转换过，所以这里讲道理不会有error

	bottle, ok := tx.Payload.(*types.Bottle)
	if ok {
		if bottle.Replier == [32]byte{} {
			// 更新投放者的瓶子集合
			key := append(throwerPrefix, bottle.Thrower[:]...)
			bids := app.db.Get(key)
			if bids == nil {
				bids = bottle.ID[:]
			} else {
				bids = append(bottle.ID[:], bids...)
			}
			app.db.Set(key, bids)

			//将新瓶子放到池子里，等待后续打捞
			bottlePool := app.db.Get(bottlePoolKey)
			if bottlePool == nil {
				bottlePool = bottle.ID[:]
			} else {
				bottlePool = append(bottlePool, bottle.ID[:]...)
			}
			app.db.Set(bottlePoolKey, bottlePool)
		} else {
			bottlePool := app.db.Get(bottlePoolKey)
			removePartofBytes(bottlePool, bottle.ID.Bytes()) //从池中移除该漂流瓶
			app.db.Set(bottlePoolKey, bottlePool)
		}
	} else {
		msg, ok := tx.Payload.(*types.Message)
		if ok {
			btBytes := app.db.Get(msg.BottleID.Bytes())
			var txtemp types.Transx
			cdc.UnmarshalJSON(btBytes, &txtemp)
			bottle, _ = txtemp.Payload.(*types.Bottle)
			
			bottle.MessagesNum++
			msg.ID = bottle.MessagesNum // 消息ID就是bottle中的当前消息总数
			btBytes, _ = cdc.MarshalJSON(txtemp)
			app.db.Set(msg.BottleID.Bytes(), btBytes)

			raw, _ = cdc.MarshalJSON(tx)
		}
	}

	app.db.Set(tx.Payload.GetKey(), raw)
	
	fmt.Println("delivertx successed")
	return
}

func existsInBytes(list, part []byte) bool {
	size := len(part)
	for i := 0; i < len(list); i += size {
		temp := list[i : i+size]
		if bytes.Equal(temp, part) {
			return true
		}
	}
	return false
}

func removePartofBytes(origin, diss []byte) {
	size := len(diss)
	for i := 0; i < len(origin); i += size {
		part := origin[i : i+size]
		if bytes.Equal(part, diss) {
			origin = append(origin[:i], origin[i+size:]...)
			return
		}
	}
}

func (app *driftBottleApplication) Commit() tmtypes.ResponseCommit {
	salvagedBottles = salvagedBottles[:0] //清空salvagedBottles

	app.BlockHeight++
	appBytes, err := json.Marshal(app)
	if err != nil {
		panic(err)
	}
	app.db.Set([]byte(appKEY), appBytes)
	return tmtypes.ResponseCommit{Data: app.AppHash} //{Data: appBytes} 将height放入apphash将引起无限生成块，因为每次commit的apphash都与之前一个不一样
}

func getMatchMap(submatches []string, groupNames []string) map[string]string {
	result := make(map[string]string)
	for i, name := range groupNames {
		if i != 0 && name != "" {
			result[name] = submatches[i]
		}
	}
	return result
}

func (app *driftBottleApplication) Query(req tmtypes.RequestQuery) (rsp tmtypes.ResponseQuery) {
	//match, _ := regexp.MatchString(queryPathPattern, req.Path)
	reg := regexp.MustCompile(queryPathPattern)
	submatches := reg.FindStringSubmatch(req.Path)

	if submatches == nil {
		rsp.Code = 1
		rsp.Info = "Invalid argument - path"
		println("Invalid argument - path")
	} else if req.Path == "/salvage" {
		bottlePool := app.db.Get(bottlePoolKey)
		if bottlePool == nil || len(bottlePool) == 0 {
			rsp.Info = "漂流瓶被捞完啦!"
			return
		}
		idlen := uuid.Size
		psize := len(bottlePool) / idlen
		rand.Seed(time.Now().UnixNano())
		idx := rand.Intn(psize)
		btBytes := app.db.Get(bottlePool[idx*idlen : idx*idlen+idlen])
		rsp.Value = btBytes
		return
	} else {
		groupNames := reg.SubexpNames()
		matchmap := getMatchMap(submatches, groupNames)

		var uk []byte
		err := cdc.UnmarshalJSON([]byte(matchmap["uk"]), &uk)
		if err != nil {
			rsp.Code = 1
			rsp.Log = err.Error()
			return
		}

		if matchmap["bid"] != "" {
			btKey := uuid.FromStringOrNil(matchmap["bid"]) //[]byte(matchmap["bid"])
			if matchmap["msgs"] != "" {
				msid, _ := strconv.Atoi(matchmap["msid"])
				// meid := bottle.MessagesNum
				// if smeid := matchmap["meid"]; smeid != "" {
				// 	meid, _ := strconv.Atoi(smeid)
				// }
				msgKey := types.GetMessageKey(btKey, uint16(msid))
				rsp.Value = app.db.Get(msgKey)
				return
			}

			btBytes := app.db.Get(btKey.Bytes())
			rsp.Value = btBytes
			return
		}
		key := append(throwerPrefix, uk...)
		bids := app.db.Get(key)
		rsp.Value = bids
		//fmt.Printf("%v,%v", submatches, groupNames)
	}
	return
}
