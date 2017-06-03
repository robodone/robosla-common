package device_api

import (
	"fmt"

	"github.com/gorilla/websocket"
	"github.com/robodone/robosla-common/pkg/syncws"
)

var (
	TestAPIServer = "test1.robosla.com"
)

func Connect(apiServer string) (*syncws.Socket, error) {
	url := fmt.Sprintf("wss://%s/device-api/v1", apiServer)
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %q: %v", url, err)
	}
	return syncws.NewSocket(conn), nil
}
