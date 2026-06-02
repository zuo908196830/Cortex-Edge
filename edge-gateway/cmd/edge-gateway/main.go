package main

import (
	"edge-gateway/internal/api/mqtt"
)

func main() {
	server := mqtt.NewServer()
	server.SubscribeMqtt()
	select {}
}
