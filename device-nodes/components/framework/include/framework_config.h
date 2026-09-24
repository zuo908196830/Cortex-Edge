#pragma once

#ifdef __cplusplus
extern "C" {
#endif

// ============================================================================
// 默认节点基本属性
// ============================================================================
#define FRAMEWORK_DEFAULT_DEVICE_ID        "ESP32_001"
#define FRAMEWORK_DEFAULT_DEVICE_NAME      "灯"
#define FRAMEWORK_DEFAULT_DEVICE_TYPE      "light"
#define FRAMEWORK_DEFAULT_DEVICE_DESC      "房间中的灯，仅支持开关"
#define FRAMEWORK_DEFAULT_SW_VERSION       "0.0.1"
#define FRAMEWORK_DEFAULT_POWER_ON_STATE   "off"
#define FRAMEWORK_DEFAULT_LOCATION_ID      "room_001"

// ============================================================================
// Wi-Fi 默认连接参数
// ============================================================================
#define FRAMEWORK_DEFAULT_WIFI_SSID        "zzzz"
#define FRAMEWORK_DEFAULT_WIFI_PASS        "13052746981"
#define FRAMEWORK_DEFAULT_WIFI_MAX_RETRY   5

// ============================================================================
// MQTT 默认连接与主题参数
// ============================================================================
#define FRAMEWORK_DEFAULT_MQTT_BROKER_URI  "mqtt://192.168.146.180:1883"
#define FRAMEWORK_DEFAULT_MQTT_USER        "esp32_admin"
#define FRAMEWORK_DEFAULT_MQTT_PASS        "12345678"

// 主题命名规则
#define FRAMEWORK_TOPIC_COMMAND_PATTERN    "edge/server/%s/command"
#define FRAMEWORK_TOPIC_ACK_PATTERN        "edge/server/%s/ack"
#define FRAMEWORK_TOPIC_REGIST_PATTERN     "edge/device/%s/regist"

#ifdef __cplusplus
}
#endif
