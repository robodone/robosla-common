package syncws

import (
	"log"
	"net"
	"sync"

	"github.com/gorilla/websocket"
)

type Socket struct {
	wmu  sync.Mutex
	rmu  sync.Mutex
	conn *websocket.Conn
}

func NewSocket(conn *websocket.Conn) *Socket {
	return &Socket{conn: conn}
}

func (s *Socket) WriteMessage(data []byte) error {
	s.wmu.Lock()
	defer s.wmu.Unlock()
	err := s.conn.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		log.Printf("Failed to write to a websocket: %v", err)
	}
	return err
}

func (s *Socket) ReadMessage() ([]byte, error) {
	s.rmu.Lock()
	defer s.rmu.Unlock()
	for {
		messageType, p, err := s.conn.ReadMessage()
		if err != nil {
			return nil, err
		}
		if messageType != websocket.TextMessage {
			log.Printf("messageType: %d, ignoring...\n", messageType)
			continue
		}
		return p, nil
	}
}

func (s *Socket) Close() error {
	return s.conn.Close()
}

func (s *Socket) RemoteAddr() net.Addr {
	return s.conn.RemoteAddr()
}
