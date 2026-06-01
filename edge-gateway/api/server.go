package api

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"edge-gateway/api/model"
	"edge-gateway/gateway"

	"github.com/pascaldekloe/mqtt"
)

type Server struct {
	gateway *gateway.Gateway
	client  *mqtt.Client
}

func NewServer() *Server {
	client, err := mqtt.VolatileSession("demo-client", &mqtt.Config{
		Dialer:       mqtt.NewDialer("tcp", "mq1.example.com:1883"),
		PauseTimeout: 4 * time.Second,
		CleanSession: true,
	})
	if err != nil {
		panic(err)
	}

	return &Server{
		gateway: gateway.New(),
		client:  client,
	}
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

	go func(client *mqtt.Client) {
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
	var registMessage model.RegistMessage
	if err := json.NewDecoder(bytes.NewReader(message)).Decode(&registMessage); err != nil {
		log.Println("decode regist message error:", err)
		return
	}
	s.gateway.RegisterDevice(context.Background(), registMessage.Device)
}

func (s *Server) handleHeartbeatMessage(message []byte) {
}

func (s *Server) handleCommandAckMessage(message []byte) {
}

func (s *Server) handleUnknownTopic(topic, message []byte) {
}
