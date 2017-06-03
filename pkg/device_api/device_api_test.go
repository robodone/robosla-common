package device_api

import (
	"errors"
	"testing"
	"time"
)

const backlogSize = 1

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
	to := make(chan string, backlogSize)
	from := make(chan string, backlogSize)
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
	defer func() {
		if err := srv.Stop(); err != nil {
			t.Errorf("Failed to stop the server: %v", err)
		}
	}()

	//t.Fatalf("TestHelloOK, 100")
	client := NewClient(conn1)
	defer client.Stop()

	sub, err := client.SubString("login.deviceName")
	if err != nil {
		t.Fatalf("Failed to subscribe for login/deviceName: %v", err)
	}
	defer sub.Unsub()

	//t.Fatalf("TestHelloOK, 200")
	if err := client.Hello(TestGoodCookie); err != nil {
		t.Fatalf("Failed to send hello: %v", err)
	}
	//t.Fatalf("TestHelloOK, 300")
	select {
	case deviceName := <-sub.C():
		if deviceName != TestDeviceName {
			t.Errorf("Wrong device name. Want: %s, got: %s", TestDeviceName, deviceName)
		}
	case <-time.Tick(time.Second):
		t.Errorf("Expected update not received")
	}
}
