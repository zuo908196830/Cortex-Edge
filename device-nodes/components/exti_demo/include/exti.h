#pragma once

#include "driver/gpio.h"
#include "led.h"
#include "esp_system.h"

#ifdef __cplusplus
extern "C" {
#endif

#define BUTTON GPIO_NUM_8

void button_init(void);

#ifdef __cplusplus
}
#endif

