package device_api

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/robodone/robosla-common/pkg/pubsub"
)

type Client struct {
	conn    Conn
	nd      *pubsub.Node
	stopped chan bool
}

func NewClient(conn Conn) *Client {
	c := &Client{
		conn:    conn,
		nd:      pubsub.NewNode(),
		stopped: make(chan bool, 1),
	}
	go c.run()
	return c
}

func (c *Client) run() {
	for {
		select {
		case <-c.stopped:
			return
		case msg := <-c.conn.In():
			log.Printf("Server reply received: %s", msg)
			err := c.nd.Pub(msg)
			if err != nil {
				log.Printf("Error: failed to publish server updates: %v", err)
			}
		}
	}
}

func (c *Client) Stop() error {
	// Client does not own the connection.
	close(c.stopped)
	c.nd.Stop()
	return nil
}

func (c *Client) sendHello(cookie string) error {
	return send(c.conn, &Request{
		Cmd:    "hello",
		Cookie: cookie,
	})
}

func (c *Client) Hello(cookie string) (machineName string, err error) {
	sub, err := c.SubString("login.deviceName")
	if err != nil {
		return "", fmt.Errorf("failed to subscribe for login/deviceName: %v", err)
	}
	if err := c.sendHello(cookie); err != nil {
		return "", fmt.Errorf("failed to send hello: %v", err)
	}
	var deviceName string
	select {
	case deviceName = <-sub.C():
	case <-time.Tick(60 * time.Second):
		// TODO(krasin): use contexts. Or use grpc and delete this code.
		return "", errors.New("Hello: timed out")
	}
	sub.Unsub()
	return deviceName, nil
}

func (c *Client) SubString(path string) (*pubsub.StringSub, error) {
	return c.nd.SubString(path)
}
