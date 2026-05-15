###程序名称：FruitSortBackend
版本：v1.0
日期：2026-05-16

功能描述：用于鸿蒙前端水果分选功能的后端逻辑，负责TCP数据解析、状态管理和结果下发。


本文档说明 Go 后端如何接收 CTCP/TCP 数据，解析成 `StGlobal` / `StStatistics`，再通过 WebSocket 推送给鸿蒙前端。

核心结论：

```text
下位机 / FSM
    ↓ TCP
Go CTCP Server
    ↓ 根据命令号解析 payload
StGlobal / StStatistics
    ↓ json.MarshalIndent
JSON 字符串
    ↓ PublishWebSocketJSON
WebSocket topic=stglobal / statistics
    ↓
鸿蒙前端 HarmonyWebSocketClient 接收
```

## 1. HTTP / WebSocket 服务入口

位置：

```text
E:\goTest\go\ohos\Tcp\server.go
```

关键代码：

```go
const (
	serverHost = "127.0.0.1"
	serverPort = 18080
	serverAddr = "127.0.0.1:18080"
)
```

说明：

Go 后端启动的是本地服务：

```text
127.0.0.1:18080
```

前端 WebSocket 连接地址对应：

```text
ws://127.0.0.1:18080/ws/data
```

路由注册位置：

```text
E:\goTest\go\ohos\Tcp\server.go
```

代码：

```go
func newRouter() http.Handler {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.GET("/ping", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"framework": "gin",
			"message":   "pong from Go",
		})
	})
	registerWebSocketRoutes(router)
	return router
}
```

这里调用：

```go
registerWebSocketRoutes(router)
```

进入 WebSocket 路由注册。

## 2. WebSocket 路由和连接建立

位置：

```text
E:\goTest\go\ohos\Tcp\websocket.go
```

路由代码：

```go
func registerWebSocketRoutes(router *gin.Engine) {
	router.GET("/ws", handleWebSocket)
	router.GET("/ws/data", handleWebSocketData)
}

func handleWebSocketData(ctx *gin.Context) {
	handleWebSocket(ctx)
}
```

说明：

`/ws` 和 `/ws/data` 最后都会走同一个处理函数：

```go
handleWebSocket(ctx)
```

真正升级 WebSocket 的位置：

```go
func handleWebSocket(ctx *gin.Context) {
	defaultWebSocketHub.start()

	conn, err := webSocketUpgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		return
	}

	client := &webSocketClient{
		hub:  defaultWebSocketHub,
		conn: conn,
		send: make(chan []byte, webSocketSendBufferSize),
	}

	client.hub.register <- client
	go client.writePump()

	client.sendFrame(webSocketFrame{
		Type: "ready",
	})
	client.readPump()
}
```

这里做了几件事：

```text
1. 启动 defaultWebSocketHub
2. 把 HTTP 连接升级成 WebSocket
3. 创建 webSocketClient
4. 注册到 hub
5. 启动 writePump，专门负责给前端写数据
6. 发送 ready 帧
7. readPump 开始读取前端发来的消息
```

## 3. WebSocket Hub 的作用

位置：

```text
E:\goTest\go\ohos\Tcp\websocket.go
```

结构体：

```go
type webSocketHub struct {
	once       sync.Once
	register   chan *webSocketClient
	unregister chan *webSocketClient
	broadcast  chan []byte
	clients    map[*webSocketClient]struct{}
}
```

说明：

`hub` 是 WebSocket 客户端管理中心，负责：

```text
register   新客户端连接
unregister 客户端断开
broadcast  后端有新数据时广播给所有客户端
clients    当前所有 WebSocket 客户端
```

广播逻辑：

```go
case payload := <-h.broadcast:
	for client := range h.clients {
		select {
		case client.send <- payload:
		default:
			delete(h.clients, client)
			close(client.send)
		}
	}
```

即使现在只有 1 到 3 个客户端，也保留 hub 的原因是：

```text
1. 避免多个 goroutine 同时写同一个 websocket.Conn
2. 统一管理连接断开
3. 后端只需要 PublishWebSocketJSON，不需要知道有几个前端
4. 后面增加调试页面或多个前端时不用改广播逻辑
```

## 4. 前端连接后为什么要发 requestStGlobal

问题背景：

如果 Go 后端启动时立刻请求 `StGlobal`，但是鸿蒙前端还没连上 WebSocket，那么前端会漏掉第一次全量配置。

所以现在改成：

```text
前端 WebSocket 连接成功
    ↓
发送 {"type":"requestStGlobal"}
    ↓
Go 后端收到这个信号
    ↓
Go 再向 FSM 发送 DISPLAY_ON / SYNC 请求
    ↓
FSM 返回 StGlobal
    ↓
Go 推送 topic=stglobal 给前端
```

Go 接收前端消息的位置：

```text
E:\goTest\go\ohos\Tcp\websocket.go
```

代码：

```go
func (c *webSocketClient) handleIncoming(payload []byte) {
	text := strings.TrimSpace(string(payload))
	if strings.EqualFold(text, "ping") {
		c.sendFrame(webSocketFrame{Type: "pong"})
		return
	}

	control, ok := parseWebSocketControlMessage(text)
	if !ok {
		return
	}

	switch control.Type {
	case "requestStGlobal":
		c.handleRequestStGlobal()
	}
}
```

处理 `requestStGlobal`：

```go
func (c *webSocketClient) handleRequestStGlobal() {
	// 前端 WebSocket 连接成功后发 requestStGlobal，表示前端已经准备接收数据。
	// 这里异步触发 CTCP 客户端发送 DISPLAY_ON，避免阻塞 WebSocket 的读循环。
	go func() {
		if result := RequestStGlobalFromDefaultFSM(); result != 0 {
			setCTCPServerLastMessage("WebSocket requestStGlobal failed: result=%d", result)
		}
	}()
}
```

真正发送请求的位置：

```text
E:\goTest\go\ohos\Tcp\ctcpclient.go
```

代码：

```go
func RequestStGlobalFromDefaultFSM() int {
	// 前端 WebSocket 已连接后调用这个入口。发送SYNC
	return StartCTCPClient("", 0, cTCPDefaultFSMID, cTCPHCDisplayOn, nil)
}
```

## 5. CTCP 命令号

位置：

```text
E:\goTest\go\ohos\Tcp\ctcpserver.go
```

关键命令：

```go
const (
	cmdFSMConfig     = int32(0x1000)
	cmdFSMStatistics = int32(0x1001)
	cmdFSMGradeInfo  = int32(0x1002)
)
```

目前重点是两个：

```text
0x1000 -> StGlobal，全局配置
0x1001 -> StStatistics，实时统计
```

## 6. StGlobal 数据链路

结构体位置：

```text
E:\goTest\go\ohos\Tcp\ctcp_structs.go
```

结构体入口：

```go
type StGlobal struct {
	Sys   StSysConfig
	Grade StGradeInfo
	GExit StGlobalExitInfo

	GWeight StGlobalWeightBaseInfo
	AnalogDensity StAnalogDensity
	Exit          [12]StExitInfo
	Paras         [12]StParas
	Weights       [12]StWeightBaseInfo
	Motor         [48]StMotorInfo
	CFSMInfo      [12]uint8
	CIPMInfo      [12]uint8

	NSubsysId int32
	NVersion  int32
}
```

解析位置：

```text
E:\goTest\go\ohos\Tcp\ctcpserver.go
```

代码：

```go
case cmdFSMConfig:
	stg, err := ParseData[StGlobal](payload)
	if err != nil {
		return
	}
	stgJSON, jsonErr := FormatDataFullJSON(stg)
	goSz := int(unsafe.Sizeof(StGlobal{}))
	setCTCPServerLastMessage(
		"CTCP %s: sizeof(StGlobal)=%d, payload=%d bytes, nSubsysId=%d, nVersion=%d",
		cTCPCommandName(head.NCmdId),
		goSz,
		len(payload),
		stg.NSubsysId,
		stg.NVersion,
	)
	if jsonErr == nil && stgJSON != "" {
		setCTCPLastStGlobalFullJSON(stgJSON)
		if err := PublishWebSocketJSON(webSocketTopicStGlobal, stgJSON); err != nil {
			setCTCPServerLastMessage("CTCP StGlobal WebSocket 推送失败: %v", err)
		}
		appendCTCPLogChunks("CTCP StGlobal 全量", stgJSON)
	} else {
		setCTCPServerLastMessage("CTCP StGlobal 全量 JSON 生成失败: %v", jsonErr)
	}
```

这段代码的含义：

```text
payload
    ↓
ParseData[StGlobal](payload)
    ↓
得到 Go 结构体 StGlobal
    ↓
FormatDataFullJSON(stg)
    ↓
得到 JSON 字符串
    ↓
PublishWebSocketJSON("stglobal", stgJSON)
    ↓
推给前端
```

## 7. StStatistics 数据链路

结构体位置：

```text
E:\goTest\go\ohos\Tcp\ctcp_structs.go
```

当前 Go 后端使用的结构体：

```go
type StStatistics struct {
	NGradeCount         [256]uint64
	NWeightGradeCount   [256]uint64
	NExitCount          [48]uint64
	NExitWeightCount    [48]uint64
	NChannelTotalCount  uint64
	NChannelWeightCount uint64
	NSubsysId           int32
	NBoxGradeCount      [256]int32
	NBoxGradeWeight     [256]int32

	NTotalCupNum          int32
	NInterval             int32
	NIntervalSumperminute int32
	NCupState             uint16
	NPulseInterval        uint16
	NUnpushFruitCount     uint16

	NNetState      uint8
	NWeightSetting uint8

	NSCMState    uint8
	NIQSNetState uint8
	NLockState   uint8

	ExitBoxNum [48]uint32
}
```

解析位置：

```text
E:\goTest\go\ohos\Tcp\ctcpserver.go
```

代码：

```go
case cmdFSMStatistics:
	state, err := parseStStatistics48Payload(payload)
	if err != nil {
		setCTCPServerLastMessage("CTCP StStatistics 解析失败: %v", err)
		return
	}

	appendCTCPLogChunks("CTCP StStatistics Go结构体", fmt.Sprintf("%+v", state))

	stateJSON, jsonErr := FormatDataFullJSON(state)
	if stateJSON != "" && jsonErr == nil {
		appendCTCPLogChunks("CTCP StStatistics JSON字符串", stateJSON)
		if err := PublishWebSocketJSON(webSocketTopicStatistics, stateJSON); err != nil {
			setCTCPServerLastMessage("CTCP StStatistics WebSocket 推送失败: %v", err)
		}
		return
	} else {
		setCTCPServerLastMessage("CTCP StStatistics JSON 生成失败: %v", jsonErr)
	}
```

这段代码的含义：

```text
payload
    ↓
parseStStatistics48Payload(payload)
    ↓
得到 Go 结构体 StStatistics
    ↓
输出 Go 结构体日志
    ↓
FormatDataFullJSON(state)
    ↓
输出 JSON 字符串日志
    ↓
PublishWebSocketJSON("statistics", stateJSON)
    ↓
推给前端
```

## 8. StStatistics 为什么没有直接用 ParseData

`StGlobal` 可以直接：

```go
ParseData[StGlobal](payload)
```

但是 `StStatistics` 现在单独走：

```go
parseStStatistics48Payload(payload)
```

原因是当前 Go 端的 `StStatistics` 结构体是为了前端显示整理过的结构，不完全等同于原始 48 工程里的内存布局。

所以这里手动按固定位置读取字段，然后填入当前 Go 结构体。

核心代码：

```go
func parseStStatistics48Payload(payload []byte) (StStatistics, error) {
	const (
		gradeCountOffset         = 0
		weightGradeCountOffset   = gradeCountOffset + 256*8
		exitCountOffset          = weightGradeCountOffset + 256*8
		exitWeightCountOffset    = exitCountOffset + 48*8
		channelTotalCountOffset  = exitWeightCountOffset + 48*8
		channelWeightCountOffset = channelTotalCountOffset + 12*8
		nSubsysIDOffset          = channelWeightCountOffset + 12*8
	)
```

然后逐个读取：

```go
for i := 0; i < len(state.NGradeCount); i++ {
	state.NGradeCount[i] = readUint64LE(payload, gradeCountOffset+i*8)
}

for i := 0; i < len(state.NWeightGradeCount); i++ {
	state.NWeightGradeCount[i] = uint64FromDouble(readFloat64LE(payload, weightGradeCountOffset+i*8))
}

for i := 0; i < len(state.NExitCount); i++ {
	state.NExitCount[i] = readUint64LE(payload, exitCountOffset+i*8)
}

for i := 0; i < len(state.NExitWeightCount); i++ {
	state.NExitWeightCount[i] = uint64FromDouble(readFloat64LE(payload, exitWeightCountOffset+i*8))
}
```

通道统计现在汇总成一个总值：

```go
for i := 0; i < 12; i++ {
	state.NChannelTotalCount += readUint64LE(payload, channelTotalCountOffset+i*8)
	state.NChannelWeightCount += uint64FromDouble(readFloat64LE(payload, channelWeightCountOffset+i*8))
}
```

所以后端发给前端的 JSON 里：

```text
NChannelTotalCount  是一个总数
NChannelWeightCount 是一个总重量
```

前端 mapper 需要兼容这种标量值。

## 9. JSON 转换

位置：

```text
E:\goTest\go\ohos\Tcp\ctcp_stglobal_to_json.go
```

代码：

```go
func FormatDataFullJSON[T any](data T) (string, error) {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
```

说明：

这个函数把 Go 结构体转成 JSON 字符串。

注意：

```text
Go 结构体字段必须首字母大写，json.Marshal 才能导出。
```

比如：

```go
NGradeCount
NSubsysId
ExitBoxNum
```

如果没有写 json tag，那么 JSON 字段名就是 Go 字段名：

```json
{
  "NGradeCount": [],
  "NSubsysId": 256,
  "ExitBoxNum": []
}
```

前端再通过 mapper 把这些大写字段转成内部小写字段。

## 10. WebSocket 推送格式

位置：

```text
E:\goTest\go\ohos\Tcp\websocket.go
```

WebSocket 帧结构：

```go
type webSocketFrame struct {
	Type  string          `json:"type"`
	Topic string          `json:"topic,omitempty"`
	Data  json.RawMessage `json:"data,omitempty"`
	At    int64           `json:"at"`
}
```

生成数据帧：

```go
func newWebSocketDataFrame(topic string, raw json.RawMessage) ([]byte, error) {
	if !json.Valid(raw) {
		return nil, errInvalidWebSocketJSON
	}
	return json.Marshal(webSocketFrame{
		Type:  "data",
		Topic: topic,
		Data:  cloneRawMessage(raw),
		At:    time.Now().UnixMilli(),
	})
}
```

所以前端最终收到的格式是：

```json
{
  "type": "data",
  "topic": "statistics",
  "data": {
    "NGradeCount": [522366, 0, 0],
    "NChannelTotalCount": 522366,
    "NChannelWeightCount": 234984990,
    "NSubsysId": 256
  },
  "at": 1710000000000
}
```

或者：

```json
{
  "type": "data",
  "topic": "stglobal",
  "data": {
    "Sys": {
      "ExitState": [],
      "NExitNum": 18
    },
    "NSubsysId": 40201,
    "NVersion": 131583
  },
  "at": 1710000000000
}
```

## 11. 发布函数 PublishWebSocketJSON

位置：

```text
E:\goTest\go\ohos\Tcp\websocket.go
```

代码：

```go
func PublishWebSocketJSON(topic string, jsonText string) error {
	topic = normalizeTopic(topic, webSocketTopicData)
	raw, err := normalizeWebSocketJSON(jsonText)
	if err != nil {
		return err
	}

	frame, err := newWebSocketDataFrame(topic, raw)
	if err != nil {
		return err
	}

	defaultWebSocketHub.publish(topic, frame)
	return nil
}
```

说明：

这个函数做了 4 件事：

```text
1. 标准化 topic
2. 校验 jsonText 是合法 JSON
3. 包成统一 WebSocket frame
4. 交给 hub 广播
```

后端业务代码只需要调用：

```go
PublishWebSocketJSON(webSocketTopicStGlobal, stgJSON)
PublishWebSocketJSON(webSocketTopicStatistics, stateJSON)
```

不需要关心当前有几个前端连接。

## 12. Topic 设计

位置：

```text
E:\goTest\go\ohos\Tcp\websocket.go
```

代码：

```go
const (
	webSocketTopicData       = "data"
	webSocketTopicStGlobal   = "stglobal"
	webSocketTopicStatistics = "statistics"
	webSocketTopicGrade      = "grade"
)
```

说明：

```text
stglobal   全局配置数据
statistics 实时统计数据
grade      等级配置数据
data       默认数据类型
```

前端根据 `topic` 判断数据类型：

```text
topic=stglobal   -> StGlobalJsonMapper
topic=statistics -> StStatisticsJsonMapper
```

## 13. 日志排查点

### 13.1 StGlobal 日志

位置：

```text
E:\goTest\go\ohos\Tcp\ctcpserver.go
```

关键日志：

```go
setCTCPServerLastMessage(
	"CTCP %s: sizeof(StGlobal)=%d, payload=%d bytes, nSubsysId=%d, nVersion=%d",
	cTCPCommandName(head.NCmdId),
	goSz,
	len(payload),
	stg.NSubsysId,
	stg.NVersion,
)
```

可以看：

```text
sizeof(StGlobal)
payload 长度
nSubsysId
nVersion
```

### 13.2 StStatistics Go 结构体日志

代码：

```go
appendCTCPLogChunks("CTCP StStatistics Go结构体", fmt.Sprintf("%+v", state))
```

可以看到 Go 解析后的结构体内容，例如：

```text
NGradeCount:[522366 0 0 ...]
NChannelTotalCount:522366
NChannelWeightCount:234984990
NSubsysId:256
```

### 13.3 StStatistics JSON 字符串日志

代码：

```go
appendCTCPLogChunks("CTCP StStatistics JSON字符串", stateJSON)
```

可以确认：

```text
结构体转 JSON 后字段是否正确
字段名是不是 NGradeCount / NSubsysId 这种大写
数值有没有变成 0
```

## 14. 一句话回答主管

可以这样说：

```text
Go 后端通过 CTCP Server 接收下位机 TCP 数据，根据命令号区分数据类型。
0x1000 解析成 StGlobal，0x1001 解析成 StStatistics。
解析后统一通过 FormatDataFullJSON 转成 JSON 字符串，再调用 PublishWebSocketJSON。
PublishWebSocketJSON 会把业务 JSON 包成 {type, topic, data, at} 这种统一 WebSocket 数据帧，然后交给 hub 广播给前端。
前端连接成功后会主动发 requestStGlobal，后端收到后再请求 StGlobal，这样避免前端还没连接时漏掉第一次全量配置。
```

## 15. 当前链路总图

```text
鸿蒙前端启动
    ↓
连接 ws://127.0.0.1:18080/ws/data
    ↓
Go handleWebSocket 建立连接
    ↓
前端发送 {"type":"requestStGlobal"}
    ↓
Go handleRequestStGlobal
    ↓
RequestStGlobalFromDefaultFSM
    ↓
StartCTCPClient 发送 DISPLAY_ON / SYNC 请求
    ↓
FSM 返回 0x1000 StGlobal
    ↓
Go ParseData[StGlobal]
    ↓
FormatDataFullJSON
    ↓
PublishWebSocketJSON("stglobal", stgJSON)
    ↓
前端收到 topic=stglobal
```

```text
FSM 周期上报统计
    ↓
Go CTCP Server 收到 0x1001
    ↓
parseStStatistics48Payload
    ↓
得到 StStatistics
    ↓
FormatDataFullJSON
    ↓
PublishWebSocketJSON("statistics", stateJSON)
    ↓
前端收到 topic=statistics
```
