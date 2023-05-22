package demo

import (
	"fmt"
	"net"
	"sync/atomic"
	"time"
)

type Session interface {
	ID() int64
	UpdateAliveTime()
	AliveTime() int64
	Close()
	Send(bs []byte) error
	SendSync(bs []byte) error
}

type BaseSession struct {
	id           int64
	conn         net.Conn
	aliveTime    int64
	sendChannel  chan []byte
	closeChannel chan struct{}
	closeFlag    int32
}

func newBaseSession(id int64, conn net.Conn, sendChannelSize int) *BaseSession {
	s := &BaseSession{
		id:           id,
		conn:         conn,
		aliveTime:    time.Now().Unix(),
		sendChannel:  make(chan []byte, sendChannelSize),
		closeChannel: make(chan struct{}),
		closeFlag:    0,
	}
	go s.pollSendChannel()
	return s
}

func (b *BaseSession) ID() int64 {
	return b.id
}

func (b *BaseSession) UpdateAliveTime() {
	b.aliveTime = time.Now().Unix()
}

func (b *BaseSession) AliveTime() int64 {
	return b.aliveTime
}

func (b *BaseSession) Close() {
	if atomic.CompareAndSwapInt32(&b.closeFlag, 0, 1) {
		close(b.closeChannel)
		_ = b.conn.Close()
	}
}

func (b *BaseSession) closed() bool {
	return atomic.LoadInt32(&b.closeFlag) == 1
}

func (b *BaseSession) Send(bs []byte) error {
	if b.closed() {
		return fmt.Errorf("session已关闭")
	}
	select {
	case b.sendChannel <- bs:
	case <-time.After(time.Second):
		return fmt.Errorf("session发送队列阻塞超时")
	}
	return nil
}

func (b *BaseSession) SendSync(bs []byte) error {
	if b.closed() {
		return fmt.Errorf("session已关闭")
	}
	_, err := b.conn.Write(bs)
	return err
}

func (b *BaseSession) pollSendChannel() {
	defer b.Close()
	for {
		select {
		case bytes := <-b.sendChannel:
			_, err := b.conn.Write(bytes)
			if err != nil {
				Stderr("连接写数据错误:" + err.Error())
				return
			}
		case <-b.closeChannel:
			return
		}
	}
}

type ClientSession struct {
	*BaseSession
}

type GameServerSession struct {
	*BaseSession
}

func NewClientSession(id int64, conn net.Conn) Session {
	return &ClientSession{
		BaseSession: newBaseSession(id, conn, 16),
	}
}

func NewGameServerSession(id int64, conn net.Conn) Session {
	return &GameServerSession{
		BaseSession: newBaseSession(id, conn, 256),
	}
}
