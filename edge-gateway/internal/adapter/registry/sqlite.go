package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"edge-gateway/internal/adapter/store"
	"edge-gateway/internal/model/device"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository() *Repository {
	db, err := store.DB()
	if err != nil {
		panic(err)
	}
	registry, err := NewRepositoryWithDB(db)
	if err != nil {
		panic(err)
	}

	return registry
}

func NewRepositoryWithDB(db *gorm.DB) (*Repository, error) {
	if err := db.AutoMigrate(&deviceRecord{}); err != nil {
		return nil, fmt.Errorf("migrate devices table: %w", err)
	}

	return &Repository{db: db}, nil
}

func (r *Repository) Register(ctx context.Context, dev device.Device) error {
	if err := dev.ValidateForRegister(); err != nil {
		return err
	}

	row, err := newDeviceRecord(dev)
	if err != nil {
		return err
	}

	return r.db.Save(&row).Error
}

func (r *Repository) Get(ctx context.Context, id device.ID) (device.Device, error) {
	var row deviceRecord
	err := r.db.Where("device_id = ?", string(id)).First(&row).Error
	if err != nil {
		return device.Device{}, nil
	}

	return row.toModel()
}

func (r *Repository) List(ctx context.Context) ([]device.Device, error) {
	var rows []deviceRecord
	if err := r.db.Find(&rows).Error; err != nil {
		return nil, err
	}

	devices := make([]device.Device, 0, len(rows))
	for _, row := range rows {
		dev, err := row.toModel()
		if err != nil {
			return nil, err
		}
		devices = append(devices, dev)
	}

	return devices, nil
}

// deviceRecord 是 devices 表的 GORM 映射，只在 registry 持久化适配器内部使用。
type deviceRecord struct {
	ID               uint      `gorm:"primaryKey"`
	DeviceID         string    `gorm:"column:device_id;uniqueIndex;not null"`
	Name             string    `gorm:"column:name;not null"`
	DeviceType       string    `gorm:"column:device_type;not null"`
	Description      string    `gorm:"column:description"`
	SwVersion        string    `gorm:"column:sw_version"`
	PowerOnState     string    `gorm:"column:power_on_state"`
	CurrentStateJSON string    `gorm:"column:current_state_json;type:text"`
	CapabilitiesJSON string    `gorm:"column:capabilities_json;type:text"`
	LastSeenAt       time.Time `gorm:"column:last_seen_at"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (deviceRecord) TableName() string {
	return "devices"
}

func newDeviceRecord(dev device.Device) (deviceRecord, error) {
	currentState, err := json.Marshal(dev.CurrentState)
	if err != nil {
		return deviceRecord{}, fmt.Errorf("marshal current state: %w", err)
	}

	capabilities, err := json.Marshal(dev.Capabilities)
	if err != nil {
		return deviceRecord{}, fmt.Errorf("marshal capabilities: %w", err)
	}

	return deviceRecord{
		DeviceID:         string(dev.ID),
		Name:             dev.Name,
		DeviceType:       dev.Type,
		Description:      dev.Description,
		SwVersion:        dev.SwVersion,
		PowerOnState:     dev.PowerOnState,
		CurrentStateJSON: string(currentState),
		CapabilitiesJSON: string(capabilities),
		LastSeenAt:       time.Now(),
	}, nil
}

func (r deviceRecord) toModel() (device.Device, error) {
	var currentState map[string]any
	if r.CurrentStateJSON != "" {
		if err := json.Unmarshal([]byte(r.CurrentStateJSON), &currentState); err != nil {
			return device.Device{}, fmt.Errorf("unmarshal current state: %w", err)
		}
	}

	var capabilities []device.Capability
	if r.CapabilitiesJSON != "" {
		if err := json.Unmarshal([]byte(r.CapabilitiesJSON), &capabilities); err != nil {
			return device.Device{}, fmt.Errorf("unmarshal capabilities: %w", err)
		}
	}

	return device.Device{
		ID:           device.ID(r.DeviceID),
		Name:         r.Name,
		Type:         r.DeviceType,
		Description:  r.Description,
		SwVersion:    r.SwVersion,
		PowerOnState: r.PowerOnState,
		CurrentState: currentState,
		Capabilities: capabilities,
	}, nil
}
