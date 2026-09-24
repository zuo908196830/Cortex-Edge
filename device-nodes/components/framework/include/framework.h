#pragma once

#include <stdbool.h>
#include <stddef.h>
#include "esp_err.h"
#include "cJSON.h"

#ifdef __cplusplus
extern "C" {
#endif

/**
 * @brief 外设动作回调函数原型
 * 
 * @param args 来自下发指令的 arguments (cJSON 对象或 NULL)
 */
typedef void (*action_handler_t)(cJSON *args);

/**
 * @brief 外设能力描述结构体 (开发者填写此结构体)
 */
typedef struct {
    const char *name;        /**< 能力唯一标识符，如 "led_on" */
    const char *description; /**< 能力中文描述，如 "开启房间中的灯" */
    const char *parameters;  /**< 参数描述字符串，如 "" */
    action_handler_t handler;/**< 对应的执行回调函数 */
} capability_desc_t;

/**
 * @brief 节点基础元数据配置
 */
typedef struct {
    const char *device_id;      /**< 设备唯一 ID，如 "ESP32_001" */
    const char *device_name;    /**< 设备显示名称，如 "灯" */
    const char *device_type;    /**< 设备类型，如 "light" */
    const char *description;    /**< 设备描述 */
    const char *sw_version;     /**< 固件版本号，如 "0.0.1" */
    const char *power_on_state; /**< 上电默认状态，如 "off" */
    const char *location_id;    /**< 部署位置 ID，如 "room_001" */
} node_info_t;

/**
 * @brief 初始化框架核心（创建能力注册表、初始化默认配置）
 * 
 * @param info 节点自定义元数据。若传入 NULL，则使用系统默认配置
 * @return esp_err_t ESP_OK 成功，其他错误码失败
 */
esp_err_t framework_init(const node_info_t *info);

/**
 * @brief 【核心接口】开发者调用此函数向框架注册一个外设能力
 * 
 * @param cap 能力描述结构体指针
 * @return esp_err_t ESP_OK 成功，其他错误码失败
 */
esp_err_t framework_register_capability(const capability_desc_t *cap);

/**
 * @brief 启动框架服务（自动联网 Wi-Fi、连接 MQTT、自描述上报并开启命令监听）
 * 
 * @return esp_err_t ESP_OK 成功，其他错误码失败
 */
esp_err_t framework_start(void);

// ============================================================================
// 框架内部模块协同接口（由框架内部各源文件调用）
// ============================================================================

/**
 * @brief 获取当前节点的元数据指针
 */
const node_info_t *framework_get_node_info(void);

/**
 * @brief 根据动作名称查找对应的执行回调函数
 */
action_handler_t framework_find_capability_handler(const char *name);

/**
 * @brief 自动收集所有已注册的能力，构建自描述 JSON 串
 * 
 * @return cJSON* 生成的 JSON 对象指针（调用方负责 cJSON_Delete 释放）
 */
cJSON *framework_build_device_info_json(void);

/**
 * @brief 框架消息路由入口
 */
void framework_route_message(const char *topic, const char *payload, int payload_len);

#ifdef __cplusplus
}
#endif
