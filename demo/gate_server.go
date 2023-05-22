package demo

import (
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type Gateway struct {
	sessionIdSn int64 // sessionID自增序列
	listenerWG  sync.WaitGroup
	clients     *ConcurrentMap[int64, Session]
	serverList  *GameServerGroup
}

func NewGateway() *Gateway {
	return &Gateway{
		sessionIdSn: 0,
		listenerWG:  sync.WaitGroup{},
		clients:     &ConcurrentMap[int64, Session]{},
		serverList:  NewGameServerGroup(),
	}
}

func (g *Gateway) Start(clientAddr, serverAddr string) error {
	fmt.Println("Starting Gateway...")

	err := g.serve(clientAddr, g.handleClientConn)
	if err != nil {
		return err
	}
	err = g.serve(serverAddr, g.handleGameServerConn)
	if err != nil {
		return err
	}

	g.startCleanupTicker()

	fmt.Println("Started Gateway.", "clientAddr:", clientAddr, "serverAddr:", serverAddr)

	g.listenerWG.Wait()
	return nil
}

func (g *Gateway) serve(addr string, handleConn func(conn net.Conn)) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	g.listenerWG.Add(1)
	go func() {
		defer func() {
			_ = listener.Close()
			g.listenerWG.Done()
		}()
		for {
			conn, err := listener.Accept()
			if err != nil {
				fmt.Println("Error accepting", err.Error())
				return
			}
			go handleConn(conn)
		}
	}()

	return nil
}

func (g *Gateway) handleClientConn(conn net.Conn) {
	clientSession := NewClientSession(g.nextSessionId(), conn)
	g.clients.Store(clientSession.ID(), clientSession)
	defer g.closeClientSession(clientSession)

	connectedSendErr := clientSession.SendSync(PackPackage(
		ServerToClient,
		EncodeSimpleMsg(SimpleMsg{
			S2CMessage,
			"连接网关成功. sessionID:" + strconv.Itoa(int(clientSession.ID())),
		}),
	))
	if connectedSendErr != nil {
		Stderr("连接网关成功后，发送数据给客户端错误: " + connectedSendErr.Error())
	}

	connReader := NewConnReader(conn)
	for {
		pkg, connReadErr := connReader.Read()
		if connReadErr != nil {
			Stderr("客户端连接读取消息错误: " + connReadErr.Error())
			return
		}
		clientSession.UpdateAliveTime()

		var clientSendErr, serverSendErr error
		switch pkg.Type {
		case Heartbeat:
			fmt.Println("Received Client Heartbeat. SessionID:" + strconv.Itoa(int(clientSession.ID())))
			clientSendErr = clientSession.Send(PackHeartbeatPackage())
		case ClientToServer:
			serverSession := g.serverList.RandomOne()
			if serverSession == nil {
				clientSendErr = clientSession.Send(PackPackage(
					ServerToClient,
					EncodeSimpleMsg(SimpleMsg{S2CMessage, "游戏服务暂不可用"}),
				))
			} else {
				PackageAttachSessionId(&pkg, clientSession.ID())
				serverSendErr = serverSession.Send(pkg.All)
			}
		}
		if clientSendErr != nil {
			Stderr("发送数据给客户端错误: " + clientSendErr.Error())
			return
		}
		if serverSendErr != nil {
			Stderr("发送数据给游戏服错误: " + serverSendErr.Error())
		}
	}
}

func (g *Gateway) handleGameServerConn(conn net.Conn) {
	serverSession := NewGameServerSession(g.nextSessionId(), conn)
	g.serverList.Add(serverSession)
	defer g.closeServerSession(serverSession)

	fmt.Println("Game Server Connected SessionID:", serverSession.ID())

	connReader := NewConnReader(conn)
	for {
		pkg, connReadErr := connReader.Read()
		if connReadErr != nil {
			Stderr("游戏服连接读取消息错误: " + connReadErr.Error())
			return
		}
		serverSession.UpdateAliveTime()

		switch pkg.Type {
		case Heartbeat:
			fmt.Println("Received Server Heartbeat. SessionID:" + strconv.Itoa(int(serverSession.ID())))
			serverSendErr := serverSession.Send(PackHeartbeatPackage())
			if serverSendErr != nil {
				Stderr("发送数据给游戏服错误: " + serverSendErr.Error())
				return
			}
		case ServerToClient:
			clientSessionId := PackagePickSessionId(&pkg)
			clientSession, ok := g.clients.Load(clientSessionId)
			if ok {
				clientSendErr := clientSession.Send(pkg.All)
				if clientSendErr != nil {
					Stderr("发送数据给客户端错误: " + clientSendErr.Error())
					g.closeClientSession(clientSession)
				}
			} else {
				Stderr("发送数据给客户端，找不到sessionID: " + strconv.Itoa(int(clientSessionId)))
			}
		case ServerBroadcast:
			g.clients.Range(func(_ int64, clientSession Session) bool {
				clientSendErr := clientSession.Send(pkg.All)
				if clientSendErr != nil {
					Stderr("发送数据给客户端错误: " + clientSendErr.Error())
					g.closeClientSession(clientSession)
				}
				return true
			})
		}
	}
}

func (g *Gateway) nextSessionId() int64 {
	return atomic.AddInt64(&g.sessionIdSn, 1)
}

func (g *Gateway) closeClientSession(session Session) {
	g.clients.Delete(session.ID())
	session.Close()
}

func (g *Gateway) closeServerSession(session Session) {
	g.serverList.Remove(session.ID())
	session.Close()
}

func (g *Gateway) startCleanupTicker() {
	go func() {
		for t := range time.Tick(ConnectionTimeoutSec * time.Second) {
			unixNow := t.Unix()
			g.clients.Range(func(_ int64, s Session) bool {
				if unixNow-s.AliveTime() > ConnectionTimeoutSec {
					fmt.Println("客户端连接过期. sessionID:", s.ID())
					g.closeClientSession(s)
				}
				return true
			})
			for _, s := range g.serverList.ExpiredTimeoutSession(unixNow) {
				fmt.Println("游戏服连接过期. sessionID:", s.ID())
				g.closeServerSession(s)
			}
		}
	}()
}

type GameServerGroup struct {
	serverList []Session
	rwLock     sync.RWMutex
}

func NewGameServerGroup() *GameServerGroup {
	return &GameServerGroup{
		serverList: make([]Session, 0),
		rwLock:     sync.RWMutex{},
	}
}

func (g *GameServerGroup) RandomOne() Session {
	g.rwLock.RLock()
	defer g.rwLock.RUnlock()
	if len(g.serverList) == 0 {
		return nil
	}
	return g.serverList[rand.Intn(len(g.serverList))]
}

func (g *GameServerGroup) Add(s Session) {
	g.rwLock.Lock()
	defer g.rwLock.Unlock()
	g.serverList = append(g.serverList, s)
}

func (g *GameServerGroup) Remove(id int64) {
	g.rwLock.Lock()
	defer g.rwLock.Unlock()
	var idx = -1
	for i, v := range g.serverList {
		if v.ID() == id {
			idx = i
			break
		}
	}
	if idx > -1 {
		g.serverList = append(g.serverList[:idx], g.serverList[idx+1:]...)
	}
}

func (g *GameServerGroup) ExpiredTimeoutSession(unixNow int64) []Session {
	var timeout []Session
	g.rwLock.RLock()
	defer g.rwLock.RUnlock()
	for _, s := range g.serverList {
		if unixNow-s.AliveTime() > ConnectionTimeoutSec {
			timeout = append(timeout, s)
		}
	}
	return timeout
}
