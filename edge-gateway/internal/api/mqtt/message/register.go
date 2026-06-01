package message

import "edge-gateway/internal/model/device"

type RegistMessage struct {
	MsgType string        `json:"msg_type"`
	Device  device.Device `json:"device"`
}
