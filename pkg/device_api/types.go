package device_api

import "time"

const (
	StatusOK    = "OK"
	StatusError = "Error"
	backlogSize = 30
)

type Request struct {
	Cmd     string         `json:"cmd"`
	Cookie  string         `json:"cookie,omitempty"`
	JobName string         `json:"jobName,omitempty"`
	Msg     *UplinkMessage `json:"msg,omitempty"`
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
	Type           string        `json:"type"`
	JobName        string        `json:"jobName"`
	Success        bool          `json:"success"`
	Comment        string        `json:"comment"`
	Elapsed        time.Duration `json:elapsed"`
	Remaining      time.Duration `json:duration"`
	Progress       float64       `json:"progress"`
	FrameIndex     int           `json:"frameIndex"`
	NumFrames      int           `json:"numFrames"`
	MovingState    string        `json:"movingState"`
	GripperState   string        `json:"gripperState"`
	TerminalOutput string        `json:"terminalOutput,omitempty"`

	// Data URL-encoded camera frames saved by their respective names.
	Cameras map[string]string `json:"cameras"`
}
