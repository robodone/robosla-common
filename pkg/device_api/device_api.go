package device_api

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/robodone/robosla-common/pkg/autoupdate"
)

const (
	TestAPIServer = "test1.robosla.com"
	ProdAPIServer = "prod1.robosla.com"
)

func ChooseServer(version string) string {
	if autoupdate.IsDevBuild(version) {
		return TestAPIServer
	} else {
		return ProdAPIServer
	}
}

type Message struct {
	Type int
	Data []byte
}

type Conn interface {
	io.Closer

	In() <-chan *Message
	Send(data string) error
}

func send(conn Conn, obj interface{}) error {
	data, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("failed to marshal a message to json: %v", err)
	}
	return conn.Send(string(data))
}
