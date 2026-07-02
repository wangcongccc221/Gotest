# Tcp 目录说明

`Tcp` 根目录现在只保留对 Harmony/Go 外部调用稳定的入口。

Go 里面子文件夹就是新的 package，所以这里不是单纯搬文件：能独立的模块已经拆成真 package，剩余强耦合运行时先收进 `internal/runtime`，避免根目录继续像大杂烩。

后续完整拆分设计见 `ARCHITECTURE.md`。

## 根目录

- `api.go`：对外入口 wrapper。Harmony 侧继续调用 `tcp.StartServer()`、`tcp.StartCTCPServer()`、`tcp.StartTCPClient()` 等函数。

## 已拆出的功能包

- `client/`
  - 普通 TCP 客户端：连接、发送、关闭。
  - CTCP 指令下发客户端：FSM/WAM/IPM 指令目标地址解析、SYNC 下发、WAM/FSM ID 编码。

- `protocol/`
  - FSM/WAM/IPM 线上的结构体定义，如 `StGlobal`、`StStatistics`、`StGradeInfo`。
  - 结构体大小和布局检查，如 `StGlobalLayoutReport()`。

## 运行时包

- `internal/runtime/`
  - HTTP/Gin 服务入口。
  - CTCP 接收服务。
  - WebSocket 连接、前端命令分发。
  - 主页统计、实时保存、CSV 诊断。
  - 本地配置 store、出口屏、等级设置、项目方案。

`internal/runtime` 目前是下一阶段继续拆分的对象。这里面还有几组强耦合：

- `store` 相关文件不仅读写配置，还会更新缓存、推 WebSocket、改 StGlobal 快照。
- `stats/realtime/home_stats` 共享统计窗口、实时保存和主页聚合逻辑。
- `server/websocket` 双向使用消息发布、设备命令和缓存。

这些后续要拆，需要先设计接口，不适合只靠移动文件完成。
