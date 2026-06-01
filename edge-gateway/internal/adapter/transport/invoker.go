package transport

import (
	"context"
	"fmt"

	"edge-gateway/internal/model/device"
)

type Invoker struct{}

func NewInvoker() *Invoker {
	return &Invoker{}
}

func (Invoker) Invoke(context.Context, device.Device, device.Command) (device.Result, error) {
	return device.Result{}, fmt.Errorf("device command invoker is not configured")
}
