#include "framework.h"
#include "framework_config.h"
#include "mqtt_client.h"
#include "esp_log.h"
#include <string.h>
#include <stdio.h>

static const char *TAG = "NET_MQTT";
static esp_mqtt_client_handle_t s_mqtt_client = NULL;

extern esp_err_t net_wifi_init_sta(void);

static void mqtt_event_handler(void *handler_args, esp_event_base_t base, int32_t event_id, void *event_data)
{
    esp_mqtt_event_handle_t event = (esp_mqtt_event_handle_t)event_data;
    esp_mqtt_client_handle_t client = event->client;
    const node_info_t *info = framework_get_node_info();

    switch ((esp_mqtt_event_id_t)event_id) {
    case MQTT_EVENT_CONNECTED:
        ESP_LOGI(TAG, ">>> MQTT connected to broker! <<<");

        // 1. 动态订阅该节点的指令与确认主题
        char cmd_topic[128] = {0};
        char ack_topic[128] = {0};
        snprintf(cmd_topic, sizeof(cmd_topic), FRAMEWORK_TOPIC_COMMAND_PATTERN, info->device_id);
        snprintf(ack_topic, sizeof(ack_topic), FRAMEWORK_TOPIC_ACK_PATTERN, info->device_id);

        int msg_id = esp_mqtt_client_subscribe(client, cmd_topic, 1);
        ESP_LOGI(TAG, "Subscribed to: '%s' (msg_id=%d)", cmd_topic, msg_id);

        msg_id = esp_mqtt_client_subscribe(client, ack_topic, 1);
        ESP_LOGI(TAG, "Subscribed to: '%s' (msg_id=%d)", ack_topic, msg_id);

        // 2. 自动构建节点自描述 JSON 串并发布到注册主题
        cJSON *device_info = framework_build_device_info_json();
        if (device_info != NULL) {
            char *msg = cJSON_PrintUnformatted(device_info);
            if (msg != NULL) {
                char regist_topic[128] = {0};
                snprintf(regist_topic, sizeof(regist_topic), FRAMEWORK_TOPIC_REGIST_PATTERN, info->device_id);

                ESP_LOGI(TAG, "Publishing self-description to: '%s'", regist_topic);
                esp_mqtt_client_publish(client, regist_topic, msg, 0, 1, 1);
                free(msg);
            }
            cJSON_Delete(device_info);
        } else {
            ESP_LOGE(TAG, "Failed to generate device info JSON");
        }
        break;

    case MQTT_EVENT_DISCONNECTED:
        ESP_LOGW(TAG, "MQTT connection lost");
        break;

    case MQTT_EVENT_DATA:
        ESP_LOGI(TAG, "Received message on topic: %.*s", event->topic_len, event->topic);
        // 传递给框架命令路由器进行分发
        char cur_topic[128] = {0};
        if (event->topic_len < sizeof(cur_topic)) {
            memcpy(cur_topic, event->topic, event->topic_len);
            cur_topic[event->topic_len] = '\0';
            framework_route_message(cur_topic, event->data, event->data_len);
        } else {
            ESP_LOGE(TAG, "Topic length exceeds buffer limit");
        }
        break;

    case MQTT_EVENT_ERROR:
        ESP_LOGE(TAG, "MQTT event error");
        break;

    default:
        break;
    }
}

static esp_err_t net_mqtt_start(void)
{
    if (s_mqtt_client != NULL) {
        ESP_LOGW(TAG, "MQTT client already started");
        return ESP_OK;
    }

    esp_mqtt_client_config_t mqtt_cfg = {
        .broker = {
            .address.uri = FRAMEWORK_DEFAULT_MQTT_BROKER_URI,
        },
        .credentials = {
            .username = FRAMEWORK_DEFAULT_MQTT_USER,
            .authentication = {
                .password = FRAMEWORK_DEFAULT_MQTT_PASS,
            },
        },
    };

    s_mqtt_client = esp_mqtt_client_init(&mqtt_cfg);
    if (s_mqtt_client == NULL) {
        ESP_LOGE(TAG, "Failed to initialize MQTT client handle");
        return ESP_FAIL;
    }

    esp_err_t err = esp_mqtt_client_register_event(s_mqtt_client, ESP_EVENT_ANY_ID, mqtt_event_handler, NULL);
    if (err != ESP_OK) {
        ESP_LOGE(TAG, "Failed to register MQTT event handler: %d", err);
        return err;
    }

    err = esp_mqtt_client_start(s_mqtt_client);
    if (err != ESP_OK) {
        ESP_LOGE(TAG, "Failed to start MQTT client: %d", err);
        return err;
    }

    ESP_LOGI(TAG, "MQTT client started, connecting to %s ...", FRAMEWORK_DEFAULT_MQTT_BROKER_URI);
    return ESP_OK;
}

esp_err_t framework_start(void)
{
    ESP_LOGI(TAG, "Starting Framework services...");

    // 1. 启动 Wi-Fi 联网
    esp_err_t ret = net_wifi_init_sta();
    if (ret != ESP_OK) {
        ESP_LOGE(TAG, "Wi-Fi connection failed, skipping MQTT client start");
        return ret;
    }

    // 2. 启动 MQTT 客户端
    ret = net_mqtt_start();
    if (ret != ESP_OK) {
        ESP_LOGE(TAG, "MQTT client start failed: %d", ret);
        return ret;
    }

    ESP_LOGI(TAG, "Framework services started successfully!");
    return ESP_OK;
}
