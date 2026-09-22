#include "wifi.h"

static const char *TAG = "wifi_sta";

// FreeRTOS 事件组标志位
#define WIFI_CONNECTED_BIT BIT0  // 成功连上 AP 并获取到 IP
#define WIFI_FAIL_BIT      BIT1  // 达到最大重试次数，连接失败

#define MAXIMUM_RETRY      5     // 最大断线重试次数

// s_wifi_event_group 是一个全局事件组句柄，由 xEventGroupCreate() 在堆上动态分配内存创建。
// 它作为媒介，用于在异步的“Wi-Fi 事件回调线程”与同步的“主初始化线程”之间传递状态与同步唤醒。
static EventGroupHandle_t s_wifi_event_group;
static int s_retry_num = 0;

/**
 * @brief Wi-Fi 与 IP 底层事件回调函数
 * 
 * @details 响应底层数据链路层 (WIFI_EVENT) 和网络层 (IP_EVENT) 的事件。
 *          处理 Wi-Fi 启动连接、断开重连逻辑，并在成功获取 IP 地址后唤醒阻塞的主线程。
 * 
 * @param arg         用户在注册事件时传入的上下文参数（此例传为 NULL）
 * @param event_base  事件基类标识符（如 WIFI_EVENT 或 IP_EVENT）
 * @param event_id    具体的事件 ID（如 WIFI_EVENT_STA_START, IP_EVENT_STA_GOT_IP 等）
 * @param event_data  随事件附带的数据结构指针（如 ip_event_got_ip_t）
 */
static void wifi_event_handler(void* arg, esp_event_base_t event_base,
                               int32_t event_id, void* event_data)
{
    if (event_base == WIFI_EVENT && event_id == WIFI_EVENT_STA_START) {
        ESP_LOGI(TAG, "Wi-Fi Station 启动成功，正在连接: %s ...", WIFI_SSID);
        esp_wifi_connect();
    } else if (event_base == WIFI_EVENT && event_id == WIFI_EVENT_STA_DISCONNECTED) {
        if (s_retry_num < MAXIMUM_RETRY) {
            esp_wifi_connect();
            s_retry_num++;
            ESP_LOGW(TAG, "Wi-Fi 连接断开，正在尝试第 %d/%d 次重连...", s_retry_num, MAXIMUM_RETRY);
        } else {
            ESP_LOGE(TAG, "连接失败，已达到最大重试次数 (%d 次)！", MAXIMUM_RETRY);
            // xEventGroupSetBits 是非阻塞调用。它会将事件组中指定的位（此处为 WIFI_FAIL_BIT）设为 1。
            // 此时，正在 xEventGroupWaitBits 处挂起等待此事件的主线程会立刻被解除阻塞、重新进入就绪态。
            xEventGroupSetBits(s_wifi_event_group, WIFI_FAIL_BIT);
        }
    } else if (event_base == IP_EVENT && event_id == IP_EVENT_STA_GOT_IP) {
        ip_event_got_ip_t* event = (ip_event_got_ip_t*) event_data;
        ESP_LOGI(TAG, ">>> 联网成功！获取到 IP 地址: " IPSTR " <<<", IP2STR(&event->ip_info.ip));
        s_retry_num = 0; // 重置重试计数
        // 获取 IP 成功后，非阻塞地向事件组写入 WIFI_CONNECTED_BIT 信号，唤醒主线程。
        xEventGroupSetBits(s_wifi_event_group, WIFI_CONNECTED_BIT);
    }
}

/**
 * @brief 初始化 NVS、网络栈，并以 Station (STA) 模式连接指定 Wi-Fi
 * 
 * @details 执行步骤：
 *          1. 初始化 NVS 闪存；
 *          2. 创建 FreeRTOS 事件组；
 *          3. 初始化 LwIP 网络协议栈及默认事件循环；
 *          4. 配置并启动 Wi-Fi 驱动；
 *          5. 阻塞挂起等待，直到成功获得 IP 或超过最大重试次数。
 * 
 * @return esp_err_t ESP_OK: 成功连接 AP 并获取到 IP 地址；
 *                   ESP_FAIL: 达到最大重试次数依然无法连接。
 */
esp_err_t wifi_init_sta(void)
{
    s_retry_num = 0; // 显式复位重试计数器

    // 1. 初始化 NVS 闪存（底层 Wi-Fi 射频校准必须使用）
    esp_err_t ret = nvs_flash_init();
    if (ret == ESP_ERR_NVS_NO_FREE_PAGES || ret == ESP_ERR_NVS_NEW_VERSION_FOUND) {
        ESP_ERROR_CHECK(nvs_flash_erase());
        ret = nvs_flash_init();
    }
    ESP_ERROR_CHECK(ret);

    // 创建 FreeRTOS 事件组用于同步等待连接结果
    s_wifi_event_group = xEventGroupCreate();

    // 2. 初始化底层网络协议栈（LWIP）与系统默认事件循环
    ESP_ERROR_CHECK(esp_netif_init());
    
    // 兼容可能已创建过默认事件循环的情况
    esp_err_t event_err = esp_event_loop_create_default();
    if (event_err != ESP_OK && event_err != ESP_ERR_INVALID_STATE) {
        ESP_ERROR_CHECK(event_err);
    }

    esp_netif_create_default_wifi_sta();

    // 3. 初始化 Wi-Fi 驱动底层资源
    wifi_init_config_t cfg = WIFI_INIT_CONFIG_DEFAULT();
    ESP_ERROR_CHECK(esp_wifi_init(&cfg));

    // 4. 注册 Wi-Fi 事件与 IP 事件监听器
    esp_event_handler_instance_t instance_any_id;
    esp_event_handler_instance_t instance_got_ip;
    ESP_ERROR_CHECK(esp_event_handler_instance_register(WIFI_EVENT,
                                                        ESP_EVENT_ANY_ID,
                                                        &wifi_event_handler,
                                                        NULL,
                                                        &instance_any_id));
    ESP_ERROR_CHECK(esp_event_handler_instance_register(IP_EVENT,
                                                        IP_EVENT_STA_GOT_IP,
                                                        &wifi_event_handler,
                                                        NULL,
                                                        &instance_got_ip));

    // 5. 设置 Station 模式与目标 Wi-Fi 账号密码
    wifi_config_t wifi_config = {
        .sta = {
            .ssid = WIFI_SSID,
            .password = WIFI_PASSWORD,
            .threshold.authmode = strlen(WIFI_PASSWORD) == 0 ? WIFI_AUTH_OPEN : WIFI_AUTH_WPA2_PSK,
        },
    };
    ESP_ERROR_CHECK(esp_wifi_set_mode(WIFI_MODE_STA));
    ESP_ERROR_CHECK(esp_wifi_set_config(WIFI_IF_STA, &wifi_config));

    // 6. 启动 Wi-Fi 硬件射频
    ESP_LOGI(TAG, "正在启动 Wi-Fi Station 硬件...");
    ESP_ERROR_CHECK(esp_wifi_start());

    // 7. 阻塞等待连接结果（成功获得 IP 或重试超时失败）
    EventBits_t bits = xEventGroupWaitBits(s_wifi_event_group,
            WIFI_CONNECTED_BIT | WIFI_FAIL_BIT,
            pdFALSE,
            pdFALSE,
            portMAX_DELAY);

    if (bits & WIFI_CONNECTED_BIT) {
        ESP_LOGI(TAG, "成功连入目标 Wi-Fi: %s", WIFI_SSID);
        return ESP_OK;
    } else if (bits & WIFI_FAIL_BIT) {
        ESP_LOGE(TAG, "无法连入目标 Wi-Fi: %s", WIFI_SSID);
        // 注销事件监听并释放事件组资源
        esp_event_handler_instance_unregister(WIFI_EVENT, ESP_EVENT_ANY_ID, instance_any_id);
        esp_event_handler_instance_unregister(IP_EVENT, IP_EVENT_STA_GOT_IP, instance_got_ip);
        vEventGroupDelete(s_wifi_event_group);
        return ESP_FAIL;
    } else {
        ESP_LOGE(TAG, "未知异常事件");
        return ESP_ERR_INVALID_STATE;
    }
}
