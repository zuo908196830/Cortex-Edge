include($ENV{IDF_PATH}/tools/cmake/version.cmake)

set(srcs "led.c")
set(public_requires "driver" "esp_driver_gpio" "esp_system" "handler")

idf_component_register(SRCS ${srcs}
                       INCLUDE_DIRS "include"
                       REQUIRES ${public_requires})

