package device_api

const (
	StatusOK    = "OK"
	StatusError = "Error"
	backlogSize = 30
)

type Request struct {
	Cmd            string `json:"cmd"`
	Cookie         string `json:"cookie,omitempty"`
	JobName        string `json:"jobName,omitempty"`
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
	TS    int64  `json:"ts"`
	Value string `json:"value"`
}

type UplinkMessage struct {
	Type       string  `json:"type"`
	JobName    string  `json:"jobName"`
	Success    bool    `json:"success"`
	Comment    string  `json:"comment"`
	Progress   float64 `json:"progress"`
	FrameIndex int     `json:"frameIndex"`
}
