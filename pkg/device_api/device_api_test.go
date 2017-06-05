package device_api

import (
	"errors"
	"testing"
)

const testBacklogSize = 1

var (
	TestGoodCookie = "this cookie is good"
	TestDeviceName = "Test-001"
)

type TestConn struct {
	in  <-chan string
	out chan<- string
}

func (tc *TestConn) In() <-chan string {
	return tc.in
}

func (tc *TestConn) Send(data string) error {
	select {
	case tc.out <- data:
	default:
		return errors.New("failed to send a message due to a backlog")
	}
	return nil
}

func (tc *TestConn) Close() error {
	// It's the responsibility of the caller to not write into Out() if the connection was closed.
	close(tc.out)
	return nil
}

func newTestConnPair() (Conn, Conn) {
	to := make(chan string, testBacklogSize)
	from := make(chan string, testBacklogSize)
	return &TestConn{
			in:  to,
			out: from,
		}, &TestConn{
			in:  from,
			out: to,
		}
}

type TestServerImpl struct {
}

func (ts *TestServerImpl) Hello(cookie string, resp *Response) error {
	if cookie == TestGoodCookie {
		resp.Login = &Login{
			DeviceName: TestDeviceName,
		}
		return nil
	}
	return errors.New("bad cookie")
}

func (ts *TestServerImpl) RegisterDevice(cookie string, resp *Response) error {
	return errors.New("TestServerImpl.RegisterDevice: not implemented")
}

func (ts *TestServerImpl) SendTerminalOutput(out string, resp *Response) error {
	return errors.New("TestServerImpl.SendTerminalOutput: not implemented")
}

func (ts *TestServerImpl) RunPusher(closed <-chan bool, respCh chan<- *Response) {
	// We don't have any tests for push responses yet.
	close(respCh)
}

// TestHello opens a connection from the client to the server,
// the client sends a hello request with a cookie, which the server handles
// and issues a response. In one case, it's placing device name into the state
// (acknowledging that the cookie was accepted). In the other case, the server
// returns an error, and the client shuts the connection.
func TestHelloOK(t *testing.T) {
	conn0, conn1 := newTestConnPair()
	defer conn0.Close()
	defer conn1.Close()

	// Creating test server
	testSrvImpl := new(TestServerImpl)
	srv := NewServer(conn0, testSrvImpl)
	go srv.Run()
	defer func() {
		if err := srv.Stop(); err != nil {
			t.Errorf("Failed to stop the server: %v", err)
		}
	}()

	client := NewClient(conn1)
	defer client.Stop()

	deviceName, err := client.Hello(TestGoodCookie)
	if err != nil {
		t.Errorf("Hello(%q) failed: %v", TestGoodCookie, err)
	}
	if deviceName != TestDeviceName {
		t.Errorf("Wrong device name. Want: %s, got: %s", TestDeviceName, deviceName)
	}
}
