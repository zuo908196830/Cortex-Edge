package llm

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"edge-gateway/internal/model/device"
)

// parseCommands 将 LLM message.content 中的 JSON 对象解析为设备指令列表。
func parseCommands(content string) ([]device.Command, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("llm response content is empty")
	}

	var response commandResponse
	decoder := json.NewDecoder(strings.NewReader(content))
	decoder.UseNumber()
	if err := decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("decode llm commands: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("decode llm commands: response must contain only one JSON object")
	}
	if response.Commands == nil {
		return nil, fmt.Errorf("decode llm commands: missing commands field")
	}
	return *response.Commands, nil
}
