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
	mqttadapter "edge-gateway/internal/adapter/mqtt"
	"edge-gateway/internal/adapter/registry"
	"edge-gateway/internal/adapter/transport"
	"edge-gateway/internal/api/mqtt/ack"
	mqttmsg "edge-gateway/internal/api/mqtt/message"
	"edge-gateway/internal/model/device"
	app "edge-gateway/internal/service/gateway"
)

type Server struct {
	gateway    *app.Service
	mqttClient *mqttadapter.Mqtt
	invoker    *transport.Invoker
}

func NewServer() *Server {
	mqttAdapter, err := mqttadapter.NewMqtt()
	if err != nil {
		panic(err)
	}

	invoker := transport.NewInvokerWithMqtt(mqttAdapter)
	gateway := app.NewService(
		registry.NewRepository(),
		llm.NewPlanner(),
		invoker,
	)

	return &Server{
		gateway:    gateway,
		mqttClient: mqttAdapter,
		invoker:    invoker,
	}
}

func NewServerWithAdapter(gateway *app.Service, mqttAdapter *mqttadapter.Mqtt) *Server {
	return &Server{
		gateway:    gateway,
		mqttClient: mqttAdapter,
		invoker:    transport.NewInvokerWithMqtt(mqttAdapter),
	}
}

func NewServerWithClient(gateway *app.Service, mqttAdapter *mqttadapter.Mqtt) *Server {
	return NewServerWithAdapter(gateway, mqttAdapter)
}

func defaultGatewayService() *app.Service {
	return app.NewService(
		registry.NewRepository(),
		llm.NewPlanner(),
		transport.NewInvoker(),
	)
}

func (s *Server) SubscribeMqtt() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s.startReadLoop()
	select {
	case <-s.mqttClient.Online():
	case <-ctx.Done():
		panic(fmt.Errorf("connect mqtt broker: %w", ctx.Err()))
	}

	err := s.mqttClient.Subscribe(
		ctx,
		0,
		"edge/device/+/regist",
		"edge/device/+/heartbeat",
		"edge/device/+/command_ack",
		"edge/device/+/message",
	)
	if err != nil {
		panic(err)
	}
	log.Println("subscribe mqtt topics success")
}

func (s *Server) startReadLoop() {
	go func(client *mqttadapter.Mqtt) {
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
	}(s.mqttClient)
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
	case "message":
		s.handleMessageMessage(message)
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
	s.mqttClient.Publish(context.Background(), ackMsg, fmt.Sprintf("edge/device/%s/regist_ack", deviceID), 10*time.Second)
}

func (s *Server) handleHeartbeatMessage(deviceID string, message []byte) {
}

func (s *Server) handleCommandAckMessage(deviceID string, message []byte) {
	var ackMsg device.CommandAckMessage
	if err := json.NewDecoder(bytes.NewReader(message)).Decode(&ackMsg); err != nil {
		log.Println("decode command ack message error:", err)
		return
	}
	if ackMsg.DeviceID == "" {
		ackMsg.DeviceID = device.ID(deviceID)
	}
	if s.invoker != nil {
		s.invoker.HandleAck(ackMsg)
	}
}

func (s *Server) handleUnknownTopic(deviceID string, topic, message []byte) {
}

func (s *Server) handleMessageMessage(message []byte) {
	var text mqttmsg.TextMessage
	if err := json.NewDecoder(bytes.NewReader(message)).Decode(&text); err != nil {
		log.Println("decode text message error:", err)
		return
	}
	s.gateway.HandleText(context.Background(), text.Text)
}
