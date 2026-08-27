package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"edge-gateway/internal/model/device"
	app "edge-gateway/internal/service/gateway"
)

const (
	defaultChatCompletionURL = "https://open.bigmodel.cn/api/paas/v4/chat/completions"
	defaultModel             = "glm-4.5-air"
)

// Planner 负责调用 LLM，把用户文本和设备能力规划成设备执行指令。
type Planner struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewPlanner 基于环境变量创建默认的 LLM 指令规划器。
func NewPlanner() *Planner {
	return &Planner{
		// apiKey:  firstEnv("LLM_API_KEY", "BIGMODEL_API_KEY", "ZHIPUAI_API_KEY"),
		apiKey:  "34ae406f466d4c8c836b81124e08746f.eeRQC4jo7971izgQ",
		baseURL: "https://open.bigmodel.cn/api/paas/v4/chat/completions",
		model:   "glm-4.5-air",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// BuildCommands 调用 LLM 生成设备指令，并把模型输出解析为网关内部命令结构。
func (p *Planner) BuildCommands(ctx context.Context, request app.CommandRequest) ([]device.Command, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("llm api key is required")
	}
	if strings.TrimSpace(request.Utterance) == "" {
		return nil, fmt.Errorf("utterance is required")
	}
	if len(request.Devices) == 0 {
		return nil, fmt.Errorf("devices are required")
	}

	body, err := json.Marshal(chatCompletionRequest{
		Model: p.chatModel(),
		Messages: []chatMessage{
			{
				Role:    "system",
				Content: commandSystemPrompt(),
			},
			{
				Role:    "user",
				Content: commandUserPrompt(request),
			},
		},
		Temperature:    0.2,
		MaxTokens:      1024,
		ResponseFormat: responseFormat{Type: "json_object"},
		N:              1,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal llm request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.chatCompletionURL(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create llm request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	client := p.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call llm: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		var errorBody bytes.Buffer
		_, _ = errorBody.ReadFrom(resp.Body)
		return nil, fmt.Errorf("llm returned status %d: %s", resp.StatusCode, strings.TrimSpace(errorBody.String()))
	}

	var completion chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&completion); err != nil {
		return nil, fmt.Errorf("decode llm response: %w", err)
	}
	if len(completion.Choices) == 0 {
		return nil, fmt.Errorf("llm response has no choices")
	}

	commands, err := parseCommands(completion.Choices[0].Message.Content)
	if err != nil {
		return nil, err
	}
	return commands, nil
}
