package device_api

const (
	StatusOK      = "OK"
	StatusError   = "Error"
	backlogSize   = 10
	TestAPIServer = "test1.robosla.com"
)

type Request struct {
	Cmd            string `json:"cmd"`
	Cookie         string `json:"cookie,omitempty"`
	TerminalOutput string `json:"terminalOutput,omitempty"`
}

type Response struct {
	Status string      `json:"status"`
	Error  string      `json:"error,omitempty"`
	Login  *Login      `json:"login,omitempty"`
	TS     *TimeSeries `json:"ts,omitempty"`
}

type Login struct {
	Cookie     string `json:"cookie"`
	DeviceName string `json:"deviceName,omitempty"`
}

type TimeSeries struct {
	Gcode []*StringSample `json:"gcode,omitempty"`
}

type StringSample struct {
	TS    int64  `json:ts"`
	Value string `json:"value"`
}
