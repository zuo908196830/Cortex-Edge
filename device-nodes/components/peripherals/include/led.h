#pragma once

#include "driver/gpio.h"
#include <stdbool.h>

#ifdef __cplusplus
extern "C" {
#endif

/**
 * @brief 初始化 LED 外设并向框架注册能力 (led_on, led_off, led_toggle)
 * 
 * @param pin LED 连接的 GPIO 引脚编号 (如 GPIO_NUM_2)
 */
void led_init(gpio_num_t pin);

/**
 * @brief 设置 LED 状态
 * 
 * @param on true 打开, false 关闭
 */
void led_set(bool on);

/**
 * @brief 翻转 LED 状态
 */
void led_toggle(void);

/**
 * @brief 获取 LED 当前电平状态
 */
bool led_get_state(void);

#ifdef __cplusplus
}
#endif
