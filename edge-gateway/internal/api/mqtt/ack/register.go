package ack

import "encoding/json"

type RegisterAckMessage struct {
	MessageID string `json:"message_id"`
	Success   bool   `json:"success"`
	Message   string `json:"message"`
}

func BuildRegisterAckMessage(messageID string, success bool, message string) []byte {
	msg, _ := json.Marshal(RegisterAckMessage{
		MessageID: messageID,
		Success:   success,
		Message:   message,
	})
	return msg
}
