#include "button.h"
#include "led.h"
#include "esp_log.h"
#include "esp_timer.h"
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "freertos/queue.h"
#include <inttypes.h>

static const char *TAG = "PERIPH_BTN";
static gpio_num_t s_button_pin = GPIO_NUM_8;
static button_press_cb_t s_on_press_cb = NULL;
static QueueHandle_t s_btn_evt_queue = NULL;
static volatile int64_t s_last_press_time = 0;

static void IRAM_ATTR button_isr_handler(void *arg)
{
    int64_t now = esp_timer_get_time();
    // 200ms 防抖
    if (now - s_last_press_time < 200000) {
        return;
    }
    s_last_press_time = now;

    uint32_t pin = (uint32_t)(uintptr_t)arg;
    if (s_btn_evt_queue != NULL) {
        BaseType_t high_task_wakeup = pdFALSE;
        xQueueSendFromISR(s_btn_evt_queue, &pin, &high_task_wakeup);
        if (high_task_wakeup) {
            portYIELD_FROM_ISR();
        }
    }
}

static void button_task(void *pvParameters)
{
    uint32_t pin;
    while (1) {
        if (xQueueReceive(s_btn_evt_queue, &pin, portMAX_DELAY)) {
            ESP_LOGI(TAG, "Button pressed on GPIO %" PRIu32 ", executing callback...", pin);
            if (s_on_press_cb != NULL) {
                s_on_press_cb();
            } else {
                // 默认本地行为：翻转 LED
                led_toggle();
            }
        }
    }
}

void button_init(gpio_num_t pin, button_press_cb_t on_press)
{
    s_button_pin = pin;
    s_on_press_cb = on_press;
    ESP_LOGI(TAG, "Initializing Button on GPIO %d...", s_button_pin);

    // 1. 创建按键事件队列与处理任务
    s_btn_evt_queue = xQueueCreate(10, sizeof(uint32_t));
    xTaskCreate(button_task, "button_task", 2048, NULL, 10, NULL);

    // 2. 配置按键 GPIO
    gpio_config_t io_conf = {
        .pin_bit_mask = (1ULL << s_button_pin),
        .mode = GPIO_MODE_INPUT,
        .pull_up_en = GPIO_PULLUP_DISABLE,
        .pull_down_en = GPIO_PULLDOWN_ENABLE,
        .intr_type = GPIO_INTR_POSEDGE,
    };
    esp_err_t err = gpio_config(&io_conf);
    if (err != ESP_OK) {
        ESP_LOGE(TAG, "Failed to configure Button GPIO %d: %d", s_button_pin, err);
        return;
    }

    // 3. 安装并注册 ISR 服务
    err = gpio_install_isr_service(0);
    if (err != ESP_OK && err != ESP_ERR_INVALID_STATE) {
        ESP_LOGE(TAG, "Failed to install ISR service: %d", err);
        return;
    }

    err = gpio_isr_handler_add(s_button_pin, button_isr_handler, (void *)(uintptr_t)s_button_pin);
    if (err != ESP_OK) {
        ESP_LOGE(TAG, "Failed to add ISR handler for GPIO %d: %d", s_button_pin, err);
    } else {
        ESP_LOGI(TAG, "Button ISR handler successfully registered for GPIO %d", s_button_pin);
    }
}
