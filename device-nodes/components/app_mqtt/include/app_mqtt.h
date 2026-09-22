#pragma once

#include "esp_err.h"
#include "mqtt_client.h"

#ifdef __cplusplus
extern "C" {
#endif

// 默认连接的 Broker URI、用户名和密码
#define DEFAULT_MQTT_BROKER_URI "mqtt://192.168.146.180:1883"
#define DEFAULT_MQTT_USER       "esp32_admin"
#define DEFAULT_MQTT_PASS       "12345678"

/**
 * @brief 初始化并启动 MQTT 客户端，并直接订阅传入的 Topic 列表 (必须以 NULL 结尾)
 * 
 * @param broker_uri MQTT Broker 地址 (例如: "mqtt://192.168.1.100:1883"，若传 NULL 则使用默认值)
 * @param username   鉴权用户名 (如果传 NULL 则使用默认值)
 * @param password   鉴权密码 (如果传 NULL 则使用默认值)
 * @param sub_topics 需直接订阅的 Topic 字符串数组，必须以 NULL 结尾 (如 {"edge/server/ESP32_001/command", "edge/server/ESP32_001/status", NULL})
 * @return esp_err_t ESP_OK 成功, 其它错误码失败
 */
esp_err_t mqtt_app_start(const char *broker_uri, 
                         const char *username, 
                         const char *password, 
                         const char **sub_topics);

/**
 * @brief 发布 MQTT 消息到任意指定 Topic
 * 
 * @param topic  目标主题
 * @param data   消息内容字符串
 * @param qos    服务质量等级 (0, 1, 2)
 * @param retain 保留标志 (0 或 1)
 * @return int 成功返回 msg_id (>=0)，失败返回 -1
 */
int mqtt_app_publish(const char *topic, const char *data, int qos, int retain);

/**
 * @brief 动态订阅一个指定的 MQTT 主题
 * 
 * @param topic 目标主题
 * @param qos   服务质量等级 (0, 1, 2)
 * @return int 成功返回 msg_id (>=0)，失败返回 -1
 */
int mqtt_app_subscribe(const char *topic, int qos);

#ifdef __cplusplus
}
#endif
