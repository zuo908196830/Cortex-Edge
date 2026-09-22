#include "led.h"
#include "driver/gpio.h"
#include "esp_system.h"
#include <stdio.h>
#include "cJSON.h"

gpio_num_t led_gpio_num = GPIO_NUM_2;

void led_init(gpio_num_t gpio_num) {
    printf("[LED] Initializing LED on GPIO %d...\n", gpio_num);
    led_gpio_num = gpio_num; // Store the GPIO number for later use
    gpio_config_t *io_conf = &(gpio_config_t){
        .pin_bit_mask = (1ULL << gpio_num),
        .mode = GPIO_MODE_INPUT_OUTPUT,
        .pull_up_en = GPIO_PULLUP_DISABLE,
        .pull_down_en = GPIO_PULLDOWN_ENABLE,
        .intr_type = GPIO_INTR_DISABLE
    };
    esp_err_t err = gpio_config(io_conf);
    if (err != ESP_OK) {
        printf("[LED] Failed to configure GPIO %d: %d\n", gpio_num, err);
        esp_restart();
    }
    printf("[LED] LED on GPIO %d initialized successfully.\n", gpio_num);

    regist("led_on", led_on);
    regist("led_off", led_off);
    regist("led_toggle", led_toggle);
}

void led_on(cJSON *args) {
    printf("[LED] Turning LED ON (GPIO %d LOW)\n", led_gpio_num);
    gpio_set_level(led_gpio_num, 0); // Using the configured GPIO number
}

void led_off(cJSON *args) {
    printf("[LED] Turning LED OFF (GPIO %d HIGH)\n", led_gpio_num);
    gpio_set_level(led_gpio_num, 1); // Using the configured GPIO number
}

void led_toggle(cJSON *args) {
    int current_level = gpio_get_level(led_gpio_num);
    gpio_set_level(led_gpio_num, !current_level);
    printf("[LED] Toggled LED on GPIO %d to %d\n", led_gpio_num, !current_level);
}

// 能力列表
cJSON *led_capabilities_json() {
    cJSON *capabilities = cJSON_CreateArray();
    if (capabilities == NULL) {
        return NULL;
    }

    // 添加能力对象
    cJSON *capability_on = cJSON_CreateObject();
    if (capability_on == NULL) {
        cJSON_Delete(capabilities);
        return NULL;
    }
    cJSON_AddStringToObject(capability_on, "name", "led_on");
    cJSON_AddStringToObject(capability_on, "description", "开启房间中的灯");
    cJSON_AddStringToObject(capability_on, "parameters", "");

    // 添加能力对象
    cJSON *capability_off = cJSON_CreateObject();
    if (capability_off == NULL) {
        cJSON_Delete(capabilities);
        return NULL;
    }
    cJSON_AddStringToObject(capability_off, "name", "led_off");
    cJSON_AddStringToObject(capability_off, "description", "关闭房间中的灯");
    cJSON_AddStringToObject(capability_off, "parameters", "");

    // 添加能力对象
    cJSON *capability_toggle = cJSON_CreateObject();
    if (capability_toggle == NULL) {
        cJSON_Delete(capabilities);
        return NULL;
    }
    cJSON_AddStringToObject(capability_toggle, "name", "led_toggle");
    cJSON_AddStringToObject(capability_toggle, "description", "切换房间中的灯状态");
    cJSON_AddStringToObject(capability_toggle, "parameters", "");

    // 将能力对象添加到数组中
    cJSON_AddItemToArray(capabilities, capability_on);
    cJSON_AddItemToArray(capabilities, capability_off);
    cJSON_AddItemToArray(capabilities, capability_toggle);

    return capabilities;
}
