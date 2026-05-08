# Go 在 HarmonyOS 上编译运行说明

## 1. 目标

本项目验证的是：使用 Go 编写业务能力，将 Go 代码交叉编译成 HarmonyOS 可加载的 Native 动态库，然后通过 C++ NAPI 暴露给 ArkTS 页面调用。

整体调用链路如下：

```text
ArkTS 页面
  -> import libentry.so
  -> C++ NAPI 层
  -> Go 导出的 C ABI 函数
  -> Go 内部业务代码
```

对应项目文件：

```text
entry/src/main/ets/pages/Index.ets
entry/src/main/cpp/napi_init.cpp
go/ohos/main.go
go/ohos/*.go
```

## 2. Go 动态库编译方式

Go 代码位于：

```text
go/ohos
```

编译脚本为：

```text
go/ohos/build_ohos.ps1
```

修改 Go 代码后，需要重新执行：

```powershell
.\go\ohos\build_ohos.ps1
```

该脚本主要做了几件事：

| 步骤 | 作用 |
| --- | --- |
| 设置 `OHOS_SDK` | 指向 HarmonyOS Native SDK |
| 设置 `GOARCH=arm64` | 编译 arm64 架构产物 |
| 设置 `GOOS=android` | 使用 Go Android cgo runtime，支持 NAPI 运行时加载 |
| 设置 `CGO_ENABLED=1` | 开启 cgo，允许 Go 导出 C ABI |
| 设置 `CC/CXX/AR/LD` | 使用 HarmonyOS SDK 内的 LLVM 工具链 |
| 设置 `--target=aarch64-linux-ohos` | 让 clang 按 HarmonyOS 目标平台编译 |
| 执行 `go build -buildmode=c-shared` | 生成 `.so` 动态库和 `.h` 头文件 |

这里使用 `GOOS=android` 的原因是：普通 Go 发行版没有官方 `GOOS=openharmony`，而 Go Linux/arm64 c-shared 产物带有 initial-exec TLS，不能被 HarmonyOS NAPI 运行时 `dlopen`。项目通过 `ohos_android_tls_wrap.go` 禁用 Android 固定 TLS_SLOT_APP 路径，让 Go runtime 使用 pthread key 探测 HarmonyOS 线程里的真实 TLS 位置。

## 3. 编译产物位置

执行 `go/ohos/build_ohos.ps1` 后，Go 动态库输出到：

```text
entry/libs/arm64-v8a/libohos.so
```

同时生成 Go 导出的 C 头文件：

```text
entry/libs/arm64-v8a/libohos.h
```

脚本还会把头文件复制到 C++ NAPI 的 include 目录：

```text
entry/src/main/cpp/include/libohos.h
```

这个头文件用于 C++ 层声明和调用 Go 导出的函数。

## 4. liblog.so 的作用

`GOOS=android` 方案会涉及 Android log 相关符号。HarmonyOS 没有原生 Android `liblog`，因此项目中实现了一个兼容层：

```text
native/log_compat
```

该兼容层会编译生成：

```text
entry/libs/arm64-v8a/liblog.so
```

它内部使用 HarmonyOS 的 `hilog` 实现 Android log 接口兼容，保证 Go 动态库能够正常加载。

## 5. C++ NAPI 层如何调用 Go

Go 函数在 `go/ohos/main.go` 中通过 `//export` 导出，例如：

```go
//export GoStartServer
func GoStartServer() C.int {
	return C.int(startServer())
}
```

编译后，`libohos.h` 会生成对应声明：

```c
extern int GoStartServer(void);
```

C++ NAPI 层在：

```text
entry/src/main/cpp/napi_init.cpp
```

里面直接调用这些 Go 导出函数，再把结果转换成 ArkTS 可接收的数据类型。例如：

```text
ArkTS 调用 startServer()
  -> C++ NAPI 调用 GoStartServer()
  -> Go 启动 Gin HTTP 服务
  -> C++ 返回端口号给 ArkTS
```

## 6. libentry.so 与 libohos.so 的关系

`libohos.so` 是 Go 编译出来的动态库。

`libentry.so` 是 HarmonyOS Native 模块，由 C++ NAPI 编译生成。

CMake 配置位于：

```text
entry/src/main/cpp/CMakeLists.txt
```

其中会链接 Go 动态库：

```text
entry/libs/arm64-v8a/libohos.so
```

最终 Native 构建输出：

```text
entry/build/default/intermediates/cmake/default/obj/arm64-v8a/libentry.so
```

ArkTS 页面实际 import 的是：

```ts
import testNapi from 'libentry.so';
```

也就是说，ArkTS 不直接调用 Go 动态库，而是先调用 `libentry.so`，再由 `libentry.so` 调用 `libohos.so`。

## 7. HarmonyOS 打包时包含哪些动态库

需要放入应用的动态库主要在：

```text
entry/libs/arm64-v8a/
```

当前关键文件：

```text
libohos.so
```

其中：

| 文件 | 来源 | 作用 |
| --- | --- | --- |
| `libohos.so` | Go 编译产物 | 承载 Go 业务能力 |
| `libentry.so` | HarmonyOS Native 构建产物 | NAPI 桥接层，供 ArkTS 调用 |

`libentry.so` 不需要手动放到 `entry/libs`，它由 DevEco/Hvigor Native 构建流程生成。

## 8. 修改 Go 代码后的流程

如果只是修改 Go 内部逻辑，例如 HTTP、TCP、OPC UA、Modbus、WebSocket、ORM 的实现：

```text
1. 修改 go/ohos/*.go
2. 执行 go/ohos/build_ohos.ps1
3. 重新运行或重新构建 HarmonyOS 应用
```

如果新增一个 ArkTS 需要直接调用的 Go 能力，需要额外增加桥接：

```text
1. 在 go/ohos/main.go 使用 //export 导出 Go 函数
2. 执行 go/ohos/build_ohos.ps1 重新生成 libohos.so 和 libohos.h
3. 在 entry/src/main/cpp/napi_init.cpp 增加 NAPI 包装函数
4. 在 napi_property_descriptor 中注册方法名
5. ArkTS 通过 libentry.so 调用该方法
```

## 9. 当前验证结论

本项目已经验证 Go 可以在 HarmonyOS 应用内以动态库方式运行。

当前 Go 动态库中已承载：

```text
Gin HTTP 服务
WebSocket 服务端
TCP 服务端和客户端
OPC UA 服务端
Modbus TCP 服务端
GORM ORM 数据库能力
```

结论：

```text
Go 源码可以通过 build_ohos.ps1 交叉编译为 HarmonyOS arm64 动态库 libohos.so。
HarmonyOS C++ NAPI 层可以链接该动态库，并将 Go 能力暴露给 ArkTS 调用。
应用运行时，Go 业务逻辑实际运行在 HarmonyOS App 进程内。
```
