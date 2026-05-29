package llm

import (
	"context"
	"strings"

	"edge-gateway/device"
)

// CommandRequest 是发给 LLM 的设备指令生成请求。
type CommandRequest struct {
	Utterance string          `json:"utterance"`
	Devices   []device.Device `json:"devices"`
}

// CommandPlanner 负责根据用户文本和设备能力生成设备指令。
type CommandPlanner interface {
	BuildCommands(ctx context.Context, request CommandRequest) ([]device.Command, error)
}

// CapabilityPrompt 返回设备操作能力的紧凑文本描述。
func CapabilityPrompt(devices []device.Device) string {
	var builder strings.Builder
	for _, d := range devices {
		builder.WriteString("device=")
		builder.WriteString(string(d.ID))
		builder.WriteString(" name=")
		builder.WriteString(d.Name)
		builder.WriteString(" operations=")
		for index, op := range d.Operations {
			if index > 0 {
				builder.WriteString(",")
			}
			builder.WriteString(op.Name)
		}
		builder.WriteByte('\n')
	}
	return builder.String()
}
