#pragma once

#include <stdint.h>
#include "cJSON.h"
#include "driver/gpio.h"
#include "register.h"

#ifdef __cplusplus
extern "C" {
#endif

extern gpio_num_t led_gpio_num;

#define TOGGLE_LED gpio_set_level(led_gpio_num, !gpio_get_level(led_gpio_num))

void led_init(gpio_num_t gpio_num);
void led_on(cJSON *args);
void led_off(cJSON *args);
void led_toggle(cJSON *args);
cJSON *led_capabilities_json();

#ifdef __cplusplus
}
#endif

