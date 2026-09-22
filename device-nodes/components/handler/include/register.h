#pragma once
#include "cJSON.h"
#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif
#define DEFAULT_REGISTRY_SIZE 8

typedef void (*cmd_handler_t)(cJSON *args);
typedef struct command_registry command_registry_t;

void init_register(size_t cap);
void free_register();
void regist(const char *key, cmd_handler_t handler);
cmd_handler_t get_handler(const char *key);

#ifdef __cplusplus
}
#endif
