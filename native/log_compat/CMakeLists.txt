cmake_minimum_required(VERSION 3.5.0)
project(ohos_android_log_compat C)

add_library(log SHARED log.c)
target_include_directories(log PUBLIC ${CMAKE_CURRENT_SOURCE_DIR}/include)
target_link_libraries(log PUBLIC libhilog_ndk.z.so)
