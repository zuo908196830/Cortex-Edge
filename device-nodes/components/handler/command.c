#include "router.h"
#include "register.h"
#include "led.h"
#include "cJSON.h"
#include "esp_log.h"
#include <string.h>

static const char *TAG = "COMMAND";

void handle_command(cJSON *command) {
    // 处理命令消息
    if (command == NULL) {
        return;
    }

    cJSON *operation = cJSON_GetObjectItem(command, "operation");
    cJSON *args = cJSON_GetObjectItem(command, "arguments");
    if (cJSON_IsString(operation) && operation->valuestring != NULL) {
        cmd_handler_t handle = get_handler(operation->valuestring);
        if (handle) {
            handle(args);
        } else {
            ESP_LOGW(TAG, "Unknown command handler for: %s", operation->valuestring);
        }
    } else {
        ESP_LOGW(TAG, "Invalid or missing 'operation' field in command");
    }
}
