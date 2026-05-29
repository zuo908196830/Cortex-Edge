package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"

	"edge-gateway/api"
	"edge-gateway/device"
	"edge-gateway/gateway"
	"edge-gateway/llm"
	"edge-gateway/registry"
	"edge-gateway/speech"
	"edge-gateway/transport"
)

type speechStub struct{}

func (speechStub) Transcribe(context.Context, speech.Audio) (speech.Transcript, error) {
	return speech.Transcript{}, errors.New("speech recognizer is not configured")
}

type commandPlannerStub struct{}

func (commandPlannerStub) BuildCommands(context.Context, llm.CommandRequest) ([]device.Command, error) {
	return nil, errors.New("llm command planner is not configured")
}

func main() {
	addr := os.Getenv("GATEWAY_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	gw := gateway.New(
		registry.NewMemoryRegistry(),
		speechStub{},
		commandPlannerStub{},
		transport.NewHTTPInvoker(nil),
	)

	server := &http.Server{
		Addr:    addr,
		Handler: api.NewServer(gw).Handler(),
	}

	log.Printf("edge gateway listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
