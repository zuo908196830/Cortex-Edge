#pragma once
#include "cJSON.h"
#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif
#define DEFAULT_REGISTRY_SIZE 8

cJSON *device_info_json();

#ifdef __cplusplus
}
#endif
