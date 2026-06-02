package message

import "edge-gateway/internal/model/device"

type RegistMessage struct {
	MessageID string        `json:"message_id"`
	MsgType   string        `json:"msg_type"`
	Device    device.Device `json:"device"`
}
