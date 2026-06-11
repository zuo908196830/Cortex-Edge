package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"edge-gateway/internal/model/device"
	app "edge-gateway/internal/service/gateway"
)

// TestBuildCommandsCallsLLMWithExplicitResponseFormat 验证请求会明确约束 JSON 响应格式并解析指令。
func TestBuildCommandsCallsLLMWithExplicitResponseFormat(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization = %q", got)
		}

		var request chatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request.Model != defaultModel {
			t.Fatalf("model = %q, want %q", request.Model, defaultModel)
		}
		if request.ResponseFormat.Type != "json_object" {
			t.Fatalf("response_format.type = %q, want json_object", request.ResponseFormat.Type)
		}
		if request.N != 1 {
			t.Fatalf("n = %d, want 1", request.N)
		}
		if len(request.Messages) != 2 {
			t.Fatalf("messages len = %d, want 2", len(request.Messages))
		}
		if !strings.Contains(request.Messages[0].Content, "响应格式必须完全符合") {
			t.Fatalf("system prompt does not describe response format: %s", request.Messages[0].Content)
		}
		if !strings.Contains(request.Messages[1].Content, "light-1") {
			t.Fatalf("user prompt does not include devices: %s", request.Messages[1].Content)
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewBufferString(`{"choices":[{"message":{"role":"assistant","content":"{\"commands\":[{\"device_id\":\"light-1\",\"operation\":\"set_brightness\",\"arguments\":{\"level\":30}}]}"}}]}`)),
		}, nil
	})

	planner := &Planner{
		apiKey:     "test-key",
		baseURL:    "https://example.test/chat/completions",
		httpClient: &http.Client{Transport: transport},
	}

	commands, err := planner.BuildCommands(context.Background(), app.CommandRequest{
		Utterance: "把客厅灯调到30%",
		Devices: []device.Device{
			{
				ID:   "light-1",
				Name: "客厅灯",
				Capabilities: []device.Capability{
					{
						Name: "set_brightness",
						Parameters: []device.ParameterSpec{
							{Name: "level", Type: device.ValueInt},
						},
					},
				},
			},
		},
	})
	fmt.Println(commands)
	if err != nil {
		t.Fatalf("BuildCommands returned error: %v", err)
	}
	if len(commands) != 1 {
		t.Fatalf("commands len = %d, want 1", len(commands))
	}
	if commands[0].DeviceID != "light-1" || commands[0].Operation != "set_brightness" {
		t.Fatalf("command = %+v", commands[0])
	}
	if got := commands[0].Arguments["level"].(json.Number).String(); got != "30" {
		t.Fatalf("level = %s, want 30", got)
	}
}

// TestParseCommandsRejectsExtraText 验证解析器会拒绝 JSON 对象之外的附加文本。
func TestParseCommandsRejectsExtraText(t *testing.T) {
	_, err := parseCommands(`{"commands":[]} extra`)
	if err == nil {
		t.Fatal("parseCommands returned nil error")
	}
}

// roundTripFunc 让测试可以用函数替代真实 HTTP Transport。
type roundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip 执行测试注入的 HTTP 请求处理函数。
func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
