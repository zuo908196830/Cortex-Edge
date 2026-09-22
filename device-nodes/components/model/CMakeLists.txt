include($ENV{IDF_PATH}/tools/cmake/version.cmake)

set(srcs "device.c")
set(public_requires "cjson" "led")

idf_component_register(SRCS ${srcs}
                       INCLUDE_DIRS "include"
                       REQUIRES ${public_requires})
