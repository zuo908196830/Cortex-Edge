#pragma once

#include <stddef.h>
#include "cJSON.h"

#ifdef __cplusplus
extern "C" {
#endif

/**
 * @brief MQTT 主题路由分发函数
 * 
 * @param topic   收到数据的 Topic NUL 结尾字符串
 * @param msg     收到的 Payload 数据指针
 * @param msg_len Payload 数据字节长度
 */
void route(const char *topic, const char *msg, const int msg_len);

void handle_command(cJSON *json);

#ifdef __cplusplus
}
#endif
