include($ENV{IDF_PATH}/tools/cmake/version.cmake)

set(srcs "router.c" "register.c" "command.c")
set(public_requires "led" "cjson" "esp_event")

idf_component_register(SRCS ${srcs}
                       INCLUDE_DIRS "include"
                       REQUIRES ${public_requires})
