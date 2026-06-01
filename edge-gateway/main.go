package main

import (
	"context"
	"errors"

	"edge-gateway/device"
	"edge-gateway/llm"
	"edge-gateway/speech"
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

}
