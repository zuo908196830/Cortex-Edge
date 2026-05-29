package registry

import (
	"context"
	"sync"

	"edge-gateway/device"
)

// Registry 保存网关可用的设备能力。
type Registry interface {
	Register(ctx context.Context, dev device.Device) error
	Get(ctx context.Context, id device.ID) (device.Device, bool)
	List(ctx context.Context) ([]device.Device, error)
}

type MemoryRegistry struct {
	mu      sync.RWMutex
	devices map[device.ID]device.Device
}

func NewMemoryRegistry() *MemoryRegistry {
	return &MemoryRegistry{devices: make(map[device.ID]device.Device)}
}

func (r *MemoryRegistry) Register(_ context.Context, dev device.Device) error {
	dev = dev.Normalize()
	if err := dev.ValidateForRegister(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.devices[dev.ID] = dev
	return nil
}

func (r *MemoryRegistry) Get(_ context.Context, id device.ID) (device.Device, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	dev, ok := r.devices[id]
	return dev, ok
}

func (r *MemoryRegistry) List(_ context.Context) ([]device.Device, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]device.Device, 0, len(r.devices))
	for _, dev := range r.devices {
		items = append(items, dev)
	}
	return items, nil
}
