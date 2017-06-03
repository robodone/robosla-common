package device_api

const (
	StatusOK      = "OK"
	StatusError   = "Error"
	backlogSize   = 10
	TestAPIServer = "test1.robosla.com"
)

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

type Impl interface {
	Hello(cookie string, resp *Response) error
}