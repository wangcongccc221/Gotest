# Golang 跨端能力验证测试文档

## 1. 验证范围

本次验证 Golang 通过 Native 动态库方式在 HarmonyOS 应用中运行，覆盖以下能力：

- HTTP 服务端，Web 框架选择 Gin
- WebSocket 服务端，基于 Gin + gorilla/websocket
- TCP 服务端与 TCP 客户端
- OPC UA 服务端与辅助客户端接口
- Modbus TCP 服务端与辅助客户端接口
- ORM 框架，选择 GORM

整体调用链路：

```text
ArkTS/ETS -> C++ NAPI -> Go c-shared 动态库 -> Go 功能模块
```
### 测试结论

| 能力 | 验证结果 |
| --- | --- |
| HTTP / Gin | 通过 |
| WebSocket 服务端 | 通过 |
| TCP 服务端 | 通过 |
| TCP 客户端 | 通过 |
| OPC UA 服务端 | 通过 |
| Modbus TCP 服务端 | 通过 |
| GORM ORM | 通过 |
| SQLite 文件落盘 | 通过 |

结论：

```text
Golang 可以通过 c-shared 动态库方式集成到 HarmonyOS Native 层，
并通过 C++ NAPI 暴露给 ArkTS 使用。
本次验证的 HTTP、WebSocket、TCP、OPC UA、Modbus TCP、GORM ORM 均已跑通。
```

## 2. 框架选择说明

### 2.1 Web 框架选择 Gin

选择原因：

- Gin 是 Go 生态中成熟度较高的 Web 框架，使用范围广，资料丰富。
- Gin 基于 Go 标准库 `net/http`，适合在 Native 动态库中嵌入 HTTP 服务。
- Gin 路由写法简单，方便统一暴露 HTTP、WebSocket、OPC UA、Modbus、ORM 等测试接口。
- Gin 社区活跃，长期维护，适合作为本次跨端能力验证的 Web 框架。

### 2.2 ORM 框架选择 GORM

选择原因：

- GORM 是 Go 生态中成熟度较高的 ORM 框架。
- GORM 支持模型映射、自动迁移、增删改查、关联关系、Hook 等常见 ORM 能力。
- GORM 社区成熟，文档完善，使用人数多，后续维护成本较低。
- 本项目使用 `github.com/glebarez/sqlite` 作为纯 Go SQLite Driver，避免 C SQLite 依赖带来的 HarmonyOS 交叉编译复杂度。

## 3. HTTP / Gin 验证

服务端口：

```text
18080
```

测试接口：

```text
GET http://设备IP:18080/ping
```

验证结果：

```json
{
  "framework": "gin",
  "message": "pong from Go"
}
```

结论：HTTP 服务端运行正常，Gin 框架可在 HarmonyOS Go 动态库中使用。

截图：

![HTTP Gin 接口返回结果](image-1.png)

## 4. WebSocket 服务端验证

WebSocket 地址：

```text
ws://设备IP:18080/ws
```

测试方式：

```text
连接 WebSocket 后发送文本消息
```

验证结果：


![WebSocket 回显测试结果](image-2.png)

结论：WebSocket 服务端可正常连接和回显消息。

## 5. TCP 服务端与客户端验证

TCP 服务端端口：

```text
18081
```

TCP 客户端本地端口：

```text
18082
```
![alt text](image-7.png)


结论：Go TCP 服务端与 TCP 客户端均可在应用中运行，消息收发正常。


## 6. OPC UA 验证

OPC UA 服务端地址：

```text
opc.tcp://设备IP:4840
```

测试节点：

```text
ns=1;s=Tag1
ns=1;s=Tag2
ns=1;s=Message
ns=1;s=Running
ns=1;s=Timestamp
```

验证结果：

```text
OPC UA 客户端可连接服务端，并可读取 HarmonyGo 命名空间下的节点数据。
```

结论：OPC UA 服务端可在 HarmonyOS Go 动态库中运行。

截图：

![OPC UA 客户端连接与节点读取结果](image-3.png)

## 7. Modbus TCP 验证

Modbus TCP 服务端地址：

```text
设备IP:5020
```

支持功能：

```text
FC3  读取保持寄存器
FC6  写单个寄存器
FC16 写多个寄存器
```

初始寄存器：

```text
0 -> 100
1 -> 200
2 -> 300
3 -> 400
4 -> 500
```

验证结果：

截图：

![Modbus TCP 寄存器读取结果](image-4.png)

```text
Modbus TCP 工具可连接 5020 端口，并可读取 Holding Registers。
```

结论：Modbus TCP 服务端可在 HarmonyOS Go 动态库中运行，寄存器读写正常。


## 8. ORM框架验证

ORM 框架：

```text
gorm
```

SQLite Driver：

```text
github.com/glebarez/sqlite
```

数据库文件：

```text
/data/app/el2/100/database/com.nutpi.myapplication/entry/harmony_go_orm.db
```

测试接口：

```text
GET  http://设备IP:18080/orm
GET  http://设备IP:18080/orm/devices
POST http://设备IP:18080/orm/devices?name=PLC-A&protocol=Modbus%20TCP&endpoint=tcp://127.0.0.1:5020
```

验证结果：

```text
GORM 初始化成功。
SQLite 数据库文件成功落盘。
可查询默认数据。
可新增数据。
重启应用后数据仍然存在。
```

结论：GORM 可在 HarmonyOS Go 动态库中运行，并可完成本地 SQLite 文件持久化。

截图：

![GORM SQLite 数据库落盘结果](image-6.png)


