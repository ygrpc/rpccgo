cmake_minimum_required(VERSION 3.22.1)

project(rpccgo_android_foreground_service_jni LANGUAGES CXX)

set(RPCCGO_SHARED_LIB_DIR "${CMAKE_CURRENT_SOURCE_DIR}/../jniLibs/${ANDROID_ABI}")

add_library(rpccgo_android_foreground_service_jni SHARED
    rpccgo/foreground_service.foreground_service_demo.jni.cpp
)

target_include_directories(rpccgo_android_foreground_service_jni PRIVATE
    "${RPCCGO_SHARED_LIB_DIR}"
)

target_link_libraries(rpccgo_android_foreground_service_jni PRIVATE
    "-L${RPCCGO_SHARED_LIB_DIR}"
    "-l:librpccgo_android_foreground_service.so"
    log
)
