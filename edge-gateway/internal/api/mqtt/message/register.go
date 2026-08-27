package message

import "edge-gateway/internal/model/device"

type RegistMessage struct {
	MessageID string        `json:"message_id"`
	MsgType   string        `json:"msg_type"`
	Device    device.Device `json:"device"`
}

type TextMessage struct {
	LocationID      string `json:"location_id"`
	PermissionLevel int    `json:"permission_level"`
	Text            string `json:"text"`
}
