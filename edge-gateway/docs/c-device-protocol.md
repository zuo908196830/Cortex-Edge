# C 设备接入协议

网关优先使用 `http_json` 协议调用设备，设备端即使用 C 编写，也只需要实现一个简单 HTTP JSON 接口。

## 设备注册

设备启动后向网关注册自身能力：

```http
POST /devices
Content-Type: application/json
```

```json
{
  "id": "light-1",
  "name": "living-room-light",
  "protocol": "http_json",
  "endpoint": "http://192.168.1.20:9001",
  "description": "living room light",
  "operations": [
    {
      "name": "turn_on",
      "description": "turn on light"
    },
    {
      "name": "set_brightness",
      "description": "set brightness",
      "parameters": [
        {
          "name": "level",
          "type": "integer",
          "required": true,
          "description": "0-100"
        }
      ]
    }
  ]
}
```

说明：

- `protocol` 可以不传，网关默认使用 `http_json`。
- 字段名全部使用 snake_case，方便 C 端用 cJSON、jsmn 等库解析。
- 参数类型建议优先使用 `string`、`integer`、`number`、`bool`，减少 C 端动态类型处理成本。
- 暂时不建议设备端暴露复杂嵌套对象；如果必须使用，可声明 `object` 或 `array`。

## 网关调用设备

网关会调用设备注册的 endpoint，并拼接操作路径：

```http
POST {endpoint}/operations/{operation}
Content-Type: application/json
Accept: application/json, text/plain
X-Device-ID: light-1
X-Operation: set_brightness
```

请求体：

```json
{
  "device_id": "light-1",
  "operation": "set_brightness",
  "arguments": {
    "level": 80
  }
}
```

无参数操作的请求体：

```json
{
  "device_id": "light-1",
  "operation": "turn_on"
}
```

## 设备响应

推荐返回 JSON：

```json
{
  "output": {
    "success": true
  }
}
```

失败时：

```json
{
  "error": "brightness level out of range"
}
```

为了兼容简单 C 设备，网关也接受这些响应：

- `204 No Content`：表示执行成功，没有输出。
- `200 OK` 且空响应体：表示执行成功，没有输出。
- `200 OK` 且 `Content-Type: text/plain`：响应体会作为字符串输出。

非 2xx 状态码会被网关视为执行失败。

## C 端最小处理逻辑

设备端只需要实现路由：

```text
POST /operations/turn_on
POST /operations/set_brightness
```

处理步骤：

1. 读取 HTTP 请求体。
2. 解析 JSON 中的 `operation` 和 `arguments`。
3. 根据 operation 调用本地函数。
4. 返回 `{"output": ...}` 或 `{"error": "..."}`。

网关已经会在调用前校验：

- 设备是否已注册。
- 操作名是否存在。
- 必填参数是否存在。
- 参数类型是否匹配。
