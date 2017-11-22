package opapi

import "time"

type Printer struct {
	ID          int64  `json:"id,omitempty"`
	DeviceName  string `json:"deviceName,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	Status      string `json:"status,omitempty"`
	ResinHexID  string `json:"resinHexID,omitempty"`
}

type PrinterForFrameIndex struct {
	FrameIndex int `json:"frameIndex"`
	NumFrames  int `json:"numFrames"`
}

type PrinterForProgress struct {
	Progress  float64       `json:"progress"`
	Elapsed   time.Duration `json:"elapsed"`
	Remaining time.Duration `json:"remaining"`
}

type PrinterForVideo struct {
	VideoURL string `json:"videoURL"`
}

type PrinterForMovingState struct {
	MovingState string `json:"movingState"`
}

type PrinterForGripperState struct {
	GripperState string `json:"gripperState"`
}

type PrinterForCameras struct {
	Cameras map[string]string `json:"cameras"`
}
