package gateway

import (
	"context"
	"fmt"
	"strings"

	"edge-gateway/internal/model/device"
	"edge-gateway/internal/model/voice"
)

// Service 编排注册、语音识别、意图解析和设备调用等用例。
type Service struct {
	devices DeviceRepository
	speech  SpeechRecognizer
	planner CommandPlanner
	invoker CommandInvoker
}

func NewService(devices DeviceRepository, speech SpeechRecognizer, planner CommandPlanner, invoker CommandInvoker) *Service {
	return &Service{devices: devices, speech: speech, planner: planner, invoker: invoker}
}

func (s *Service) RegisterDevice(ctx context.Context, dev device.Device) error {
	return s.devices.Register(ctx, dev)
}

func (s *Service) ListDevices(ctx context.Context) ([]device.Device, error) {
	return s.devices.List(ctx)
}

func (s *Service) HandleVoice(ctx context.Context, audio []byte, mimeType string) ([]device.Result, error) {
	if len(audio) == 0 {
		return nil, fmt.Errorf("audio is required")
	}
	if s.speech == nil {
		return nil, fmt.Errorf("speech recognizer is not configured")
	}

	transcript, err := s.speech.Transcribe(ctx, voice.Audio{
		Data:     audio,
		MIMEType: mimeType,
	})
	if err != nil {
		return nil, fmt.Errorf("transcribe voice: %w", err)
	}

	return s.HandleText(ctx, transcript.Text)
}

func (s *Service) HandleText(ctx context.Context, utterance string) ([]device.Result, error) {
	utterance = strings.TrimSpace(utterance)
	if utterance == "" {
		return nil, fmt.Errorf("utterance is required")
	}
	if s.planner == nil {
		return nil, fmt.Errorf("llm command planner is not configured")
	}

	devices, err := s.devices.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	if len(devices) == 0 {
		return nil, fmt.Errorf("no devices registered")
	}

	commands, err := s.planner.BuildCommands(ctx, CommandRequest{
		Utterance: utterance,
		Devices:   devices,
	})
	if err != nil {
		return nil, fmt.Errorf("build commands: %w", err)
	}

	return s.executeCommands(ctx, commands)
}

func (s *Service) executeCommands(ctx context.Context, commands []device.Command) ([]device.Result, error) {
	if s.invoker == nil {
		return nil, fmt.Errorf("device command invoker is not configured")
	}

	results := make([]device.Result, 0, len(commands))
	for _, command := range commands {
		dev, err := s.devices.Get(ctx, command.DeviceID)
		if err != nil {
			return results, fmt.Errorf("device %q is not registered, error: %w", command.DeviceID, err)
		}
		if err := dev.ValidateCommand(command); err != nil {
			return results, fmt.Errorf("validate command: %w", err)
		}

		result, err := s.invoker.Invoke(ctx, dev, command)
		if err != nil {
			return results, err
		}
		results = append(results, result)
	}

	return results, nil
}
