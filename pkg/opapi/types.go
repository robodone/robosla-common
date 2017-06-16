package opapi

type Printer struct {
	ID          int64  `json:"id,omitempty"`
	DeviceName  string `json:"deviceName,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	Status      string `json:"status,omitempty"`
}

type PrinterForFrameIndex struct {
	FrameIndex int `json:"frameIndex"`
}

type PrinterForProgress struct {
	Progress float64 `json:"progress"`
}

type PrinterForVideo struct {
	VideoURL string `json:"videoURL"`
}
