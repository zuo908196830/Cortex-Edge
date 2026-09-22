#include "include/exti.h"
#include "hal/gpio_types.h"
#include <stdio.h>
#include <stdint.h>
#include "esp_rom_sys.h"
#include "esp_timer.h"

static volatile int64_t last_press_time = 0; 

void IRAM_ATTR button_isr_handler(void *arg){
    int64_t now = esp_timer_get_time();
    if (now - last_press_time < 200000) {
        return;
    }
    last_press_time = now;
    gpio_num_t pin = (gpio_num_t)(uintptr_t)arg;
    esp_rom_printf("[ISR] Button ISR triggered on GPIO %d!\n", pin);
    if (pin == BUTTON) {
        esp_rom_printf("[ISR] Toggling LED state...\n");
        esp_rom_printf("[ISR] Current LED state: %d\n", gpio_get_level(led_gpio_num));
        TOGGLE_LED; // Toggle the LED state
    }
}

void button_init(void)
{
    printf("[EXTI] Initializing Button on GPIO %d...\n", BUTTON);
    gpio_config_t io_conf = {
        .pin_bit_mask = (1ULL << BUTTON),
        .mode = GPIO_MODE_INPUT,
        .pull_up_en = GPIO_PULLUP_DISABLE,
        .pull_down_en = GPIO_PULLDOWN_ENABLE,
        .intr_type = GPIO_INTR_POSEDGE
    };
    esp_err_t err = gpio_config(&io_conf);
    if (err != ESP_OK) {
        printf("[EXTI] Failed to configure Button GPIO: %d\n", err);
        esp_restart();
    }

    err = gpio_install_isr_service(0); // Install the ISR service with default flags
    if (err != ESP_OK && err != ESP_ERR_INVALID_STATE) { // Ignore if already installed
        printf("[EXTI] Failed to install ISR service: %d\n", err);
        esp_restart();
    }
    printf("[EXTI] ISR service ready.\n");

    err = gpio_isr_handler_add(BUTTON, button_isr_handler, (void *)(uintptr_t)BUTTON);
    if (err != ESP_OK) {
        printf("[EXTI] Failed to add ISR handler: %d\n", err);
    } else {
        printf("[EXTI] Button ISR handler successfully registered for GPIO %d.\n", BUTTON);
    }
}
