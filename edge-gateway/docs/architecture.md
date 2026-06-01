# 网关结构说明

当前网关按职责拆成几个很薄的结构，避免把语音识别、LLM 决策和设备调用混在一起。

> 协议与离线策略详见 [protocol-v0.2.md](./protocol-v0.2.md)。

## 主流程

```text
POST /voice
  -> api.Server 解析 HTTP 请求
  -> gateway.HandleVoice 接收音频
  -> speech.Recognizer.Transcribe 把语音转文本
  -> gateway.HandleText 处理文本意图
  -> llm.CommandPlanner.BuildCommands 生成设备指令
  -> device.Device.ValidateCommand 校验指令
  -> transport.Invoker.Invoke 调用 C 设备
```

## speech.Audio

`speech.Audio` 表示语音输入：

```go
type Audio struct {
    Data       []byte
    MIMEType   string
    SampleRate int
    Channels   int
    BitDepth   int
}
```

作用：保存音频二进制和音频格式。C 端如果上传裸 PCM，后续可以补充采样率、声道数和位深。

## speech.Recognizer

`speech.Recognizer` 只负责语音转文本：

```go
type Recognizer interface {
    Transcribe(ctx context.Context, audio Audio) (Transcript, error)
}
```

这个接口可以接：

- 云端 ASR。
- 本地语音识别模型。
- 支持音频输入的 LLM。

## llm.CommandRequest

`llm.CommandRequest` 是发给 LLM 的指令生成请求：

```go
type CommandRequest struct {
    Utterance        string
    Devices          []device.Device
}
```

它只包含文本意图和设备能力，不再直接包含音频。这样 LLM 决策层更清楚，也更容易测试。

## llm.CommandPlanner

`llm.CommandPlanner` 负责生成设备调用指令：

```go
type CommandPlanner interface {
    BuildCommands(ctx context.Context, request CommandRequest) ([]device.Command, error)
}
```

真实调用 LLM 的代码应该放在这个接口的实现里。它不直接操作设备，只返回命令列表。

## device.Device

`device.Device` 是设备注册时上报的能力描述：

```go
type Device struct {
    ID          ID
    Name        string
    Protocol    Protocol
    Endpoint    string
    Description string
    Operations  []OperationSpec
}
```

它告诉网关：设备是谁、在哪、支持哪些操作、每个操作需要哪些参数。


## Device 统一实体

现在设备能力统一收敛到 `device.Device`，不同模块不再各自维护一套设备结构，而是通过 `Device` 上的方法面向不同场景：

```go
dev.Normalize()          // 注册前补默认值
dev.ValidateForRegister() // 注册时校验设备能力
dev.ValidateCommand(cmd)  // 执行前校验 LLM 指令
dev.NewInvokeRequest(cmd) // 调用 C 设备前生成请求体
```

`Command`、`Result`、`InvokeRequest` 仍然保留为独立结构，因为它们不是设备本体，而是网关流转时的消息。这样既有统一设备实体，也不会让 `Device` 变成承载所有字段的大结构。

## device.Command

`device.Command` 是 LLM 生成、网关执行的标准指令：

```go
type Command struct {
    DeviceID  ID
    Operation string
    Arguments map[string]any
}
```

网关会先校验这个指令，确认设备和操作都存在，参数也符合设备注册的操作集。

## transport.Invoker

`transport.Invoker` 负责把指令发给设备：

```go
type Invoker interface {
    Invoke(ctx context.Context, dev device.Device, command device.Command) (device.Result, error)
}
```

现在提供的是 HTTP JSON 实现，适合 C 设备端实现简单的 HTTP 接口。

## 为什么不用原来的 Plan

原来的 `Plan` 同时接收音频、设备能力，并返回设备指令，职责偏大：

```text
音频处理 + 语音识别 + LLM 决策
```

现在拆成：

```text
speech.Recognizer.Transcribe：语音 -> 文本
llm.CommandPlanner.BuildCommands：文本 + 设备能力 -> 设备指令
```

这样后续替换 ASR、替换 LLM、加文本调试入口都会更简单。
