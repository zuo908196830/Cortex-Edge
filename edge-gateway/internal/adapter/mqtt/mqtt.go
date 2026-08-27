package mqtt

import (
	"context"
	"sync"
	"time"

	mqttlib "github.com/pascaldekloe/mqtt"
)

type Mqtt struct {
	Client *mqttlib.Client
}

var (
	instance *Mqtt
	initErr  error
	once     sync.Once
)

func NewMqtt() (*Mqtt, error) {
	once.Do(func() {
		var client *mqttlib.Client
		client, initErr = mqttlib.VolatileSession("edge-gateway", &mqttlib.Config{
			Dialer:       mqttlib.NewDialer("tcp", "127.0.0.1:1883"),
			PauseTimeout: 4 * time.Second,
			CleanSession: true,
		})
		if initErr == nil {
			instance = &Mqtt{Client: client}
		}
	})
	return instance, initErr
}

func (m *Mqtt) Online() <-chan struct{} {
	return m.Client.Online()
}

func (m *Mqtt) Subscribe(ctx context.Context, timeout time.Duration, topics ...string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	return m.Client.Subscribe(ctx.Done(), topics...)
}

func (m *Mqtt) ReadSlices() (message, topic []byte, err error) {
	return m.Client.ReadSlices()
}

func (m *Mqtt) ReadBackoff(err error) <-chan struct{} {
	return m.Client.ReadBackoff(err)
}

func (m *Mqtt) Publish(ctx context.Context, message []byte, topic string, timeout time.Duration) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	return m.Client.Publish(ctx.Done(), message, topic)
}
