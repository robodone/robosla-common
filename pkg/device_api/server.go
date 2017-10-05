package device_api

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/gorilla/websocket"
)

type Impl interface {
	RegisterDevice(cookie string, resp *Response) error
	Hello(cookie, jobName string, resp *Response) error
	SendTerminalOutput(out string, resp *Response) error
}

type Server struct {
	conn    Conn
	impl    Impl
	stopped chan bool
}

func NewServer(conn Conn, impl Impl) *Server {
	srv := &Server{conn: conn, impl: impl, stopped: make(chan bool)}
	return srv
}

func (srv *Server) SendBack(resp *Response) error {
	return send(srv.conn, resp)
}

func (srv *Server) Run() error {
	for {
		select {
		case <-srv.stopped:
			return nil
		case msg, ok := <-srv.conn.In():
			if !ok {
				return nil
			}
			if msg.Type != websocket.TextMessage {
				log.Printf("Unexpected message type %d, ignoring...", msg.Type)
			}
			tolog := string(msg.Data)
			if len(tolog) > 500 {
				tolog = tolog[:300] + "<...-truncated-...>"
			}
			log.Printf("Server.Run, a message was received: %s", tolog)
			srv.dispatch(string(msg.Data))
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
	case "register-device":
		err = srv.impl.RegisterDevice(req.Cookie, &resp)
	case "hello":
		err = srv.impl.Hello(req.Cookie, req.JobName, &resp)
	case "send-terminal-output":
		err = srv.impl.SendTerminalOutput(req.TerminalOutput, &resp)
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
	close(srv.stopped)
	return nil
}
