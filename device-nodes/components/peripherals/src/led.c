#include "led.h"
#include "framework.h"
#include "esp_log.h"

static const char *TAG = "PERIPH_LED";
static gpio_num_t s_led_pin = GPIO_NUM_2;

// ============================================================================
// 1. 硬件底层控制逻辑 (低电平点亮，高电平熄灭)
// ============================================================================

void led_set(bool on)
{
    // 板载 LED 通常为低电平点亮 (0 = ON, 1 = OFF)
    gpio_set_level(s_led_pin, on ? 0 : 1);
    ESP_LOGI(TAG, "LED state set to: %s (GPIO %d Level: %d)", 
             on ? "ON" : "OFF", s_led_pin, on ? 0 : 1);
}

void led_toggle(void)
{
    int cur = gpio_get_level(s_led_pin);
    gpio_set_level(s_led_pin, !cur);
    ESP_LOGI(TAG, "LED toggled to level: %d (State: %s)", 
             !cur, (!cur == 0) ? "ON" : "OFF");
}

bool led_get_state(void)
{
    return gpio_get_level(s_led_pin) == 0;
}

// ============================================================================
// 2. 开发者编写的能力动作回调函数
// ============================================================================

static void action_led_on(cJSON *args)
{
    led_set(true);
}

static void action_led_off(cJSON *args)
{
    led_set(false);
}

static void action_led_toggle(cJSON *args)
{
    led_toggle();
}

// ============================================================================
// 3. 开发者对外暴露的初始化与注册函数
// ============================================================================

void led_init(gpio_num_t pin)
{
    s_led_pin = pin;
    ESP_LOGI(TAG, "Initializing LED on GPIO %d...", s_led_pin);

    gpio_config_t io_conf = {
        .pin_bit_mask = (1ULL << s_led_pin),
        .mode = GPIO_MODE_INPUT_OUTPUT,
        .pull_up_en = GPIO_PULLUP_DISABLE,
        .pull_down_en = GPIO_PULLDOWN_ENABLE,
        .intr_type = GPIO_INTR_DISABLE,
    };
    esp_err_t err = gpio_config(&io_conf);
    if (err != ESP_OK) {
        ESP_LOGE(TAG, "Failed to configure LED GPIO %d: %d", s_led_pin, err);
        return;
    }

    // 默认关闭 LED
    led_set(false);

    // 向框架注册该外设支持的能力与回调函数
    framework_register_capability(&(capability_desc_t){
        .name = "led_on",
        .description = "开启房间中的灯",
        .parameters = "",
        .handler = action_led_on,
    });

    framework_register_capability(&(capability_desc_t){
        .name = "led_off",
        .description = "关闭房间中的灯",
        .parameters = "",
        .handler = action_led_off,
    });

    framework_register_capability(&(capability_desc_t){
        .name = "led_toggle",
        .description = "切换房间中的灯状态",
        .parameters = "",
        .handler = action_led_toggle,
    });

    ESP_LOGI(TAG, "LED peripheral initialized and capabilities registered successfully!");
}
