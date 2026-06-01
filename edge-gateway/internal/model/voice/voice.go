package voice

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
