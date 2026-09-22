#include "register.h"
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

typedef struct {
    char *key;            // 指令名称字符串
    cmd_handler_t handler;// 绑定的用户回调函数
    uint8_t occupied;     // 标记槽位是否有效
} cmd_entry_t;

struct command_registry {
    cmd_entry_t *entries; // 指令映射表
    size_t capacity;      // 映射表容量
    size_t size;          // 当前已注册的指令数量
};

command_registry_t *global_registry = NULL; // 全局指令注册表实例

// 经典 FNV-1a 字符串哈希算法
static inline uint32_t hash_fnv1a(const char *str) {
    uint32_t hash = 2166136261u;
    while (*str) {
        hash ^= (uint8_t)(*str++);
        hash *= 16777619u;
    }
    return hash;
}

// 向上补齐到最近的 2 的 N 次幂
static size_t next_pow2(size_t n) {
    size_t p = DEFAULT_REGISTRY_SIZE;
    while (p < n) p <<= 1;
    return p;
}

void init_register(size_t cap) {
    command_registry_t *registry = (command_registry_t *)malloc(sizeof(command_registry_t));
    if (!registry) return;

    registry->capacity = next_pow2(cap);
    registry->size = 0;
    registry->entries = (cmd_entry_t *)calloc(registry->capacity, sizeof(cmd_entry_t));
    if (!registry->entries) {
        free(registry);
        return;
    }
    global_registry = registry;
}

void free_register() {
    if (!global_registry) return;
    for (size_t i = 0; i < global_registry->capacity; ++i) {
        if (global_registry->entries[i].occupied) {
            free(global_registry->entries[i].key);
        }
    }
    free(global_registry->entries);
    free(global_registry);
}

void regist(const char *key, cmd_handler_t handler) {
    if (!global_registry || !key || !handler) return;

    uint32_t hash = hash_fnv1a(key);
    size_t index = hash % global_registry->capacity;

    // 线性探测法处理哈希冲突
    while (global_registry->entries[index].occupied) {
        if (strcmp(global_registry->entries[index].key, key) == 0) {
            // 如果指令已存在，更新回调函数
            global_registry->entries[index].handler = handler;
            return;
        }
        index = (index + 1) % global_registry->capacity;
    }

    // 插入新指令
    global_registry->entries[index].key = strdup(key);
    global_registry->entries[index].handler = handler;
    global_registry->entries[index].occupied = 1;
    global_registry->size++;
}

cmd_handler_t get_handler(const char *key) {
    if (!global_registry || !key) return NULL;

    uint32_t hash = hash_fnv1a(key);
    size_t index = hash % global_registry->capacity;

    // 线性探测法查找指令
    while (global_registry->entries[index].occupied) {
        if (strcmp(global_registry->entries[index].key, key) == 0) {
            return global_registry->entries[index].handler;
        }
        index = (index + 1) % global_registry->capacity;
    }

    // 如果未找到，返回 NULL
    return NULL;
}
