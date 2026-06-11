package llm

import "edge-gateway/internal/model/device"

// chatCompletionRequest 是发送给 Chat Completions 接口的请求体。
type chatCompletionRequest struct {
	Model          string         `json:"model"`
	Messages       []chatMessage  `json:"messages"`
	Temperature    float64        `json:"temperature"`
	MaxTokens      int            `json:"max_tokens"`
	ResponseFormat responseFormat `json:"response_format"`
	N              int            `json:"n,omitempty"`
}

// chatMessage 表示 Chat Completions 请求或响应中的单条对话消息。
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// responseFormat 声明模型输出格式，目前要求返回 JSON 对象。
type responseFormat struct {
	Type string `json:"type"`
}

// chatCompletionResponse 是 Chat Completions 响应中当前需要解析的最小结构。
type chatCompletionResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

// commandResponse 是模型必须输出的业务 JSON 结构。
type commandResponse struct {
	Commands *[]device.Command `json:"commands"`
}
