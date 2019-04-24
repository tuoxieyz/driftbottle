package main

import (
	"fmt"

	"github.com/tendermint/tendermint/abci/server"
)

func main() {
	//var app tmtypes.Application
	app := newDriftBottleApplication()

	svr := server.NewSocketServer(":26658", app)
	svr.Start()
	defer svr.Stop()
	fmt.Println("abci server started.")
	select {} //阻塞
}
