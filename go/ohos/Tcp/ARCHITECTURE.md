# Tcp 后端模块化设计

目标：以后找代码时按功能进入目录，不再从一个大包里扫文件。

## 当前状态

已经拆出的真 package：

- `Tcp/client`：普通 TCP 客户端、CTCP 指令下发、FSM/WAM/IPM 目标地址解析。
- `Tcp/protocol`：FSM/WAM/IPM 线上的结构体、结构体布局检查。
- `Tcp/events`：CTCP 日志缓冲、最后消息、WebSocket 发布接口。
- `Tcp/state`：第一批运行时快照，包括 StGradeInfo、StExitInfos、StMotorInfo、按 destID 缓存的等级信息、最近一次 StGlobal JSON。
- `Tcp/internal/runtime`：暂存当前强耦合运行链路。

`internal/runtime` 不能长期保留成大包。它现在只是过渡层，后续要继续拆。

## 设计原则

1. 依赖只能往底层走，不能循环 import。
2. `store` 只负责读写配置，不负责推 WebSocket。
3. `websocket` 只负责前端连接和命令入口，不直接保存复杂运行状态。
4. `state` 管所有内存快照，例如 StGlobal、StGradeInfo、出口显示配置、统计快照。
5. `events` 管日志和 WebSocket 发布接口，让 server/store/stats 不直接依赖 websocket。
6. `app` 负责启动、停止、把各模块接起来。

## 目标目录

```text
Tcp/
  api.go
  README.md
  ARCHITECTURE.md

  app/
    app.go

  client/
    ctcp_client.go
    tcp_client.go
    exports.go

  protocol/
    ctcp_structs.go
    ctcp_layout.go
    exports.go

  events/
    log.go
    publisher.go
    topics.go

  state/
    stglobal.go
    grade.go
    statistics.go
    exit_display.go
    project.go

  store/
    exit_infos.go
    exit_display.go
    exit_additional_text.go
    fruit_type_config.go
    level_aux_config.go
    project_scheme.go

  websocket/
    hub.go
    frame.go
    handlers.go
    commands_fsm.go
    commands_wam.go
    commands_ipm.go
    commands_grade.go
    exit_display.go
    database.go
    project_config.go

  receiver/
    server.go
    protocol_read.go
    command_handlers.go

  stats/
    statistics.go
    home.go
    csv.go

  realtime/
    save.go
    builder.go
    rows.go
    delta.go
    names.go
    log.go

  exitdisplay/
    broadcast.go

  about/
    api.go

  auth/
    project_auth.go
```

## 依赖方向

```text
api
  -> app

app
  -> receiver
  -> websocket
  -> stats
  -> realtime
  -> store
  -> events

receiver
  -> protocol
  -> events
  -> state
  -> stats
  -> websocket publisher interface

websocket
  -> client
  -> protocol
  -> store
  -> state
  -> events
  -> realtime

stats
  -> protocol
  -> state
  -> events
  -> realtime

realtime
  -> protocol
  -> state
  -> database

store
  -> protocol
  -> database
  -> events

state
  -> protocol

client
  -> events

protocol
  -> standard library only
```

禁止：

```text
store -> websocket
state -> websocket
state -> database
protocol -> anything in Tcp
client -> runtime/app/websocket
```

## 找代码地图

| 你要改什么 | 去哪里 |
| --- | --- |
| FSM/WAM/IPM 结构体字段 | `Tcp/protocol` |
| 下发设备指令、目标 IP/端口、destID 编码 | `Tcp/client` |
| 前端 WebSocket 连接、topic、命令入口 | `Tcp/websocket` |
| 前端查询数据库 | `Tcp/websocket/database.go`，后续可进 `Tcp/query` |
| 本地配置读写 | `Tcp/store` |
| StGlobal/StGradeInfo/统计快照 | `Tcp/state` |
| CTCP 1127/1128 接收数据 | `Tcp/receiver` |
| 本批重量、本批个数、分选效率 | `Tcp/stats` |
| 实时保存到数据库 | `Tcp/realtime` |
| 出口屏 UDP 广播 | `Tcp/exitdisplay` |
| 关于界面版本信息 | `Tcp/about` |
| 项目密码 | `Tcp/auth` |
| 启动/停止服务 | `Tcp/app` 和 `Tcp/api.go` |

## 拆分顺序

### 第一阶段：拆中间层

先拆：

- `events`：日志、topic、WebSocket 发布接口。
- `state`：运行时快照。

原因：这两个是 seam。没有它们，`store`、`websocket`、`receiver` 会互相 import，Go 会报循环引用。

### 第二阶段：拆纯业务模块

再拆：

- `store`
- `stats`
- `realtime`
- `exitdisplay`
- `about`
- `auth`

这些模块有明确职责，拆完后查代码会明显轻松。

### 第三阶段：拆入口编排

最后拆：

- `websocket`
- `receiver`
- `app`

这些现在互相调用最多，等前两阶段完成后再搬，风险最低。

## 每个包的接口方向

### events

对外提供：

- `SetLogger(func(string, ...any))`
- `Info(format string, args ...any)`
- `LastMessage() string`
- `SetPublisher(Publisher)`
- `PublishJSON(topic string, jsonText string) error`

### state

对外提供：

- `SetStGlobal(snapshot)`
- `LatestStGlobal()`
- `SetGradeInfo(destID, grade)`
- `LatestGradeInfo(destID)`
- `SetExitDisplay(...)`
- `LatestExitDisplay()`
- `SetStatistics(...)`
- `LatestStatistics(...)`

### store

对外只做读写：

- `ReadExitDisplay()`
- `SaveExitDisplay()`
- `ReadLevelAuxConfig()`
- `SaveLevelAuxConfig()`
- `ReadProjectScheme()`
- `ApplyProjectSchemeToLocalConfig()`

保存成功后是否推 WebSocket，由 `websocket` 或 `app` 决定，不在 `store` 里做。

### websocket

对外提供：

- `RegisterRoutes(router)`
- `PublishJSON(topic, jsonText)`
- `StartHub()`
- `StopHub()`

它调用 `store/client/state/stats`，但别的包不要反过来 import `websocket`。

### receiver

对外提供：

- `Start()`
- `Stop()`

收到数据后：

- 解析结构体：`protocol`
- 更新快照：`state`
- 发布前端消息：`events.PublishJSON`
- 统计计算：`stats`

## 判断是否拆对了

拆完一个包后必须满足：

1. 这个包能单独 `go test ./Tcp/<pkg>`。
2. 这个包不 import `Tcp/internal/runtime`。
3. 没有 import cycle。
4. 包名能回答“这个目录负责什么”。
5. 新人看目录名就能猜到入口文件。
