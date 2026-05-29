package speech

import "context"

// Audio 是网关接收到的语音数据。
type Audio struct {
	Data       []byte `json:"-"`
	MIMEType   string `json:"mime_type"`
	SampleRate int    `json:"sample_rate,omitempty"`
	Channels   int    `json:"channels,omitempty"`
	BitDepth   int    `json:"bit_depth,omitempty"`
}

// Transcript 是语音识别后的文本结果。
type Transcript struct {
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence,omitempty"`
}

// Recognizer 负责把语音转成文本，可以接 ASR 服务或支持音频输入的 LLM。
type Recognizer interface {
	Transcribe(ctx context.Context, audio Audio) (Transcript, error)
}
