idf_component_register(SRCS "hello_world_main.c"
                       PRIV_REQUIRES spi_flash exti_demo wifi led app_mqtt
                       INCLUDE_DIRS "")
