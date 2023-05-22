package demo

import (
	"fmt"
	"sync"
)

type GameServer struct {
	*Endpoint
}

func NewGameServer() *GameServer {
	c := &GameServer{}
	c.Endpoint = newEndpoint(64, c.handlePkg)
	return c
}

func (c *GameServer) handlePkg(pkg Package) {
	if pkg.Type != ClientToServer {
		return
	}
	clientSessionId := PackagePickSessionId(&pkg)
	fmt.Println("Received Message:", string(pkg.Body), "From ClientSessionID:", clientSessionId)
	msg, err := DecodeSimpleMsg(pkg.Body)
	if err != nil {
		sendPkg := NewPackage(
			ServerToClient,
			EncodeSimpleMsg(SimpleMsg{
				S2CMessage,
				"消息反序列化错误:" + err.Error(),
			}),
		)
		PackageAttachSessionId(&sendPkg, clientSessionId)
		err = c.Send(sendPkg.All)
		return
	}

	switch msg.Cmd {
	case C2SOperate:
		sendPkg := NewPackage(
			ServerToClient,
			EncodeSimpleMsg(SimpleMsg{
				S2CMessage,
				msg.Msg + " -- Server Response",
			}),
		)
		PackageAttachSessionId(&sendPkg, clientSessionId)
		err = c.Send(sendPkg.All)
	case C2SBroadcast:
		err = c.Send(PackPackage(ServerBroadcast,
			EncodeSimpleMsg(SimpleMsg{
				S2CMessage,
				msg.Msg + " -- Server Broadcast",
			})))
	default:
		fmt.Println("接收到未知命令")
	}
}

func (c *GameServer) Run() {
	wg := sync.WaitGroup{}
	wg.Add(1)

	var innerErr error
	go func() {
		defer wg.Done()
		innerErr = c.Connect(GatewayServerAddr)
	}()

	fmt.Println("Game Server Started")
	wg.Wait()

	if innerErr != nil {
		panic(innerErr)
	}
}
