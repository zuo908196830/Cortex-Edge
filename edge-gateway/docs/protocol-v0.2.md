# PrivHome 边缘语音控制协议规范 (v0.2)

> **私有化 IoT 控制框架**：终端自描述 · 本地 ASR/VAD · LLM 结构化指令 · 可靠路由

本文档描述 `edge-gateway` 模块所实现的边缘网关协议与数据模型，供设备端、网关开发与联调参考。

## 目录

- [1. 架构总览](#1-架构总览)
- [2. 设备注册协议 (Registry)](#2-设备注册协议-registry)
- [3. 语音交互链路 (Voice Pipeline)](#3-语音交互链路-voice-pipeline)
- [4. LLM 意图解析 (Intent Parsing)](#4-llm-意图解析-intent-parsing)
- [5. 指令生命周期与离线策略](#5-指令生命周期与离线策略)
- [6. Go 后端数据模型设计](#6-go-后端数据模型设计)
- [7. 项目价值定位](#7-项目价值定位)
- [8. 落地里程碑 (v0.1 MVP)](#8-落地里程碑-v01-mvp)

---

## 1. 架构总览

```
                    ┌─────────────┐
                    │  终端 MCU   │
                    └──────┬──────┘
           MQTT/WSS        │ 状态 / REGISTER
                           ▼
                    ┌─────────────┐
                    │ 边缘网关     │  树莓派 (edge-gateway)
                    └──────┬──────┘
                           │
         ┌─────────────────┼─────────────────┐
         ▼                 ▼                 ▼
   ┌──────────┐     ┌──────────┐     ┌──────────┐
   │ 本地 ASR │ ──► │ 本地 LLM │ ──► │ 指令下发 │
   └──────────┘     └──────────┘     └──────────┘
         ▲
         │ 唤醒词 / KWS / 录音
   ┌──────────┐
   │  麦克风  │
   └──────────┘
```

| 组件 | 职责 |
|------|------|
| 终端 MCU | 设备自描述、状态上报、执行指令 |
| 边缘网关 (`edge-gateway`) | 注册、路由、离线队列、语音编排 |
| 本地 ASR | 语音转文本 |
| 本地 LLM | 意图解析为结构化 JSON |
| 麦克风 + KWS | 唤醒与采集 |

**相关实现文档**：[网关结构说明](./architecture.md) · [C 设备 HTTP 接入](./c-device-protocol.md)

---

## 2. 设备注册协议 (Registry)

### 2.1 上行 (Device → Gateway)

**Topic / 通道**：由实现定义（MQTT 或 WSS）；HTTP 注册见 [C 设备接入协议](./c-device-protocol.md)。

**消息格式（MQTT 载荷示例）**：

```json
{
  "msg_type": "REGISTER",
  "payload": {
    "device_id": "living_room_light_01",
    "device_type": "light",
    "sw_version": "0.2.1",
    "power_on_state": "LAST",
    "current_state": {
      "power": "on",
      "brightness": 50
    },
    "capabilities": [
      { "name": "turn_on" },
      { "name": "turn_off" },
      {
        "name": "set_brightness",
        "params": {
          "level": { "type": "int", "min": 1, "max": 100 }
        }
      }
    ]
  }
}
```

| 字段 | 说明 |
|------|------|
| `device_id` | 全局唯一设备标识 |
| `device_type` | 设备类型，如 `light`、`curtain` |
| `power_on_state` | 上电默认行为：`LAST` / `ON` / `OFF` |
| `current_state` | **当前真实状态**（网关影子同步依据） |
| `capabilities` | 能力集，含动作名与参数 schema |

### 2.2 网关处理策略

| 策略 | 行为 |
|------|------|
| **Upsert** | `INSERT OR REPLACE` 全量覆盖 SQLite |
| **状态同步** | 用 `current_state` 更新 Device Shadow |
| **无版本迁移** | 每次注册即最新真相，无需增量迁移 |

---

## 3. 语音交互链路 (Voice Pipeline)

### 3.1 唤醒词 (Wake Word)

| 项 | 说明 |
|----|------|
| **目的** | 避免全时录音与误触发 |
| **方案** | 本地轻量 KWS（如 Porcupine） |
| **状态机** | `IDLE` → `WAKE` → `LISTENING` → `PROCESSING` |

### 3.2 VAD 与录音结束判定

**判决逻辑**：连续静音超时，而非「声音一停就结束」。

| 参数 | 值 | 说明 |
|------|-----|------|
| 起始连续帧 | 3–5 帧 @ 20ms | 防噪音误触 |
| **静音超时** | **300–500 ms** | **判定「一句话结束」** |
| 最大时长 | 8–15 s | 强制切出 |

### 3.3 ASR

| 项 | 说明 |
|----|------|
| 引擎 | 本地 Whisper（`tiny` / `base`） |
| 输入 | 16 kHz Mono WAV |
| 输出 | 纯文本，例如：`打开客厅灯` |

**HTTP 入口**：`POST /voice`（见 [architecture.md](./architecture.md)）。

---

## 4. LLM 意图解析 (Intent Parsing)

### 4.1 输出约束 (JSON Schema)

LLM **必须**输出严格 JSON，禁止自然语言发散。

```json
{
  "device_id": "living_room_light_01",
  "action": "set_brightness",
  "params": { "level": 30 }
}
```

### 4.2 上下文策略

- 去除历史对话，单次意图独立解析
- 注入固定规则（例如：`只说开灯` → 映射为 `living_room_light_01` + `turn_on`）

---

## 5. 指令生命周期与离线策略

### 5.1 心跳与离线

| 事件 | 网关行为 |
|------|----------|
| 心跳正常 | 设备标记 `ONLINE`，指令即时下发 |
| 心跳超时 | 标记 `OFFLINE`，新指令进入 **Pending Queue** |

### 5.2 Pending 指令合并策略

| 指令类型 | 策略 | 原因 |
|----------|------|------|
| 写值型（亮度 / 色温） | **Last-Write-Wins**（保留最后一条） | 最终以最新目标值为准 |
| 翻转型（On / Off） | **丢弃** | 防止离线期间「鬼火」 |
| 场景型 | **丢弃** | 上下文已失效 |
| OTA / 配置 | **保留** | 需完整执行 |

### 5.3 设备上线恢复

1. 收到 `REGISTER` → 标记 `ONLINE`
2. 扫描 Pending Queue
3. 过滤 TTL（默认 **5 min**）
4. 下发剩余有效指令

---

## 6. Go 后端数据模型设计

> 目标包：`edge-gateway/device`、`edge-gateway/registry`（规划中的 SQLite 持久化）。

### 6.1 数据库表 (SQLite)

```sql
CREATE TABLE devices (
  id                  INTEGER PRIMARY KEY AUTOINCREMENT,
  device_id           TEXT UNIQUE NOT NULL,
  device_type         TEXT NOT NULL,  -- "light", "curtain", "ac"
  sw_version          TEXT,
  power_on_state      TEXT,           -- "LAST", "ON", "OFF"
  capabilities_json   TEXT,           -- JSON string
  current_state_json  TEXT,           -- JSON string（关键）
  last_seen_ts        INTEGER
);
```

### 6.2 Go 结构体定义

```go
// DeviceState 标记接口
type DeviceState interface {
	IsDeviceState()
}

// LightState 灯的状态
type LightState struct {
	Power      string `json:"power"`
	Brightness int    `json:"brightness"`
}

func (LightState) IsDeviceState() {}

// CurtainState 窗帘的状态
type CurtainState struct {
	Position int `json:"position"`
}

func (CurtainState) IsDeviceState() {}

// Device 设备主模型
type Device struct {
	DeviceID         string
	DeviceType       string
	CapabilitiesRaw  string
	CurrentStateRaw  string
	CurrentState     DeviceState // 运行时解析后的状态
}
```

### 6.3 状态解析工厂

```go
func ParseState(deviceType, jsonStr string) (DeviceState, error) {
	switch deviceType {
	case "light":
		var s LightState
		if err := json.Unmarshal([]byte(jsonStr), &s); err != nil {
			return nil, err
		}
		return s, nil
	case "curtain":
		var s CurtainState
		if err := json.Unmarshal([]byte(jsonStr), &s); err != nil {
			return nil, err
		}
		return s, nil
	default:
		return nil, errors.New("unknown device type")
	}
}
```

---

## 7. 项目价值定位

> 设计并实现私有化 IoT 控制协议与边缘语音交互系统。采用设备自注册与全量状态上报消除网关冷启动盲区；利用本地唤醒词 + VAD 构建 Session 状态机，降低算力与隐私风险；针对设备离线场景设计基于 Last-Write-Wins 的指令合并策略，保证最终一致性。

---

## 8. 落地里程碑 (v0.1 MVP)

| # | 范围 | 交付物 |
|---|------|--------|
| 1 | **硬件** | 树莓派 + USB 麦 + ESP32 继电器 |
| 2 | **链路** | 唤醒词 → VAD 切句 → ASR → LLM → JSON → GPIO 控制 |
| 3 | **存储** | SQLite 存 Registry 与 Pending |
| 4 | **文档** | 协议定稿，代码开源 |
