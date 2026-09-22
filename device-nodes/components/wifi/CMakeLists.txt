include($ENV{IDF_PATH}/tools/cmake/version.cmake)

set(srcs "wifi.c")
set(public_requires "esp_wifi"  "nvs_flash" "esp_event" "esp_netif")

idf_component_register(SRCS ${srcs}
                       INCLUDE_DIRS "include"
                       REQUIRES ${public_requires})
