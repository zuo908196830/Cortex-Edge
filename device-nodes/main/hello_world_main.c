#include <stdio.h>
#include <inttypes.h>
#include "sdkconfig.h"
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "esp_chip_info.h"
#include "esp_flash.h"
#include "esp_system.h"
#include "exti.h"
#include "led.h"
#include "wifi.h"
#include "app_mqtt.h"

/**
 * @brief 应用程序主入口函数
 * 
 * @details 顺序初始化外设（LED、按键中断）及网络通信协议栈（Wi-Fi Station 模式）。
 *          在 Wi-Fi 成功获得 IP 后拉起 MQTT 客户端并直接订阅指定的 Topics。
 */
void app_main(void)
{
    printf("\n");
    printf("==================================================\n");
    printf("   ESP32-S3 Demo: LED, EXTI, Wi-Fi & MQTT Client  \n");
    printf("==================================================\n");

    // 3. 初始化 Wi-Fi 并连接目标热点
    printf("[MAIN] Initializing Wi-Fi Station component...\n");
    esp_err_t wifi_ret = wifi_init_sta();

    // 4. Wi-Fi 连接成功后启动 MQTT 客户端并直接订阅传入的 Topic 列表
    if (wifi_ret == ESP_OK) {
        printf("[MAIN] >>> Wi-Fi Connected & IP Ready! <<<\n");
        printf("[MAIN] Starting MQTT Client...\n");

        // 需直接订阅的 Topic 数组（必须以 NULL 结尾）
        const char *my_sub_topics[] = {
            "edge/server/ESP32_001/command",
            "edge/server/ESP32_001/ack",
            NULL  // NULL 标识数组结尾
        };

        // 启动 MQTT 并直接订阅传入的 topics 列表（无需传入数组长度）
        mqtt_app_start(NULL, NULL, NULL, my_sub_topics);
    } else {
        printf("[MAIN] >>> Wi-Fi Connection Failed! MQTT Client skipped. <<<\n");
    }

    printf("[MAIN] System initialization completed successfully!\n");
    printf("[MAIN] Waiting for button presses (GPIO %d)...\n", BUTTON);

    // 1. 初始化板载 LED 引脚 (GPIO 2)
    printf("[MAIN] Initializing LED component...\n");
    led_init(GPIO_NUM_2);

    // 2. 初始化按键外部中断 (GPIO 8)
    printf("[MAIN] Initializing Button EXTI component...\n");
    button_init();

    uint32_t count = 0;

    // 5. 系统心跳主循环
    while (1) {
        vTaskDelay(5000 / portTICK_PERIOD_MS);
        count++;
        printf("[MAIN] Heartbeat tick #%" PRIu32 " (System running, free heap: %" PRIu32 " bytes)\n", 
               count, esp_get_free_heap_size());
    }
}
