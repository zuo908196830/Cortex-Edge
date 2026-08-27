package gateway

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"edge-gateway/internal/model/device"
)

// Service 编排注册、语音识别、意图解析和设备调用等用例。
type Service struct {
	devices DeviceRepository
	planner CommandPlanner
	invoker CommandInvoker
}

func NewService(devices DeviceRepository, planner CommandPlanner, invoker CommandInvoker) *Service {
	return &Service{devices: devices, planner: planner, invoker: invoker}
}

func (s *Service) RegisterDevice(ctx context.Context, dev device.Device) error {
	return s.devices.Register(ctx, dev)
}

func (s *Service) ListDevices(ctx context.Context) ([]device.Device, error) {
	return s.devices.List(ctx)
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
	if len(commands) == 0 {
		return nil, nil
	}

	// 1. 按 DeviceID 归类指令
	commandMap := make(map[device.ID][]device.Command)
	for _, command := range commands {
		commandMap[command.DeviceID] = append(commandMap[command.DeviceID], command)
	}

	var (
		wg         sync.WaitGroup
		mu         sync.Mutex
		resultsMap = make(map[device.ID][]device.Result, len(commandMap))
		errs       []error
	)

	// 2. 并发执行各个设备的指令
	for devID, devCmds := range commandMap {
		wg.Add(1)
		go func(deviceID device.ID, commands []device.Command) {
			defer wg.Done()

			dev, err := s.devices.Get(ctx, deviceID)
			if err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("get device %s: %w", deviceID, err))
				mu.Unlock()
				return
			}

			for _, command := range commands {
				if err := dev.ValidateCommand(command); err != nil {
					mu.Lock()
					errs = append(errs, fmt.Errorf("validate command for device %s: %w", deviceID, err))
					mu.Unlock()
					return
				}
			}

			if err := s.invoker.Invoke(ctx, dev, commands, 5*time.Second); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("invoke commands for device %s: %w", deviceID, err))
				mu.Unlock()
				return
			}

			resList, err := s.invoker.Confirm(ctx, dev, commands, 10*time.Second)
			if err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("confirm commands for device %s: %w", deviceID, err))
				mu.Unlock()
				return
			}

			mu.Lock()
			resultsMap[deviceID] = resList
			mu.Unlock()
		}(devID, devCmds)
	}

	wg.Wait()

	// 3. 检查是否有执行/校验失败
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	// 4. 汇总执行结果
	var allResults []device.Result
	for deviceID, resList := range resultsMap {
		for _, result := range resList {
			if !result.Success {
				return nil, fmt.Errorf("command execution failed for device %q", deviceID)
			}
			allResults = append(allResults, result)
		}
	}

	return allResults, nil
}
