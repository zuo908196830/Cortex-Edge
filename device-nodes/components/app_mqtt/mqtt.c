#include "app_mqtt.h"
#include "cJSON.h"
#include "router.h"
#include "esp_log.h"
#include <stdio.h>
#include <string.h>
#include "device.h"

static const char *TAG = "MQTT_APP";
static esp_mqtt_client_handle_t s_client = NULL;
static const char **s_sub_topics = NULL; // 直接保存传入的主题指针数组

/**
 * @brief MQTT 事件回调处理函数
 */
static void mqtt_event_handler(void *handler_args, esp_event_base_t base, int32_t event_id, void *event_data)
{
    esp_mqtt_event_handle_t event = event_data;
    esp_mqtt_client_handle_t client = event->client;
    int msg_id;

    switch ((esp_mqtt_event_id_t)event_id) {
    case MQTT_EVENT_CONNECTED:
        ESP_LOGI(TAG, ">>> MQTT 成功连接到服务器！ <<<");

        // 直接遍历传入的以 NULL 结尾的主题指针数组并订阅
        if (s_sub_topics != NULL) {
            for (int i = 0; s_sub_topics[i] != NULL; i++) {
                if (strlen(s_sub_topics[i]) > 0) {
                    msg_id = esp_mqtt_client_subscribe(client, s_sub_topics[i], 1);
                    ESP_LOGI(TAG, "已直接订阅传入主题 [%d]: %s, msg_id=%d", i + 1, s_sub_topics[i], msg_id);
                }
            }
        }

        cJSON *device_info = device_info_json(); // 获取设备信息 JSON 串
        if (device_info == NULL) {
            ESP_LOGE(TAG, "生成设备信息 JSON 串失败");
            break;
        }
        const char *msg = cJSON_PrintUnformatted(device_info);
        // 连接到服务器后直接发消息进行注册
        mqtt_app_publish("edge/device/ESP32_001/regist", msg, 1, 1);
        break;

    case MQTT_EVENT_DISCONNECTED:
        ESP_LOGW(TAG, ">>> MQTT 连接断开！ <<<");
        break;

    case MQTT_EVENT_SUBSCRIBED:
        ESP_LOGI(TAG, "主题订阅成功确认, msg_id=%d", event->msg_id);
        break;

    case MQTT_EVENT_UNSUBSCRIBED:
        ESP_LOGI(TAG, "取消订阅成功确认, msg_id=%d", event->msg_id);
        break;

    case MQTT_EVENT_PUBLISHED:
        ESP_LOGI(TAG, "消息发布成功确认, msg_id=%d", event->msg_id);
        break;

    case MQTT_EVENT_DATA:
        ESP_LOGI(TAG, "收到 MQTT 订阅消息:");
        ESP_LOGI(TAG, "TOPIC: %.*s", event->topic_len, event->topic);
        ESP_LOGI(TAG, "DATA : %.*s", event->data_len, event->data);

        // 安全转换 Topic 为 C 字符串后调用 components/handler/router.c 中的 route 函数
        char cur_topic[128] = {0};
        if (event->topic_len < sizeof(cur_topic)) {
            memcpy(cur_topic, event->topic, event->topic_len);
            cur_topic[event->topic_len] = '\0';
            route(cur_topic, event->data, event->data_len);
        } else {
            ESP_LOGE(TAG, "Received topic exceeds 128 bytes limit");
        }
        break;

    case MQTT_EVENT_ERROR:
        ESP_LOGE(TAG, "MQTT 发生错误事件");
        if (event->error_handle->error_type == MQTT_ERROR_TYPE_TCP_TRANSPORT) {
            ESP_LOGE(TAG, "网络底层 errno: 0x%x", event->error_handle->esp_transport_sock_errno);
        }
        break;

    default:
        ESP_LOGI(TAG, "其他 MQTT 事件 ID: %d", event->event_id);
        break;
    }
}

/**
 * @brief 初始化并启动 MQTT 客户端，直接订阅以 NULL 结尾的主题指针数组
 */
esp_err_t mqtt_app_start(const char *broker_uri, 
                         const char *username, 
                         const char *password, 
                         const char **sub_topics)
{
    if (s_client != NULL) {
        ESP_LOGW(TAG, "MQTT 客户端已启动，无需重复初始化");
        return ESP_OK;
    }

    // 保存传入的主题指针数组
    s_sub_topics = sub_topics;

    const char *uri = (broker_uri != NULL) ? broker_uri : DEFAULT_MQTT_BROKER_URI;
    const char *user = (username != NULL) ? username : DEFAULT_MQTT_USER;
    const char *pass = (password != NULL) ? password : DEFAULT_MQTT_PASS;

    esp_mqtt_client_config_t mqtt_cfg = {
        .broker = {
            .address.uri = uri,
        },
        .credentials = {
            .username = user,
            .authentication = {
                .password = pass,
            },
        },
    };

    s_client = esp_mqtt_client_init(&mqtt_cfg);
    if (s_client == NULL) {
        ESP_LOGE(TAG, "创建 MQTT 客户端句柄失败");
        return ESP_FAIL;
    }

    esp_err_t err = esp_mqtt_client_register_event(s_client, ESP_EVENT_ANY_ID, mqtt_event_handler, NULL);
    if (err != ESP_OK) {
        ESP_LOGE(TAG, "注册 MQTT 事件回调失败: %d", err);
        return err;
    }

    err = esp_mqtt_client_start(s_client);
    if (err != ESP_OK) {
        ESP_LOGE(TAG, "启动 MQTT 客户端失败: %d", err);
        return err;
    }

    ESP_LOGI(TAG, "MQTT 客户端启动成功，目标 Broker: %s", uri);
    return ESP_OK;
}

/**
 * @brief 发布一条 MQTT 消息到指定主题
 */
int mqtt_app_publish(const char *topic, const char *data, int qos, int retain)
{
    if (s_client == NULL) {
        ESP_LOGE(TAG, "MQTT 客户端未初始化");
        return -1;
    }
    return esp_mqtt_client_publish(s_client, topic, data, 0, qos, retain);
}

/**
 * @brief 动态订阅一个指定的 MQTT 主题
 */
int mqtt_app_subscribe(const char *topic, int qos)
{
    if (s_client == NULL) {
        ESP_LOGE(TAG, "MQTT 客户端未初始化");
        return -1;
    }
    return esp_mqtt_client_subscribe(s_client, topic, qos);
}
