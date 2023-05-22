package demo

import (
	"fmt"
	"io"
	"net"
	"sync/atomic"
	"time"
)

type Endpoint struct {
	conn           net.Conn
	sendChannel    chan []byte
	closeChannel   chan struct{}
	closeFlag      int32
	packageHandler func(pkg Package)
}

func newEndpoint(sendChannelSize int, packageHandler func(pkg Package)) *Endpoint {
	return &Endpoint{
		sendChannel:    make(chan []byte, sendChannelSize),
		closeChannel:   make(chan struct{}),
		closeFlag:      0,
		packageHandler: packageHandler,
	}
}

func (e *Endpoint) Connect(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	e.conn = conn

	go e.pollSendChannel()
	go e.receive()
	go e.keepalive()

	<-e.closeChannel
	return nil
}

func (e *Endpoint) pollSendChannel() {
	defer e.Close()
	for {
		select {
		case bytes := <-e.sendChannel:
			_, err := e.conn.Write(bytes)
			if err != nil {
				Stderr("连接写数据错误:" + err.Error())
				return
			}
		case <-e.closeChannel:
			return
		}
	}
}

func (e *Endpoint) receive() {
	defer e.Close()

	connReader := NewConnReader(e.conn)
	for {
		pkg, connReadErr := connReader.Read()
		if connReadErr != nil {
			if connReadErr != io.EOF {
				Stderr("读取消息错误: " + connReadErr.Error())
			}
			return
		}
		switch pkg.Type {
		case Heartbeat:
			fmt.Println("Received Heartbeat Echo")
		default:
			e.packageHandler(pkg)
		}
	}
}

func (e *Endpoint) keepalive() {
	var err error
	defer func() {
		if err != nil {
			Stderr("Endpoint keepalive err:" + err.Error())
		}
	}()
	tick := time.Tick(HeartbeatIntervalSec * time.Second)

	for {
		select {
		case <-tick:
			err = e.Send(PackHeartbeatPackage())
			if err != nil {
				return
			}
		case <-e.closeChannel:
			return
		}
	}
}

func (e *Endpoint) Send(bs []byte) error {
	if e.closed() {
		return fmt.Errorf("连接已关闭")
	}
	e.sendChannel <- bs
	return nil
}

func (e *Endpoint) Close() {
	if atomic.CompareAndSwapInt32(&e.closeFlag, 0, 1) {
		close(e.closeChannel)
		_ = e.conn.Close()
		fmt.Println("Endpoint Conn Closed")
	}
}

func (e *Endpoint) closed() bool {
	return atomic.LoadInt32(&e.closeFlag) == 1
}
