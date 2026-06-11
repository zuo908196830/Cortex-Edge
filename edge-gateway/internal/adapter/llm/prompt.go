package llm

import (
	"encoding/json"
	"strings"

	"edge-gateway/internal/model/device"
	app "edge-gateway/internal/service/gateway"
)

// commandSystemPrompt 构造系统提示词，明确限定模型只能返回指定 JSON 格式。
func commandSystemPrompt() string {
	return strings.Join([]string{
		"你是智能家居网关的指令规划器，只把用户文本转换成设备调用指令。",
		"必须只输出一个严格 JSON 对象，禁止输出 Markdown、代码块、解释、寒暄或任何额外文本。",
		"响应格式必须完全符合：{\"commands\":[{\"device_id\":\"设备ID\",\"operation\":\"操作名\",\"arguments\":{\"参数名\":\"参数值\"}}]}",
		"如果操作不需要参数，arguments 必须是空对象 {} 或省略。",
		"如果用户意图无法匹配任何已提供设备或能力，必须输出：{\"commands\":[]}",
		"只能使用用户消息中列出的 device_id、operation 和参数名；不要编造设备、操作或参数。",
		"参数值类型必须符合设备 capability 中的 parameters.type：string、int、bool、object、array。",
		"可以返回多条 commands，执行顺序必须与用户意图一致。",
	}, "\n")
}

// commandUserPrompt 构造用户提示词，包含用户文本和当前可用设备能力。
func commandUserPrompt(request app.CommandRequest) string {
	devicesJSON, err := json.Marshal(request.Devices)
	if err != nil {
		devicesJSON = []byte("[]")
	}

	var builder strings.Builder
	builder.WriteString("用户文本：")
	builder.WriteString(request.Utterance)
	builder.WriteString("\n\n可用设备和能力 JSON：\n")
	builder.Write(devicesJSON)
	builder.WriteString("\n\n请严格按 system 中声明的 JSON 响应格式返回。")
	return builder.String()
}

// CapabilityPrompt 返回设备操作能力的紧凑文本描述，供轻量 prompt 场景复用。
func CapabilityPrompt(devices []device.Device) string {
	var builder strings.Builder
	for _, d := range devices {
		builder.WriteString("device=")
		builder.WriteString(string(d.ID))
		builder.WriteString(" name=")
		builder.WriteString(d.Name)
		builder.WriteString(" operations=")
		for index, op := range d.Capabilities {
			if index > 0 {
				builder.WriteString(",")
			}
			builder.WriteString(op.Name)
		}
		builder.WriteByte('\n')
	}
	return builder.String()
}
