package device

import (
	"encoding/json"
	"fmt"
	"math"
)

// ID 表示注册到网关中的设备标识。
type ID string

// Protocol 表示网关调用设备时使用的通信协议。
type Protocol string

const (
	ProtocolHTTPJSON Protocol = "http_json"
)

// ValueType 描述操作参数期望的 JSON 值类型。
type ValueType string

const (
	ValueString ValueType = "string"
	ValueInt    ValueType = "int"
	ValueBool   ValueType = "bool"
	ValueObject ValueType = "object"
	ValueArray  ValueType = "array"
)

// ParameterSpec 描述设备操作支持的单个参数。
type ParameterSpec struct {
	Name        string    `json:"name"`
	Type        ValueType `json:"type"`
	Description string    `json:"description,omitempty"`
}

// Capability 描述设备暴露给网关的一个操作。
type Capability struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  []ParameterSpec `json:"parameters,omitempty"`
}

// Device 是网关内部统一使用的设备实体。
type Device struct {
	ID           ID             `json:"id"`
	Name         string         `json:"name"`
	Type         string         `json:"type"`
	Description  string         `json:"description,omitempty"`
	SwVersion    string         `json:"sw_version,omitempty"`
	PowerOnState string         `json:"power_on_state,omitempty"`
	CurrentState map[string]any `json:"current_state,omitempty"`
	Capabilities []Capability   `json:"capabilities"`
}

// Command 是 LLM 产出的标准化设备调用指令。
type Command struct {
	DeviceID  ID             `json:"device_id"`
	Operation string         `json:"operation"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

// InvokeRequest 是网关发送给设备端的最小调用负载，方便 C 端解析。
type InvokeRequest struct {
	MessageID string         `json:"message_id,omitempty"`
	DeviceID  ID             `json:"device_id,omitempty"`
	Operation string         `json:"operation"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

// CommandAckMessage 是设备端给网关发回的指令执行 ACK 响应。
type CommandAckMessage struct {
	MessageID string `json:"message_id"`
	DeviceID  ID     `json:"device_id,omitempty"`
	Success   bool   `json:"success"`
	Output    any    `json:"output,omitempty"`
	Error     string `json:"error,omitempty"`
}

// InvokeResponse 是设备端返回给网关的推荐响应格式。
type InvokeResponse struct {
	Output any    `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
}

// Result 是网关执行设备指令后返回的结果。
type Result struct {
	DeviceID  ID     `json:"device_id"`
	Success   bool   `json:"success"`
	Operation string `json:"operation"`
	Output    any    `json:"output,omitempty"`
}

func (d Device) Capability(name string) (Capability, bool) {
	for _, capability := range d.Capabilities {
		if capability.Name == name {
			return capability, true
		}
	}
	return Capability{}, false
}

func (d Device) ValidateForRegister() error {
	if d.ID == "" {
		return fmt.Errorf("device id is required")
	}
	if d.Name == "" {
		return fmt.Errorf("device name is required")
	}
	if len(d.Capabilities) == 0 {
		return fmt.Errorf("device %q must expose at least one capability", d.ID)
	}

	seen := make(map[string]struct{}, len(d.Capabilities))
	for _, capability := range d.Capabilities {
		if capability.Name == "" {
			return fmt.Errorf("device %q has a capability without name", d.ID)
		}
		if _, ok := seen[capability.Name]; ok {
			return fmt.Errorf("device %q has duplicate capability %q", d.ID, capability.Name)
		}
		seen[capability.Name] = struct{}{}
	}
	return nil
}

func (d Device) ValidateCommand(cmd Command) error {
	if cmd.DeviceID != d.ID {
		return fmt.Errorf("command device %q does not match device %q", cmd.DeviceID, d.ID)
	}

	capability, ok := d.Capability(cmd.Operation)
	if !ok {
		return fmt.Errorf("device %q does not support capability %q", d.ID, cmd.Operation)
	}

	for _, param := range capability.Parameters {
		value, ok := cmd.Arguments[param.Name]
		if !ok {
			return fmt.Errorf("capability %q requires parameter %q", capability.Name, param.Name)
		}
		if err := validateValueType(param, value); err != nil {
			return fmt.Errorf("capability %q parameter %q: %w", capability.Name, param.Name, err)
		}
	}

	return nil
}

func (d Device) NewInvokeRequest(cmd Command) InvokeRequest {
	return InvokeRequest{
		DeviceID:  d.ID,
		Operation: cmd.Operation,
		Arguments: cmd.Arguments,
	}
}

func (d Device) NewInvokeRequestWithID(cmd Command, messageID string) InvokeRequest {
	return InvokeRequest{
		MessageID: messageID,
		DeviceID:  d.ID,
		Operation: cmd.Operation,
		Arguments: cmd.Arguments,
	}
}

func validateValueType(param ParameterSpec, value any) error {
	switch param.Type {
	case "":
		return nil
	case ValueString:
		if _, ok := value.(string); ok {
			return nil
		}
	case ValueInt:
		if isInteger(value) {
			return nil
		}
	case ValueBool:
		if _, ok := value.(bool); ok {
			return nil
		}
	case ValueObject:
		if _, ok := value.(map[string]any); ok {
			return nil
		}
	case ValueArray:
		if _, ok := value.([]any); ok {
			return nil
		}
	default:
		return fmt.Errorf("unsupported type %q", param.Type)
	}
	return fmt.Errorf("expected %s, got %T", param.Type, value)
}

func isInteger(value any) bool {
	switch v := value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	case float32:
		return math.Trunc(float64(v)) == float64(v)
	case float64:
		return math.Trunc(v) == v
	case json.Number:
		_, err := v.Int64()
		return err == nil
	default:
		return false
	}
}
