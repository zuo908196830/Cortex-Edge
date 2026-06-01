package registry

import (
	"context"
	"log"
	"sync"

	"edge-gateway/device"
)

// Registry 保存网关可用的设备能力。
type Registry interface {
	Regist(ctx context.Context, dev device.Device) error
	Get(ctx context.Context, id device.ID) (device.Device, bool)
	List(ctx context.Context) ([]device.Device, error)
}

type MemoryRegistry struct {
	mu      sync.RWMutex
	devices map[device.ID]device.Device
}

var registry *MemoryRegistry

func NewMemoryRegistry() *MemoryRegistry {
	once := sync.Once{}
	once.Do(func() {
		registry = &MemoryRegistry{devices: make(map[device.ID]device.Device)}
	})

	return registry
}

func (r *MemoryRegistry) Regist(_ context.Context, dev device.Device) error {
	if err := dev.ValidateForRegister(); err != nil {
		log.Fatalf("validate device for register: %v\n", err)
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
