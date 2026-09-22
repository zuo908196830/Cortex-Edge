package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gorilla/websocket"
)

// Embed static files for single standalone binary execution
//
//go:embed static/*
var staticFS embed.FS

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for local dev tool
	},
}

// Command sent from Browser to Backend
type ClientCommand struct {
	Action string                 `json:"action"`
	Data   map[string]interface{} `json:"data"`
}

// Event sent from Backend to Browser
type BackendEvent struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

// Session holds state for a Web UI client session
type Session struct {
	wsConn     *websocket.Conn
	mqttClient mqtt.Client
	mu         sync.Mutex
}

func main() {
	// Root static file server
	subFS, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatalf("Failed to create sub filesystem: %v", err)
	}

	http.Handle("/", http.FileServer(http.FS(subFS)))
	http.HandleFunc("/ws", handleWebSocket)

	port := 8080
	fmt.Printf("====================================================\n")
	fmt.Printf("  MQTT Web Studio is running on http://localhost:%d\n", port)
	fmt.Printf("  Open your browser to access the MQTT Client Console\n")
	fmt.Printf("====================================================\n")

	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		log.Fatalf("HTTP server failure: %v", err)
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	session := &Session{wsConn: conn}

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read closed/error: %v", err)
			break
		}

		var cmd ClientCommand
		if err := json.Unmarshal(message, &cmd); err != nil {
			session.sendEvent("mqtt_error", map[string]string{"message": "Invalid JSON format from client"})
			continue
		}

		session.handleCommand(cmd)
	}

	// Clean up MQTT client connection on WS disconnect
	session.mu.Lock()
	if session.mqttClient != nil && session.mqttClient.IsConnected() {
		session.mqttClient.Disconnect(250)
	}
	session.mu.Unlock()
}

func (s *Session) sendEvent(event string, data interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.wsConn == nil {
		return
	}

	evt := BackendEvent{
		Event: event,
		Data:  data,
	}

	bytes, err := json.Marshal(evt)
	if err != nil {
		log.Printf("Failed to marshal event %s: %v", event, err)
		return
	}

	s.wsConn.WriteMessage(websocket.TextMessage, bytes)
}

func (s *Session) handleCommand(cmd ClientCommand) {
	switch cmd.Action {
	case "connect":
		s.handleConnect(cmd.Data)
	case "disconnect":
		s.handleDisconnect()
	case "subscribe":
		s.handleSubscribe(cmd.Data)
	case "unsubscribe":
		s.handleUnsubscribe(cmd.Data)
	case "publish":
		s.handlePublish(cmd.Data)
	default:
		s.sendEvent("mqtt_error", map[string]string{"message": fmt.Sprintf("Unknown command action: %s", cmd.Action)})
	}
}

func (s *Session) handleConnect(data map[string]interface{}) {
	protocol, _ := data["protocol"].(string)
	host, _ := data["host"].(string)
	portVal, _ := data["port"].(float64)
	clientId, _ := data["clientId"].(string)
	username, _ := data["username"].(string)
	password, _ := data["password"].(string)
	cleanSession, _ := data["cleanSession"].(bool)
	keepAliveVal, _ := data["keepAlive"].(float64)

	if protocol == "" {
		protocol = "tcp"
	}
	if host == "" {
		host = "127.0.0.1"
	}
	if portVal == 0 {
		portVal = 1883
	}
	if keepAliveVal == 0 {
		keepAliveVal = 60
	}

	brokerURI := fmt.Sprintf("%s://%s:%d", protocol, host, int(portVal))

	s.sendEvent("mqtt_connecting", map[string]string{"server": brokerURI})

	s.mu.Lock()
	if s.mqttClient != nil && s.mqttClient.IsConnected() {
		s.mqttClient.Disconnect(250)
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerURI)
	opts.SetClientID(clientId)
	if username != "" {
		opts.SetUsername(username)
	}
	if password != "" {
		opts.SetPassword(password)
	}
	opts.SetCleanSession(cleanSession)
	opts.SetKeepAlive(time.Duration(keepAliveVal) * time.Second)
	opts.SetAutoReconnect(true)

	// Global message handler for incoming MQTT messages
	opts.SetDefaultPublishHandler(func(c mqtt.Client, m mqtt.Message) {
		s.sendEvent("mqtt_message", map[string]interface{}{
			"topic":   m.Topic(),
			"payload": string(m.Payload()),
			"qos":     m.Qos(),
			"retain":  m.Retained(),
		})
	})

	opts.OnConnect = func(c mqtt.Client) {
		log.Printf("MQTT Connected to %s", brokerURI)
		s.sendEvent("mqtt_connected", map[string]string{"server": brokerURI})
	}

	opts.OnConnectionLost = func(c mqtt.Client, err error) {
		log.Printf("MQTT Connection Lost: %v", err)
		s.sendEvent("mqtt_disconnected", map[string]string{"reason": err.Error()})
	}

	client := mqtt.NewClient(opts)
	s.mqttClient = client
	s.mu.Unlock()

	go func() {
		token := client.Connect()
		if token.Wait() && token.Error() != nil {
			log.Printf("Failed to connect to MQTT broker %s: %v", brokerURI, token.Error())
			s.sendEvent("mqtt_error", map[string]string{
				"message": fmt.Sprintf("Failed to connect to %s: %v", brokerURI, token.Error()),
			})
		}
	}()
}

func (s *Session) handleDisconnect() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.mqttClient != nil && s.mqttClient.IsConnected() {
		s.mqttClient.Disconnect(250)
		s.sendEvent("mqtt_disconnected", map[string]string{"reason": "User disconnected"})
	}
}

func (s *Session) handleSubscribe(data map[string]interface{}) {
	s.mu.Lock()
	client := s.mqttClient
	s.mu.Unlock()

	if client == nil || !client.IsConnected() {
		s.sendEvent("mqtt_error", map[string]string{"message": "MQTT client not connected"})
		return
	}

	topic, _ := data["topic"].(string)
	qosVal, _ := data["qos"].(float64)

	if topic == "" {
		s.sendEvent("mqtt_error", map[string]string{"message": "Subscription topic cannot be empty"})
		return
	}

	qos := byte(qosVal)

	token := client.Subscribe(topic, qos, func(c mqtt.Client, m mqtt.Message) {
		s.sendEvent("mqtt_message", map[string]interface{}{
			"topic":   m.Topic(),
			"payload": string(m.Payload()),
			"qos":     m.Qos(),
			"retain":  m.Retained(),
		})
	})

	if token.Wait() && token.Error() != nil {
		s.sendEvent("mqtt_error", map[string]string{
			"message": fmt.Sprintf("Subscribe failed for %s: %v", topic, token.Error()),
		})
	} else {
		s.sendEvent("mqtt_subscribed", map[string]interface{}{
			"topic": topic,
			"qos":   qos,
		})
	}
}

func (s *Session) handleUnsubscribe(data map[string]interface{}) {
	s.mu.Lock()
	client := s.mqttClient
	s.mu.Unlock()

	if client == nil || !client.IsConnected() {
		s.sendEvent("mqtt_error", map[string]string{"message": "MQTT client not connected"})
		return
	}

	topic, _ := data["topic"].(string)
	if topic == "" {
		return
	}

	token := client.Unsubscribe(topic)
	if token.Wait() && token.Error() != nil {
		s.sendEvent("mqtt_error", map[string]string{
			"message": fmt.Sprintf("Unsubscribe failed for %s: %v", topic, token.Error()),
		})
	} else {
		s.sendEvent("mqtt_unsubscribed", map[string]string{
			"topic": topic,
		})
	}
}

func (s *Session) handlePublish(data map[string]interface{}) {
	s.mu.Lock()
	client := s.mqttClient
	s.mu.Unlock()

	if client == nil || !client.IsConnected() {
		s.sendEvent("mqtt_error", map[string]string{"message": "MQTT client not connected"})
		return
	}

	topic, _ := data["topic"].(string)
	qosVal, _ := data["qos"].(float64)
	retain, _ := data["retain"].(bool)
	payload, _ := data["payload"].(string)

	if topic == "" {
		s.sendEvent("mqtt_error", map[string]string{"message": "Target topic cannot be empty"})
		return
	}

	qos := byte(qosVal)

	token := client.Publish(topic, qos, retain, payload)
	if token.Wait() && token.Error() != nil {
		s.sendEvent("mqtt_error", map[string]string{
			"message": fmt.Sprintf("Publish failed to %s: %v", topic, token.Error()),
		})
	} else {
		s.sendEvent("mqtt_published", map[string]interface{}{
			"topic":   topic,
			"qos":     qos,
			"retain":  retain,
			"payload": payload,
		})
	}
}
