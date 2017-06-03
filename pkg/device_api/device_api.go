package device_api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"

	"github.com/gorilla/websocket"
	"github.com/robodone/robosla-common/pkg/pubsub"
	"github.com/robodone/robosla-common/pkg/syncws"
)

var (
	TestAPIServer = "test1.robosla.com"
)

const (
	StatusOK    = "OK"
	StatusError = "Error"

	backlogSize = 10
)

func ConnectWS(apiServer string) (Conn, error) {
	url := fmt.Sprintf("wss://%s/device-api/v1", apiServer)
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %q: %v", url, err)
	}
	sock := syncws.NewSocket(conn)
	return newWSConn(sock), nil
}

type Impl interface {
	Hello(cookie string, resp *Response) error
}

type Conn interface {
	io.Closer

	In() <-chan string
	Send(data string) error
}

type Server struct {
	conn    Conn
	impl    Impl
	stopped chan bool
}

func NewServer(conn Conn, impl Impl) *Server {
	srv := &Server{conn: conn, impl: impl, stopped: make(chan bool, 1)}
	go srv.run()
	return srv
}

func (srv *Server) run() {
	for {
		select {
		case <-srv.stopped:
			return
		case msg := <-srv.conn.In():
			log.Printf("Server.Run, a message was received: %v", msg)
			srv.dispatch(msg)
		}
	}
}

func (srv *Server) replyUserError(userMessage string) {
	err := send(srv.conn, &Response{
		Status: StatusError,
		Error:  userMessage,
	})
	if err != nil {
		log.Printf("replyUserError(%q) failed: %v", userMessage, err)
	}
}

func (srv *Server) replyErr(err error) {
	log.Printf("replyErr(%v)", err)
	srv.replyUserError("backend error")
}

func (srv *Server) dispatch(msg string) {
	var req Request
	err := json.Unmarshal([]byte(msg), &req)
	if err != nil {
		srv.replyUserError("malformed request")
		return
	}
	var resp Response
	switch req.Cmd {
	case "":
		srv.replyUserError("command not set")
		return
	case "hello":
		err = srv.impl.Hello(req.Cookie, &resp)
	default:
		srv.replyUserError(fmt.Sprintf("unsupported command %q", req.Cmd))
		return
	}
	if err != nil {
		srv.replyErr(err)
		return
	}
	resp.Status = StatusOK
	err = send(srv.conn, &resp)
	if err != nil {
		log.Printf("failed to send a response: %v", err)
	}
}

func (srv *Server) Stop() error {
	// At this time, we don't want to take the ownership over connections.
	// May be, reconsider in the future.
	srv.stopped <- true
	return nil
}

type Request struct {
	Cmd    string `json:"cmd"`
	Cookie string `json:"cookie,omitempty"`
}

type Response struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
	Login  *Login `json:"login,omitempty"`
}

type Login struct {
	DeviceName string `json:"deviceName,omitempty"`
}

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

func (c *Client) Hello(cookie string) error {
	return send(c.conn, &Request{
		Cmd:    "hello",
		Cookie: cookie,
	})
}

func (c *Client) SubString(path string) (*pubsub.StringSub, error) {
	return c.nd.SubString(path)
}

func send(conn Conn, obj interface{}) error {
	data, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("failed to marshal a message to json: %v", err)
	}
	return conn.Send(string(data))
}

type WSConn struct {
	sock *syncws.Socket
	inCh chan string
}

func newWSConn(sock *syncws.Socket) *WSConn {
	wsc := &WSConn{
		sock: sock,
		inCh: make(chan string, backlogSize),
	}
	go wsc.run()
	return wsc
}

func (wsc *WSConn) run() {
	for {
		p, err := wsc.sock.ReadMessage()
		if err != nil {
			log.Printf("ReadMessage failed: %v. Stop listening for incoming messages.", err)
			return
		}
		log.Printf("A message received: %s", string(p))
		select {
		case wsc.inCh <- string(p):
		default:
			log.Printf("Incoming message dropped due to reaching the limit for backlog (%s messages)", backlogSize)
		}
	}
}

func (wsc *WSConn) Close() error {
	return wsc.sock.Close()
}

func (wsc *WSConn) In() <-chan string {
	return wsc.inCh
}

func (wsc *WSConn) Send(msg string) error {
	log.Printf("WSConn.Send(%s)", msg)
	return wsc.sock.WriteMessage([]byte(msg))
}
