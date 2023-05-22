package demo

import (
	"bufio"
	"fmt"
	"os"
	"sync"
	"time"
)

type Client struct {
	*Endpoint
}

func NewClient() *Client {
	c := &Client{}
	c.Endpoint = newEndpoint(5, c.handlePkg)
	return c
}

func (c *Client) handlePkg(pkg Package) {
	fmt.Println("Received Message:", string(pkg.Body))
}

func (c *Client) Run() {
	wg := sync.WaitGroup{}
	wg.Add(2)

	var innerErr error
	go func() {
		defer wg.Done()
		innerErr = c.Connect(GatewayClientAddr)
	}()

	go func() {
		defer wg.Done()
		time.Sleep(time.Millisecond * 300)
		fmt.Println("输入1:xxx 发送普通消息. 输入2:xxx 发送广播消息.")
		for {
			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n')
			innerErr = c.Send(PackPackage(ClientToServer, []byte(input)))
			if innerErr != nil {
				return
			}
		}
	}()

	wg.Wait()

	if innerErr != nil {
		panic(innerErr)
	}
}
