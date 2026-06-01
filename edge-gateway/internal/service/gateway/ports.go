package gateway

import (
	"context"

	"edge-gateway/internal/model/device"
	"edge-gateway/internal/model/voice"
)

// CommandRequest 是发给 LLM 的设备指令生成请求。
type CommandRequest struct {
	Utterance string          `json:"utterance"`
	Devices   []device.Device `json:"devices"`
}

// DeviceRepository 是网关应用服务访问设备集合的端口。
type DeviceRepository interface {
	Register(ctx context.Context, dev device.Device) error
	Get(ctx context.Context, id device.ID) (device.Device, bool)
	List(ctx context.Context) ([]device.Device, error)
}

// CommandPlanner 负责根据用户文本和设备能力生成设备指令。
type CommandPlanner interface {
	BuildCommands(ctx context.Context, request CommandRequest) ([]device.Command, error)
}

// SpeechRecognizer 负责把语音转成文本，可以接 ASR 服务或支持音频输入的 LLM。
type SpeechRecognizer interface {
	Transcribe(ctx context.Context, audio voice.Audio) (voice.Transcript, error)
}

// CommandInvoker 将设备指令分发到具体的设备通信通道。
type CommandInvoker interface {
	Invoke(ctx context.Context, dev device.Device, command device.Command) (device.Result, error)
}
