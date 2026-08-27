package gateway

import (
	"context"
	"time"

	"edge-gateway/internal/model/device"
)

// CommandRequest 是发给 LLM 的设备指令生成请求。
type CommandRequest struct {
	Utterance string          `json:"utterance"`
	Devices   []device.Device `json:"devices"`
}

// DeviceRepository 是网关应用服务访问设备集合的端口。
type DeviceRepository interface {
	Register(ctx context.Context, dev device.Device) error
	Get(ctx context.Context, id device.ID) (device.Device, error)
	List(ctx context.Context) ([]device.Device, error)
}

// CommandPlanner 负责根据用户文本和设备能力生成设备指令。
type CommandPlanner interface {
	BuildCommands(ctx context.Context, request CommandRequest) ([]device.Command, error)
}

// CommandInvoker 将设备指令分发到具体的设备通信通道。
type CommandInvoker interface {
	// 发送指令到信道中，支持指定超时时间
	Invoke(ctx context.Context, dev device.Device, commands []device.Command, timeout time.Duration) error
	// 确认指令是否执行成功，支持指定等待 ACK 的超时时间
	Confirm(ctx context.Context, dev device.Device, commands []device.Command, timeout time.Duration) ([]device.Result, error)
}
