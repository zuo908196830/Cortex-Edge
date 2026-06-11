package llm

import (
	"os"
	"strings"
)

// chatCompletionURL 返回 LLM Chat Completions 接口地址，未配置时使用默认智谱地址。
func (p *Planner) chatCompletionURL() string {
	if p.baseURL != "" {
		return p.baseURL
	}
	return defaultChatCompletionURL
}

// chatModel 返回本次请求使用的模型，未配置时使用默认模型。
func (p *Planner) chatModel() string {
	if p.model != "" {
		return p.model
	}
	return defaultModel
}

// firstEnv 按顺序读取环境变量，返回第一个非空值。
func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}
