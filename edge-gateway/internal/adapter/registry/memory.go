package registry

import (
	"context"
	"sync"

	"edge-gateway/internal/model/device"
)

type MemoryRepository struct {
	mu      sync.RWMutex
	devices map[device.ID]device.Device
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{devices: make(map[device.ID]device.Device)}
}

func (r *MemoryRepository) Register(_ context.Context, dev device.Device) error {
	if err := dev.ValidateForRegister(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.devices[dev.ID] = dev

	return nil
}

func (r *MemoryRepository) Get(_ context.Context, id device.ID) (device.Device, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	dev, ok := r.devices[id]

	return dev, ok
}

func (r *MemoryRepository) List(_ context.Context) ([]device.Device, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]device.Device, 0, len(r.devices))
	for _, dev := range r.devices {
		items = append(items, dev)
	}

	return items, nil
}
