# DISPLAY_ON 数据上行链路培训 PPT 设计

## 目标

制作一套约 30 分钟的中文培训 PPT，用一条真实数据链路解释：前端连接并发送请求，Go 后端把请求翻译为 FSM 指令，FSM 返回数据，Go 解析并推回前端，前端完成映射和页面显示。

培训结束后，听众应能：

1. 说清前端、Go 后端、FSM 三方各自负责什么。
2. 沿代码追踪 `requestStGlobal -> DISPLAY_ON -> 1128 -> stglobal/statistics -> AppStorage`。
3. 区分 WebSocket JSON、CTCP SYNC 二进制帧和 ArkTS 数据映射。
4. 数据不显示时，按链路节点判断问题停在哪一层。

## 受众与边界

- 听众包含前端和后端开发，均有开发经验，但没有使用过这套链路。
- 不讲 ArkTS、Go 或网络协议的基础语法，只解释理解链路所需的概念。
- 主案例固定为首页数据上行链路：`DISPLAY_ON` 指令与 FSM `1128` 回包。
- 不展开等级设置等其他业务闭环，只在需要对比时一句带过。
- 不修改任何业务代码。
- 代码片段全部来自当前工作区以及当前前端工程，标注真实文件路径和符号名。

## 交付物

- PowerPoint：`E:\goTest\outputs\一条数据的旅程-前端到FSM再回到界面.pptx`
- PPT 制作源文件：保存在 `E:\goTest\outputs`，便于后续更新代码片段和重新生成。

## 讲解结构

采用“跟着一条数据走完全程”的单线叙事，共 12 页：

1. **封面**：一条数据的旅程。
2. **三个角色**：前端、Go 后端、FSM 的职责和通信方式。
3. **全景链路**：一次请求从连接到显示的完整时序。
4. **连接成功**：WebSocket 建立、`ready`、重连与首轮请求的关系。
5. **前端请求**：前端发送的 JSON 及 `requestStGlobal` 等请求入口。
6. **后端路由**：Go 如何按 `type` 分发，并进入 `requestStGlobal` 处理器。
7. **发送指令**：Go 如何构造 16 字节 SYNC 帧并发送 `DISPLAY_ON`。
8. **FSM 返回**：FSM 反向连接 `1128`，Go 分三段读取并识别命令。
9. **解析与缓存**：二进制转 Go 结构体、结构体布局约束和缓存解耦。
10. **推回前端**：Go 统一封装 WebSocket `data/topic` 帧并广播。
11. **映射与显示**：前端按 topic 分发，映射 ArkTS 结构体，写入状态并刷新 UI。
12. **排查与复盘**：按八个节点检查日志、数据和状态，回看整条链路。

## 每页内容规则

- 每页只回答一个问题，正文不超过 5 个要点。
- 关键代码页展示 8 到 15 行代码，保留足够上下文，不截取孤立单行。
- 代码旁固定给出“输入、处理、输出”和一句结论。
- 首次出现的术语立即解释，例如 FSM、topic、SYNC、payload、映射。
- 不假设“连接成功”等于“业务数据已返回”，明确每个成功状态的边界。
- 详细代码只保留主路径；异常分支集中放在最后一页的排查表中。

## 视觉设计

- 16:9 宽屏，白色或浅灰背景，深灰正文。
- 蓝色只表示请求方向，绿色只表示返回方向，橙色用于风险和排查提示。
- 不使用装饰性渐变、复杂动画、立体图形或大面积图片。
- 流程图统一从左到右，三方泳道位置固定，避免页面之间角色跳位。
- 标题约 28 到 32pt，正文不低于 18pt，代码不低于 14pt。
- 页脚显示当前链路阶段，例如 `前端请求 / FSM 交互 / 返回显示`。

## 代码范围

Go 侧重点文件：

- `go/ohos/Tcp/internal/runtime/websocket.go`
- `go/ohos/Tcp/internal/runtime/websocket_handlers.go`
- `go/ohos/Tcp/client/ctcp_client.go`
- `go/ohos/Tcp/internal/runtime/ctcp_server.go`
- `go/ohos/Tcp/internal/runtime/ctcp_protocol.go`
- `go/ohos/Tcp/internal/runtime/ctcp_statistics.go`
- `go/ohos/Tcp/internal/runtime/home_stats.go`

ArkTS 侧重点文件：

- `entry/src/main/ets/entryability/EntryAbility.ets`
- `entry/src/main/ets/utils/network/HarmonyWebSocketClient.ets`
- `entry/src/main/ets/protocol/StStatisticsJsonMapper.ets`
- `entry/src/main/ets/protocol/GlobalDataInterface.ets`
- `entry/src/main/ets/protocol/UIDataSync.ets`

## 时间分配

- 角色、术语与全景链路：5 分钟。
- 连接和前端请求：5 分钟。
- Go 到 FSM、1128 返回与解析：10 分钟。
- 推回前端、映射和显示：6 分钟。
- 排查复盘与提问：4 分钟。

## 验证标准

- PowerPoint 和 WPS 均可打开，页数为 12 页。
- 逐页渲染后无文字溢出、代码截断、元素重叠或不可读的小字。
- 全景图与各代码页使用同一条链路、同一组命令和 topic，不混入其他案例。
- 所有代码符号、路径、端口、命令号和 JSON 字段与当前代码一致。
- 最后一页的排查顺序与前 11 页的链路顺序一致，不能新增另一套状态来源。
