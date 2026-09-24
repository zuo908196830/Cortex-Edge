#include "framework.h"
#include "esp_log.h"
#include <string.h>

static const char *TAG = "ROUTER";

static void handle_command_payload(cJSON *json)
{
    if (json == NULL) {
        ESP_LOGW(TAG, "Empty JSON payload");
        return;
    }

    cJSON *op_item = cJSON_GetObjectItem(json, "operation");
    cJSON *args_item = cJSON_GetObjectItem(json, "arguments");

    if (!cJSON_IsString(op_item) || op_item->valuestring == NULL) {
        ESP_LOGW(TAG, "Missing or invalid 'operation' in command payload");
        return;
    }

    const char *op_name = op_item->valuestring;
    ESP_LOGI(TAG, "Dispatching operation: '%s'", op_name);

    action_handler_t handler = framework_find_capability_handler(op_name);
    if (handler != NULL) {
        handler(args_item);
        ESP_LOGI(TAG, "Operation '%s' executed successfully", op_name);
    } else {
        ESP_LOGW(TAG, "No handler registered for operation '%s'", op_name);
    }
}

void framework_route_message(const char *topic, const char *payload, int payload_len)
{
    if (topic == NULL || payload == NULL || payload_len <= 0) {
        return;
    }

    // 转换为以 NULL 结尾的安全字符串
    char *buf = (char *)malloc(payload_len + 1);
    if (buf == NULL) {
        ESP_LOGE(TAG, "Failed to allocate buffer for incoming payload");
        return;
    }
    memcpy(buf, payload, payload_len);
    buf[payload_len] = '\0';

    cJSON *json = cJSON_Parse(buf);
    free(buf);

    if (json == NULL) {
        ESP_LOGE(TAG, "Failed to parse incoming payload as JSON from topic: %s", topic);
        return;
    }

    if (strstr(topic, "/command") != NULL) {
        handle_command_payload(json);
    } else if (strstr(topic, "/ack") != NULL) {
        ESP_LOGI(TAG, "Received ACK message on topic: %s", topic);
    } else if (strstr(topic, "/status") != NULL) {
        ESP_LOGI(TAG, "Received STATUS message on topic: %s", topic);
    } else {
        ESP_LOGW(TAG, "Unhandled topic: %s", topic);
    }

    cJSON_Delete(json);
}
