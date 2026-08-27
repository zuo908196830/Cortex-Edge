package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"edge-gateway/internal/adapter/mqtt"
	"edge-gateway/internal/model/device"
)

type pendingCmd struct {
	operation string
	messageID string
	ackChan   chan device.CommandAckMessage
}

type Invoker struct {
	mqtt      *mqtt.Mqtt
	mu        sync.Mutex
	pending   map[string]chan device.CommandAckMessage
	devMsgIDs map[device.ID][]pendingCmd
}

var (
	invokerInstance *Invoker
	invokerOnce     sync.Once
)

func NewInvoker() *Invoker {
	invokerOnce.Do(func() {
		mqttAdapter, err := mqtt.NewMqtt()
		if err != nil {
			panic(err)
		}
		invokerInstance = &Invoker{
			mqtt:      mqttAdapter,
			pending:   make(map[string]chan device.CommandAckMessage),
			devMsgIDs: make(map[device.ID][]pendingCmd),
		}
	})
	return invokerInstance
}

func NewInvokerWithMqtt(mqttAdapter *mqtt.Mqtt) *Invoker {
	invokerOnce.Do(func() {
		if mqttAdapter == nil {
			var err error
			mqttAdapter, err = mqtt.NewMqtt()
			if err != nil {
				panic(err)
			}
		}
		invokerInstance = &Invoker{
			mqtt:      mqttAdapter,
			pending:   make(map[string]chan device.CommandAckMessage),
			devMsgIDs: make(map[device.ID][]pendingCmd),
		}
	})
	return invokerInstance
}

func (i *Invoker) Invoke(ctx context.Context, dev device.Device, commands []device.Command, timeout time.Duration) error {
	if i.mqtt == nil {
		return fmt.Errorf("mqtt client is not initialized")
	}

	for idx, cmd := range commands {
		msgID := fmt.Sprintf("cmd-%s-%s-%d-%d", dev.ID, cmd.Operation, time.Now().UnixNano(), idx)
		ackChan := make(chan device.CommandAckMessage, 1)

		i.mu.Lock()
		i.pending[msgID] = ackChan
		i.devMsgIDs[dev.ID] = append(i.devMsgIDs[dev.ID], pendingCmd{
			operation: cmd.Operation,
			messageID: msgID,
			ackChan:   ackChan,
		})
		i.mu.Unlock()

		req := dev.NewInvokeRequestWithID(cmd, msgID)
		payload, err := json.Marshal(req)
		if err != nil {
			return fmt.Errorf("marshal invoke request for device %s operation %s: %w", dev.ID, cmd.Operation, err)
		}

		topic := fmt.Sprintf("edge/device/%s/command", dev.ID)
		if err := i.mqtt.Publish(ctx, payload, topic, timeout); err != nil {
			return fmt.Errorf("publish command for device %s operation %s: %w", dev.ID, cmd.Operation, err)
		}
	}
	return nil
}

func (i *Invoker) Confirm(ctx context.Context, dev device.Device, commands []device.Command, timeout time.Duration) ([]device.Result, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	i.mu.Lock()
	pCmds := i.devMsgIDs[dev.ID]
	delete(i.devMsgIDs, dev.ID)
	i.mu.Unlock()

	results := make([]device.Result, 0, len(commands))

	for _, p := range pCmds {
		select {
		case ackMsg := <-p.ackChan:
			res := device.Result{
				DeviceID:  dev.ID,
				Success:   ackMsg.Success,
				Operation: p.operation,
				Output:    ackMsg.Output,
			}
			if !ackMsg.Success && ackMsg.Error != "" {
				res.Output = ackMsg.Error
			}
			results = append(results, res)
		case <-ctx.Done():
			results = append(results, device.Result{
				DeviceID:  dev.ID,
				Success:   false,
				Operation: p.operation,
				Output:    fmt.Sprintf("ack timeout: %v", ctx.Err()),
			})
		}

		i.mu.Lock()
		delete(i.pending, p.messageID)
		i.mu.Unlock()
	}
	return results, nil
}

func (i *Invoker) HandleAck(ackMsg device.CommandAckMessage) {
	i.mu.Lock()
	ch, ok := i.pending[ackMsg.MessageID]
	i.mu.Unlock()

	if ok {
		select {
		case ch <- ackMsg:
		default:
		}
	}
}
