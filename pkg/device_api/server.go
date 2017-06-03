package device_api

import (
	"encoding/json"
	"fmt"
	"log"
)

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