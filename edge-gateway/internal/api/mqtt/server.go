package mqtt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"edge-gateway/internal/adapter/llm"
	"edge-gateway/internal/adapter/registry"
	"edge-gateway/internal/adapter/speech"
	"edge-gateway/internal/adapter/transport"
	"edge-gateway/internal/api/mqtt/ack"
	mqttmsg "edge-gateway/internal/api/mqtt/message"
	app "edge-gateway/internal/service/gateway"

	mqttlib "github.com/pascaldekloe/mqtt"
)

type Server struct {
	gateway *app.Service
	client  *mqttlib.Client
}

func NewServer() *Server {
	client, err := mqttlib.VolatileSession("edge-gateway", &mqttlib.Config{
		Dialer:       mqttlib.NewDialer("tcp", "127.0.0.1:1883"),
		PauseTimeout: 4 * time.Second,
		CleanSession: true,
	})
	if err != nil {
		panic(err)
	}

	return NewServerWithClient(defaultGatewayService(), client)
}

func NewServerWithClient(gateway *app.Service, client *mqttlib.Client) *Server {
	return &Server{
		gateway: gateway,
		client:  client,
	}
}

func defaultGatewayService() *app.Service {
	return app.NewService(
		registry.NewRepository(),
		speech.NewRecognizer(),
		llm.NewPlanner(),
		transport.NewInvoker(),
	)
}

func (s *Server) SubscribeMqtt() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s.startReadLoop()
	select {
	case <-s.client.Online():
	case <-ctx.Done():
		panic(fmt.Errorf("connect mqtt broker: %w", ctx.Err()))
	}

	err := s.client.Subscribe(
		ctx.Done(),
		"edge/device/+/regist",
		"edge/device/+/heartbeat",
		"edge/device/+/command_ack",
	)
	if err != nil {
		panic(err)
	}
	log.Println("subscribe mqtt topics success")
}

func (s *Server) startReadLoop() {
	go func(client *mqttlib.Client) {
		for {
			message, topic, err := client.ReadSlices()
			if err != nil {
				wait := client.ReadBackoff(err)
				if wait == nil {
					return
				}
				<-wait
				continue
			}
			s.dispatchMqttMessage(topic, message)
		}
	}(s.client)
}

func (s *Server) dispatchMqttMessage(topic, message []byte) {
	parts := strings.Split(string(topic), "/")
	if len(parts) != 4 || parts[0] != "edge" || parts[1] != "device" {
		s.handleUnknownTopic("", topic, message)
		return
	}

	deviceID := parts[2]

	switch parts[3] {
	case "regist":
		s.handleRegistMessage(deviceID, message)
	case "heartbeat":
		s.handleHeartbeatMessage(deviceID, message)
	case "command_ack":
		s.handleCommandAckMessage(deviceID, message)
	default:
		s.handleUnknownTopic(deviceID, topic, message)
	}
}

func (s *Server) handleRegistMessage(deviceID string, message []byte) {
	var registMessage mqttmsg.RegistMessage
	if err := json.NewDecoder(bytes.NewReader(message)).Decode(&registMessage); err != nil {
		log.Println("decode regist message error:", err)
		return
	}
	ackMsg := ack.BuildRegisterAckMessage(registMessage.MessageID, true, "success")
	if err := s.gateway.RegisterDevice(context.Background(), registMessage.Device); err != nil {
		log.Println("register device error:", err)
		ackMsg = ack.BuildRegisterAckMessage(registMessage.MessageID, false, err.Error())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	s.client.Publish(ctx.Done(), ackMsg, fmt.Sprintf("edge/device/%s/regist_ack", deviceID))
}

func (s *Server) handleHeartbeatMessage(deviceID string, message []byte) {
}

func (s *Server) handleCommandAckMessage(deviceID string, message []byte) {
}

func (s *Server) handleUnknownTopic(deviceID string, topic, message []byte) {
}
