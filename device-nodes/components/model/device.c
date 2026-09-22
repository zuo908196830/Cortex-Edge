#include "device.h"
#include "cJSON.h"
#include "led.h"

// 自描述 json 串
cJSON *device_info_json() {
    cJSON *device_info = cJSON_CreateObject();
    if (device_info == NULL) {
        return NULL;
    }

    char *id = "ESP32_001";
    char *name = "灯";
    char *type = "light";
    char *description = "房间中的灯，仅支持开关";
    char *sw_version = "0.0.1";
    char *power_on_state = "off";
    char *location_id = "room_001";

    cJSON_AddStringToObject(device_info, "id", id);
    cJSON_AddStringToObject(device_info, "name", name);
    cJSON_AddStringToObject(device_info, "type", type);
    cJSON_AddStringToObject(device_info, "description", description);
    cJSON_AddStringToObject(device_info, "sw_version", sw_version);
    cJSON_AddStringToObject(device_info, "power_on_state", power_on_state);
    cJSON_AddStringToObject(device_info, "location_id", location_id);

    // 注册设备能力列表
    cJSON *capabilities = led_capabilities_json();
    if (capabilities == NULL) {
        cJSON_Delete(device_info);
        return NULL;
    }
    cJSON_AddItemToObject(device_info, "capabilities", capabilities);

    return device_info;
}
