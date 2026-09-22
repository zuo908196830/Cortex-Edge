include($ENV{IDF_PATH}/tools/cmake/version.cmake)

set(srcs "exti.c")
set(public_requires "driver" "esp_system" "led" "esp_timer")

idf_component_register(SRCS ${srcs}
                       INCLUDE_DIRS "include"
                       REQUIRES ${public_requires})

