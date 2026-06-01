package model

import "edge-gateway/device"

type RegistMessage struct {
	MsgType string        `json:"msg_type"`
	Device  device.Device `json:"device"`
}
