package transport

import (
	"context"

	"edge-gateway/device"
)

// Invoker 将设备指令分发到具体的设备通信通道。
type Invoker interface {
	Invoke(ctx context.Context, dev device.Device, command device.Command) (device.Result, error)
}

func NewInvoker() Invoker {
	return nil
}
