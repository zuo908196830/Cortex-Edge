#pragma once

#include "esp_err.h"
#include <string.h>
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "freertos/event_groups.h"
#include "esp_system.h"
#include "esp_wifi.h"
#include "esp_event.h"
#include "esp_log.h"
#include "nvs_flash.h"
#include "esp_netif.h"

#ifdef __cplusplus
extern "C" {
#endif

// ==========================================
// 请在此处填写你的真实 Wi-Fi 名称和密码
// ==========================================
#define WIFI_SSID       "zzzz"
#define WIFI_PASSWORD   "13052746981"

/**
 * @brief 初始化 NVS、网络栈，并以 Station (STA) 模式连接指定 Wi-Fi
 * 
 * @return esp_err_t ESP_OK: 连接成功并获取到 IP; ESP_FAIL: 连接失败/超限
 */
esp_err_t wifi_init_sta(void);

#ifdef __cplusplus
}
#endif

