package gateway

import (
	"context"
	"fmt"
	"strings"

	"edge-gateway/device"
	"edge-gateway/llm"
	"edge-gateway/registry"
	"edge-gateway/speech"
	"edge-gateway/transport"
)

type Gateway struct {
	registry registry.Registry
	speech   speech.Recognizer
	planner  llm.CommandPlanner
	invoker  transport.Invoker
}

func New() *Gateway {
	registry := registry.NewMemoryRegistry()
	speech := speech.NewRecognizer()
	planner := llm.NewCommandPlanner()
	invoker := transport.NewInvoker()

	return &Gateway{registry: registry, speech: speech, planner: planner, invoker: invoker}
}

func (g *Gateway) RegisterDevice(ctx context.Context, device device.Device) error {
	if err := g.registry.Regist(ctx, device); err != nil {
		return err
	}

	return nil
}

func (g *Gateway) ListDevices(ctx context.Context) ([]device.Device, error) {
	return g.registry.List(ctx)
}

func (g *Gateway) HandleVoice(ctx context.Context, audio []byte, mimeType string) ([]device.Result, error) {
	if len(audio) == 0 {
		return nil, fmt.Errorf("audio is required")
	}

	transcript, err := g.speech.Transcribe(ctx, speech.Audio{
		Data:     audio,
		MIMEType: mimeType,
	})
	if err != nil {
		return nil, fmt.Errorf("transcribe voice: %w", err)
	}

	return g.HandleText(ctx, transcript.Text)
}

func (g *Gateway) HandleText(ctx context.Context, utterance string) ([]device.Result, error) {
	utterance = strings.TrimSpace(utterance)
	if utterance == "" {
		return nil, fmt.Errorf("utterance is required")
	}

	devices, err := g.registry.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	if len(devices) == 0 {
		return nil, fmt.Errorf("no devices registered")
	}

	commands, err := g.planner.BuildCommands(ctx, llm.CommandRequest{
		Utterance: utterance,
		Devices:   devices,
	})
	if err != nil {
		return nil, fmt.Errorf("build commands: %w", err)
	}

	return g.executeCommands(ctx, commands)
}

func (g *Gateway) executeCommands(ctx context.Context, commands []device.Command) ([]device.Result, error) {
	results := make([]device.Result, 0, len(commands))
	for _, command := range commands {
		dev, ok := g.registry.Get(ctx, command.DeviceID)
		if !ok {
			return results, fmt.Errorf("device %q is not registered", command.DeviceID)
		}
		if err := dev.ValidateCommand(command); err != nil {
			return results, err
		}

		result, err := g.invoker.Invoke(ctx, dev, command)
		if err != nil {
			return results, err
		}
		results = append(results, result)
	}

	return results, nil
}
