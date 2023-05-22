package main

import (
	"game-gate-demo/demo"
	"math/rand"
	"os"
	"strconv"
	"time"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	args := os.Args
	if len(args) <= 1 {
		panic("请输入启动参数")
	}
	arg, err := strconv.Atoi(args[1])
	if err != nil {
		panic("启动参数请输入0/1/2")
	}
	switch arg {
	case 0:
		startGateway()
	case 1:
		startGameServer()
	case 2:
		startClient()
	default:
		panic("启动参数请输入0/1/2")
	}

}

func startGateway() {
	gw := demo.NewGateway()
	err := gw.Start(demo.GatewayClientAddr, demo.GatewayServerAddr)
	if err != nil {
		panic(err)
	}
}

func startGameServer() {
	server := demo.NewGameServer()
	server.Run()
}

func startClient() {
	client := demo.NewClient()
	client.Run()
}
