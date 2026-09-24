#include <stdio.h>
#include <inttypes.h>
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "esp_system.h"
#include "framework.h"
#include "led.h"
#include "button.h"

#define LED_PIN    GPIO_NUM_2
#define BUTTON_PIN GPIO_NUM_8

void app_main(void)
{
    printf("\n");
    printf("=====================================================\n");
    printf("   Cortex-Edge Node: Framework & Peripherals Demo    \n");
    printf("=====================================================\n");

    // 1. 初始化框架核心（传入 NULL 使用默认设备配置，也可传入自定义 node_info_t）
    printf("[MAIN] Initializing Framework core...\n");
    framework_init(NULL);

    // 2. 开发者初始化外设（外设内部会自动向框架注册其能力）
    printf("[MAIN] Initializing LED peripheral (GPIO %d)...\n", LED_PIN);
    led_init(LED_PIN);

    printf("[MAIN] Initializing Button peripheral (GPIO %d)...\n", BUTTON_PIN);
    button_init(BUTTON_PIN, NULL); // 传入 NULL 时默认翻转 LED

    // 3. 启动框架服务（自动连 Wi-Fi、连 MQTT、自描述上报并开启命令监听）
    printf("[MAIN] Starting Framework network & MQTT services...\n");
    framework_start();

    printf("[MAIN] System initialization completed successfully!\n");

    // 4. 心跳主循环
    uint32_t count = 0;
    while (1) {
        vTaskDelay(5000 / portTICK_PERIOD_MS);
        count++;
        printf("[MAIN] Heartbeat #%" PRIu32 " (Running, Free Heap: %" PRIu32 " bytes)\n",
               count, esp_get_free_heap_size());
    }
}
