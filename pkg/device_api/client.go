package device_api

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/robodone/robosla-common/pkg/pubsub"
)

type Client struct {
	conn      Conn
	nd        *pubsub.Node
	mu        sync.Mutex
	isStopped bool
	stopped   chan bool
}

// This channel will be closed, when the client is stopped.
// So, it's useful to wait like:
// <- c.Stopped()
func (c *Client) Stopped() <-chan bool {
	return c.stopped
}

func NewClient(conn Conn, nd *pubsub.Node) *Client {
	c := &Client{
		conn:    conn,
		nd:      nd,
		stopped: make(chan bool),
	}
	go c.run()
	return c
}

func (c *Client) run() {
	for {
		select {
		case <-c.stopped:
			return
		case msg, ok := <-c.conn.In():
			if !ok {
				c.Stop()
				return
			}
			if msg.Type != websocket.TextMessage {
				log.Printf("Unexpected message type %d from server. Skipping the message", msg.Type)
				continue
			}
			log.Printf("Server reply received: %s", string(msg.Data))
			err := c.nd.Pub(string(msg.Data))
			if err != nil {
				log.Printf("Error: failed to publish server updates: %v", err)
			}
		}
	}
}

func (c *Client) Stop() error {
	// Client does not own the connection.
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.isStopped {
		// It's already stopped, but it's OK.
		return nil
	}
	c.isStopped = true
	close(c.stopped)
	return nil
}

func (c *Client) sendRegisterDevice(cookie string) error {
	return send(c.conn, &Request{
		Cmd:    "register-device",
		Cookie: cookie,
	})
}

func (c *Client) RegisterDevice(userCookie string) (deviceCookie string, err error) {
	sub, err := c.SubString("login.cookie")
	if err != nil {
		return "", fmt.Errorf("failed to subscribe for login/cookie: %v", err)
	}
	defer sub.Unsub()
	if err := c.sendRegisterDevice(userCookie); err != nil {
		return "", fmt.Errorf("failed to send registerDevice: %v", err)
	}
	select {
	case deviceCookie = <-sub.C():
	case <-time.Tick(60 * time.Second):
		// TODO(krasin): use contexts. Or use grpc and delete this code.
		return "", errors.New("RegisterDevice: timed out")
	}
	return deviceCookie, nil
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
	defer sub.Unsub()
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
	return deviceName, nil
}

func (c *Client) SendTerminalOutput(out string) error {
	return send(c.conn, &Request{
		Cmd:            "send-terminal-output",
		TerminalOutput: out,
	})

}

func (c *Client) SubString(path string) (*pubsub.StringSub, error) {
	return c.nd.SubString(path)
}

func (c *Client) Sub(path string) (*pubsub.Sub, error) {
	return c.nd.Sub(path)
}
