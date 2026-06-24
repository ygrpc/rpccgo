cmake_minimum_required(VERSION 3.22)

project(rpccgo_flutter_shared_jni LANGUAGES CXX)

set(RPCCGO_SHARED_LIB_DIR "${CMAKE_CURRENT_SOURCE_DIR}/../jniLibs/${ANDROID_ABI}")

add_library(rpccgo_flutter_shared_jni SHARED
    rpccgo/shared_so.shared_so_demo.jni.cpp
)

target_compile_features(rpccgo_flutter_shared_jni PRIVATE cxx_std_17)
target_include_directories(rpccgo_flutter_shared_jni PRIVATE
    "${RPCCGO_SHARED_LIB_DIR}"
)
target_link_libraries(rpccgo_flutter_shared_jni PRIVATE
    "-L${RPCCGO_SHARED_LIB_DIR}"
    "-l:librpccgo_flutter_shared.so"
    log
)
