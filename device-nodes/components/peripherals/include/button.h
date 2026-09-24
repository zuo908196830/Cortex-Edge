#pragma once

#include "driver/gpio.h"

#ifdef __cplusplus
extern "C" {
#endif

/**
 * @brief 按键按下回调函数原型
 */
typedef void (*button_press_cb_t)(void);

/**
 * @brief 初始化按键外部中断
 * 
 * @param pin 按键连接的 GPIO 引脚编号 (如 GPIO_NUM_8)
 * @param on_press 按键触发时的回调函数（传 NULL 则使用默认的 led_toggle）
 */
void button_init(gpio_num_t pin, button_press_cb_t on_press);

#ifdef __cplusplus
}
#endif
