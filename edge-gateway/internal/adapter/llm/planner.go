package llm

import (
	"context"
	"fmt"
	"strings"

	"edge-gateway/internal/model/device"
	app "edge-gateway/internal/service/gateway"
)

type Planner struct{}

func NewPlanner() *Planner {
	return &Planner{}
}

func (Planner) BuildCommands(context.Context, app.CommandRequest) ([]device.Command, error) {
	return nil, fmt.Errorf("llm command planner is not configured")
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
		for index, op := range d.Capabilities {
			if index > 0 {
				builder.WriteString(",")
			}
			builder.WriteString(op.Name)
		}
		builder.WriteByte('\n')
	}
	return builder.String()
}
