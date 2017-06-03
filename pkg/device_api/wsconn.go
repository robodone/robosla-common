package device_api

import (
	"fmt"
	"log"

	"github.com/gorilla/websocket"
	"github.com/robodone/robosla-common/pkg/syncws"
)

type WSConn struct {
	sock *syncws.Socket
	inCh chan string
}

func ConnectWS(apiServer string) (Conn, error) {
	url := fmt.Sprintf("wss://%s/device-api/v1", apiServer)
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %q: %v", url, err)
	}
	sock := syncws.NewSocket(conn)
	return newWSConn(sock), nil
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
