package speech

import (
	"context"
	"fmt"

	"edge-gateway/internal/model/voice"
)

type Recognizer struct{}

func NewRecognizer() *Recognizer {
	return &Recognizer{}
}

func (Recognizer) Transcribe(context.Context, voice.Audio) (voice.Transcript, error) {
	return voice.Transcript{}, fmt.Errorf("speech recognizer is not configured")
}
