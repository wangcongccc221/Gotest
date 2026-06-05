# OHOS Go Module Layout

这个目录是 HarmonyOS 侧调用的 Go 后端实现。当前整理采用低风险方案：只在同一个 package 内按职责拆文件，不改 package 名，不改对外 API。

## Modules

- `Tcp`
  - CTCP 客户端和服务端通信。
  - WebSocket/NAPI 桥接，接收前端请求并推送后端数据。
  - 下位机回推结构体解析、运行统计缓存、主页统计推送。
  - 实时加工数据的 3 秒调度、聚合、DTO 构造。

- `database`
  - SQLite/GORM 初始化和迁移。
  - `tb_*.go` 数据表模型。
  - 实时加工数据入库入口 `SaveRealtimeFruitInfo`。

- `modbus`
  - Modbus 服务端相关实现。

- `opcua`
  - OPC UA 服务端相关实现。

## Current Split

- `Tcp/ctcpserver.go`
  - CTCP 监听、连接生命周期、命令分发。

- `Tcp/ctcp_protocol.go`
  - CTCP SYNC、命令头、payload 读取协议。

- `Tcp/ctcp_statistics.go`
  - `StStatistics` 分选速度缓存、后端计算、WebSocket 推送。

- `Tcp/ctcp_config_cache.go`
  - `StGlobal` 中图像参数和出口配置的快照缓存、周期日志。

- `Tcp/realtime_save.go`
  - 实时保存调度、节流、状态和日志。

- `Tcp/realtime_save_builder.go`
  - 将 CTCP 统计和全局配置聚合成数据库保存输入。

- `Tcp/realtime_save_rows.go`
  - 构造等级、出口、子系统、加工过程行数据。

- `Tcp/realtime_save_names.go`
  - 固定长度 C 字符串、GBK/UTF-8 名称解析和数值保护。

- `database/realtime_save.go`
  - 实时保存 DTO 和外部入口。

- `database/realtime_save_rows.go`
  - 查询当前加工记录、替换等级/出口/子系统明细、写加工过程分钟表。

- `database/realtime_save_schema.go`
  - 实时保存涉及的字段集合和值映射。

## Data Flow

1. ArkTS 前端通过 NAPI/WebSocket 发送控制请求。
2. Go `Tcp/websocket.go` 接收请求并转成 CTCP 指令。
3. 下位机通过 CTCP 回推 `StGlobal`、`StStatistics`、等级、重量等结构体。
4. `Tcp/ctcpserver.go` 接收数据并分发解析。
5. `Tcp/ctcp_statistics.go` 缓存统计快照，并按周期推送主页统计。
6. `Tcp/realtime_save*.go` 每 3 秒从统计缓存和 `StGlobal` 快照构造保存输入。
7. `database.SaveRealtimeFruitInfo` 写入本地 SQLite/GORM 数据库。

## Refactor Rule

当前阶段不把 `Tcp` 或 `database` 继续拆成子 package。Go 文件可以继续按职责拆，但跨目录拆 package 会牵扯大量 import 和初始化顺序，留到后续大重构再做。
