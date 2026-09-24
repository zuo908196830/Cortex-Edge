#include "framework.h"
#include "framework_config.h"
#include "esp_log.h"
#include <string.h>
#include <stdlib.h>

static const char *TAG = "CAP_MGR";

#define INITIAL_CAPABILITY_CAPACITY 16

typedef struct {
    char *name;
    char *description;
    char *parameters;
    action_handler_t handler;
} internal_capability_t;

static struct {
    node_info_t node_info;
    internal_capability_t *capabilities;
    size_t count;
    size_t capacity;
    bool initialized;
} s_registry = {0};

static node_info_t s_default_info = {
    .device_id = FRAMEWORK_DEFAULT_DEVICE_ID,
    .device_name = FRAMEWORK_DEFAULT_DEVICE_NAME,
    .device_type = FRAMEWORK_DEFAULT_DEVICE_TYPE,
    .description = FRAMEWORK_DEFAULT_DEVICE_DESC,
    .sw_version = FRAMEWORK_DEFAULT_SW_VERSION,
    .power_on_state = FRAMEWORK_DEFAULT_POWER_ON_STATE,
    .location_id = FRAMEWORK_DEFAULT_LOCATION_ID,
};

esp_err_t framework_init(const node_info_t *info)
{
    if (s_registry.initialized) {
        ESP_LOGW(TAG, "Framework already initialized");
        return ESP_OK;
    }

    if (info != NULL) {
        s_registry.node_info.device_id = info->device_id ? strdup(info->device_id) : s_default_info.device_id;
        s_registry.node_info.device_name = info->device_name ? strdup(info->device_name) : s_default_info.device_name;
        s_registry.node_info.device_type = info->device_type ? strdup(info->device_type) : s_default_info.device_type;
        s_registry.node_info.description = info->description ? strdup(info->description) : s_default_info.description;
        s_registry.node_info.sw_version = info->sw_version ? strdup(info->sw_version) : s_default_info.sw_version;
        s_registry.node_info.power_on_state = info->power_on_state ? strdup(info->power_on_state) : s_default_info.power_on_state;
        s_registry.node_info.location_id = info->location_id ? strdup(info->location_id) : s_default_info.location_id;
    } else {
        s_registry.node_info = s_default_info;
    }

    s_registry.capacity = INITIAL_CAPABILITY_CAPACITY;
    s_registry.capabilities = (internal_capability_t *)calloc(s_registry.capacity, sizeof(internal_capability_t));
    if (s_registry.capabilities == NULL) {
        ESP_LOGE(TAG, "Failed to allocate memory for capability registry");
        return ESP_ERR_NO_MEM;
    }

    s_registry.count = 0;
    s_registry.initialized = true;

    ESP_LOGI(TAG, "Framework initialized for device: [%s] (%s)", 
             s_registry.node_info.device_id, s_registry.node_info.device_name);
    return ESP_OK;
}

const node_info_t *framework_get_node_info(void)
{
    if (!s_registry.initialized) {
        return &s_default_info;
    }
    return &s_registry.node_info;
}

esp_err_t framework_register_capability(const capability_desc_t *cap)
{
    if (!s_registry.initialized) {
        esp_err_t err = framework_init(NULL);
        if (err != ESP_OK) {
            return err;
        }
    }

    if (cap == NULL || cap->name == NULL || cap->handler == NULL) {
        ESP_LOGE(TAG, "Invalid capability registration parameters");
        return ESP_ERR_INVALID_ARG;
    }

    // 检查是否已有同名能力，如有则更新
    for (size_t i = 0; i < s_registry.count; i++) {
        if (strcmp(s_registry.capabilities[i].name, cap->name) == 0) {
            ESP_LOGW(TAG, "Updating existing capability: %s", cap->name);
            free(s_registry.capabilities[i].description);
            free(s_registry.capabilities[i].parameters);
            s_registry.capabilities[i].description = cap->description ? strdup(cap->description) : strdup("");
            s_registry.capabilities[i].parameters = cap->parameters ? strdup(cap->parameters) : strdup("");
            s_registry.capabilities[i].handler = cap->handler;
            return ESP_OK;
        }
    }

    // 扩容检查
    if (s_registry.count >= s_registry.capacity) {
        size_t new_cap = s_registry.capacity * 2;
        internal_capability_t *new_arr = (internal_capability_t *)realloc(
            s_registry.capabilities, new_cap * sizeof(internal_capability_t)
        );
        if (new_arr == NULL) {
            ESP_LOGE(TAG, "Failed to expand capability registry");
            return ESP_ERR_NO_MEM;
        }
        s_registry.capabilities = new_arr;
        s_registry.capacity = new_cap;
    }

    // 新增注册
    size_t idx = s_registry.count;
    s_registry.capabilities[idx].name = strdup(cap->name);
    s_registry.capabilities[idx].description = cap->description ? strdup(cap->description) : strdup("");
    s_registry.capabilities[idx].parameters = cap->parameters ? strdup(cap->parameters) : strdup("");
    s_registry.capabilities[idx].handler = cap->handler;
    s_registry.count++;

    ESP_LOGI(TAG, "Registered capability [%zu]: '%s' (%s)", 
             s_registry.count, cap->name, cap->description ? cap->description : "");
    return ESP_OK;
}

action_handler_t framework_find_capability_handler(const char *name)
{
    if (!s_registry.initialized || name == NULL) {
        return NULL;
    }

    for (size_t i = 0; i < s_registry.count; i++) {
        if (strcmp(s_registry.capabilities[i].name, name) == 0) {
            return s_registry.capabilities[i].handler;
        }
    }

    return NULL;
}

cJSON *framework_build_device_info_json(void)
{
    if (!s_registry.initialized) {
        return NULL;
    }

    cJSON *root = cJSON_CreateObject();
    if (root == NULL) {
        return NULL;
    }

    const node_info_t *info = &s_registry.node_info;
    cJSON_AddStringToObject(root, "id", info->device_id);
    cJSON_AddStringToObject(root, "name", info->device_name);
    cJSON_AddStringToObject(root, "type", info->device_type);
    cJSON_AddStringToObject(root, "description", info->description);
    cJSON_AddStringToObject(root, "sw_version", info->sw_version);
    cJSON_AddStringToObject(root, "power_on_state", info->power_on_state);
    cJSON_AddStringToObject(root, "location_id", info->location_id);

    // 组装能力数组
    cJSON *caps_arr = cJSON_CreateArray();
    if (caps_arr == NULL) {
        cJSON_Delete(root);
        return NULL;
    }

    for (size_t i = 0; i < s_registry.count; i++) {
        cJSON *cap_item = cJSON_CreateObject();
        if (cap_item == NULL) {
            continue;
        }
        cJSON_AddStringToObject(cap_item, "name", s_registry.capabilities[i].name);
        cJSON_AddStringToObject(cap_item, "description", s_registry.capabilities[i].description);
        cJSON_AddStringToObject(cap_item, "parameters", s_registry.capabilities[i].parameters);
        cJSON_AddItemToArray(caps_arr, cap_item);
    }

    cJSON_AddItemToObject(root, "capabilities", caps_arr);
    return root;
}
