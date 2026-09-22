#include "router.h"
#include "register.h"
#include "esp_log.h"
#include <string.h>
#include "cJSON.h"

static const char *TAG = "ROUTER";

/**
 * @brief 根据 Topic 进行路由的分支选择
 */
void route(const char *topic, const char *msg, const int msg_len)
{
    cJSON *json = cJSON_Parse(msg);
    if (json == NULL) {
        ESP_LOGE("JSON", "JSON 语法解析失败！格式不正确");
        return;
    }
    if (topic == NULL) {
        return;
    }

    // 根据 Topic 后缀/关键字进行路由分支选择
    if (strstr(topic, "/command") != NULL) {
        // [分支 1]: 处理以 /command 结尾的主题
        handle_command(json);
    } else if (strstr(topic, "/status") != NULL) {
        // [分支 2]: 处理以 /status 结尾的主题

    } else if (strstr(topic, "/ack") != NULL) {
        // [分支 3]: 处理以 /ack 结尾的主题

    } else {
        // [分支 4]: 其他未知或未注册的主题

    }
}
