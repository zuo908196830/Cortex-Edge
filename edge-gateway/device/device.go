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
	ValueString  ValueType = "string"
	ValueInteger ValueType = "integer"
	ValueNumber  ValueType = "number"
	ValueBool    ValueType = "bool"
	ValueObject  ValueType = "object"
	ValueArray   ValueType = "array"
)

// ParameterSpec 描述设备操作支持的单个参数。
type ParameterSpec struct {
	Name        string    `json:"name"`
	Type        ValueType `json:"type"`
	Required    bool      `json:"required"`
	Description string    `json:"description,omitempty"`
}

// OperationSpec 描述设备暴露给网关的一个操作。
type OperationSpec struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  []ParameterSpec `json:"parameters,omitempty"`
}

// Device 是网关内部统一使用的设备实体。
type Device struct {
	ID          ID              `json:"id"`
	Name        string          `json:"name"`
	Protocol    Protocol        `json:"protocol,omitempty"`
	Endpoint    string          `json:"endpoint,omitempty"`
	Description string          `json:"description,omitempty"`
	Operations  []OperationSpec `json:"operations"`
}

// Normalize 填充设备实体中的默认值，减少 C 端注册时必须提供的字段。
func (d Device) Normalize() Device {
	if d.Protocol == "" {
		d.Protocol = ProtocolHTTPJSON
	}
	return d
}

// Command 是 LLM 产出的标准化设备调用指令。
type Command struct {
	DeviceID  ID             `json:"device_id"`
	Operation string         `json:"operation"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

// InvokeRequest 是网关发送给设备端的最小调用负载，方便 C 端解析。
type InvokeRequest struct {
	DeviceID  ID             `json:"device_id,omitempty"`
	Operation string         `json:"operation"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

// InvokeResponse 是设备端返回给网关的推荐响应格式。
type InvokeResponse struct {
	Output any    `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
}

// Result 是网关执行设备指令后返回的结果。
type Result struct {
	DeviceID  ID     `json:"device_id"`
	Operation string `json:"operation"`
	Output    any    `json:"output,omitempty"`
}

func (d Device) Operation(name string) (OperationSpec, bool) {
	for _, op := range d.Operations {
		if op.Name == name {
			return op, true
		}
	}
	return OperationSpec{}, false
}

func (d Device) ValidateForRegister() error {
	d = d.Normalize()
	if d.ID == "" {
		return fmt.Errorf("device id is required")
	}
	if d.Name == "" {
		return fmt.Errorf("device name is required")
	}
	if d.Protocol != ProtocolHTTPJSON {
		return fmt.Errorf("device %q protocol %q is not supported", d.ID, d.Protocol)
	}
	if d.Endpoint == "" {
		return fmt.Errorf("device %q endpoint is required", d.ID)
	}
	if len(d.Operations) == 0 {
		return fmt.Errorf("device %q must expose at least one operation", d.ID)
	}

	seen := make(map[string]struct{}, len(d.Operations))
	for _, op := range d.Operations {
		if op.Name == "" {
			return fmt.Errorf("device %q has an operation without name", d.ID)
		}
		if _, ok := seen[op.Name]; ok {
			return fmt.Errorf("device %q has duplicate operation %q", d.ID, op.Name)
		}
		seen[op.Name] = struct{}{}
	}
	return nil
}

func (d Device) ValidateCommand(cmd Command) error {
	if cmd.DeviceID != d.ID {
		return fmt.Errorf("command device %q does not match device %q", cmd.DeviceID, d.ID)
	}

	op, ok := d.Operation(cmd.Operation)
	if !ok {
		return fmt.Errorf("device %q does not support operation %q", d.ID, cmd.Operation)
	}

	for _, param := range op.Parameters {
		value, ok := cmd.Arguments[param.Name]
		if !ok {
			if param.Required {
				return fmt.Errorf("operation %q requires parameter %q", op.Name, param.Name)
			}
			continue
		}
		if err := validateValueType(param, value); err != nil {
			return fmt.Errorf("operation %q parameter %q: %w", op.Name, param.Name, err)
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

func validateValueType(param ParameterSpec, value any) error {
	switch param.Type {
	case "":
		return nil
	case ValueString:
		if _, ok := value.(string); ok {
			return nil
		}
	case ValueInteger:
		if isInteger(value) {
			return nil
		}
	case ValueNumber:
		if isNumber(value) {
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

func isNumber(value any) bool {
	switch value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, json.Number:
		return true
	default:
		return false
	}
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
