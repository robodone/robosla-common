package device_api

import (
	"encoding/json"
	"fmt"
	"io"
)

type Conn interface {
	io.Closer

	In() <-chan string
	Send(data string) error
}

func send(conn Conn, obj interface{}) error {
	data, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("failed to marshal a message to json: %v", err)
	}
	return conn.Send(string(data))
}
