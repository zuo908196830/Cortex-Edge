include($ENV{IDF_PATH}/tools/cmake/version.cmake)

set(srcs "mqtt.c")
set(public_requires "esp_event" "esp_netif" "mqtt" "led" "cjson" "handler" "model")

idf_component_register(SRCS ${srcs}
                       INCLUDE_DIRS "include"
                       REQUIRES ${public_requires})
