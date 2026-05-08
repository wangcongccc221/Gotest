# the minimum version of CMake.
cmake_minimum_required(VERSION 3.5.0)
project(goTest)

set(NATIVERENDER_ROOT_PATH ${CMAKE_CURRENT_SOURCE_DIR})

if(DEFINED PACKAGE_FIND_FILE)
    include(${PACKAGE_FIND_FILE})
endif()

include_directories(${NATIVERENDER_ROOT_PATH}
                    ${NATIVERENDER_ROOT_PATH}/include)

set(GO_LIB_DIR "${NATIVERENDER_ROOT_PATH}/../../../libs/${OHOS_ARCH}")
set(GO_LIB "${GO_LIB_DIR}/libohos.so")

if(NOT EXISTS "${GO_LIB}")
    message(FATAL_ERROR "Missing ${GO_LIB}. Run go/ohos/build_ohos.sh before building the HarmonyOS app.")
endif()

add_library(entry SHARED napi_init.cpp)
set_target_properties(entry PROPERTIES
    SKIP_BUILD_RPATH TRUE
    BUILD_RPATH ""
    INSTALL_RPATH ""
)
target_link_libraries(entry PUBLIC libace_napi.z.so)
