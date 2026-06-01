package mqtt

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"edge-gateway/internal/adapter/llm"
	"edge-gateway/internal/adapter/registry"
	"edge-gateway/internal/adapter/speech"
	"edge-gateway/internal/adapter/transport"
	mqttmsg "edge-gateway/internal/api/mqtt/message"
	app "edge-gateway/internal/service/gateway"

	mqttlib "github.com/pascaldekloe/mqtt"
)

type Server struct {
	gateway *app.Service
	client  *mqttlib.Client
}

func NewServer() *Server {
	client, err := mqttlib.VolatileSession("demo-client", &mqttlib.Config{
		Dialer:       mqttlib.NewDialer("tcp", "mq1.example.com:1883"),
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
		registry.NewMemoryRepository(),
		speech.NewRecognizer(),
		llm.NewPlanner(),
		transport.NewInvoker(),
	)
}

func (s *Server) SubscribeMqtt() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := s.client.Subscribe(
		ctx.Done(),
		"edge/device/+/regist",
		"edge/device/+/heartbeat",
		"edge/device/+/command_ack",
	)
	if err != nil {
		panic(err)
	}

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
		s.handleUnknownTopic(topic, message)
		return
	}

	switch parts[3] {
	case "regist":
		s.handleRegistMessage(message)
	case "heartbeat":
		s.handleHeartbeatMessage(message)
	case "command_ack":
		s.handleCommandAckMessage(message)
	default:
		s.handleUnknownTopic(topic, message)
	}
}

func (s *Server) handleRegistMessage(message []byte) {
	var registMessage mqttmsg.RegistMessage
	if err := json.NewDecoder(bytes.NewReader(message)).Decode(&registMessage); err != nil {
		log.Println("decode regist message error:", err)
		return
	}
	if err := s.gateway.RegisterDevice(context.Background(), registMessage.Device); err != nil {
		log.Println("register device error:", err)
	}
}

func (s *Server) handleHeartbeatMessage(message []byte) {
}

func (s *Server) handleCommandAckMessage(message []byte) {
}

func (s *Server) handleUnknownTopic(topic, message []byte) {
}
