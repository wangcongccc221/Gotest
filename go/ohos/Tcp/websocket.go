package tcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	webSocketTopicData       = "data"     //
	webSocketTopicStGlobal   = "stglobal" // StGlobal 全量数据
	webSocketTopicStatistics = "statistics"
	webSocketTopicGrade      = "grade"

	webSocketWriteWait           = 5 * time.Second  //写入等待
	webSocketPongWait            = 70 * time.Second // Pong 等待，比客户端心跳周期略长，允许偶尔的网络抖动
	webSocketPingPeriod          = 30 * time.Second
	webSocketMaxMessageSize      = 2056 * 2056 // 4MB 放数据用的
	webSocketSendBufferSize      = 32          //最多发送32条消息
	webSocketBroadcastBufferSize = 64          // 广播最多64条消息
)

var (
	errEmptyWebSocketJSON   = errors.New("websocket json is empty")
	errInvalidWebSocketJSON = errors.New("websocket json is invalid")
	defaultWebSocketHub     = newWebSocketHub() //
	gradeInfoCacheMu        sync.RWMutex
	gradeInfoCache          = make(map[int32]StGradeInfo)
)

var webSocketUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type webSocketFrame struct { //数据帧
	Type  string          `json:"type"`
	Topic string          `json:"topic,omitempty"`
	Data  json.RawMessage `json:"data,omitempty"`
	At    int64           `json:"at"` //时间戳
}

type webSocketControlMessage struct {
	Type       string               `json:"type"`
	FSMID      int32                `json:"fsmId,omitempty"`
	DestID     int32                `json:"destId,omitempty"`
	Grade      *StGradeInfo         `json:"grade,omitempty"`
	GradeExits []webSocketGradeExit `json:"gradeExits,omitempty"`
}

type webSocketGradeExit struct {
	Index int     `json:"index"`
	Exit  float64 `json:"exit"`
}

type webSocketHub struct {
	once       sync.Once
	register   chan *webSocketClient
	unregister chan *webSocketClient
	broadcast  chan []byte
	clients    map[*webSocketClient]struct{}
}

type webSocketClient struct {
	hub  *webSocketHub
	conn *websocket.Conn
	send chan []byte
}

func newWebSocketHub() *webSocketHub { //管理中心
	return &webSocketHub{
		register:   make(chan *webSocketClient),
		unregister: make(chan *webSocketClient),
		broadcast:  make(chan []byte, webSocketBroadcastBufferSize),
		clients:    make(map[*webSocketClient]struct{}),
	}
}

func registerWebSocketRoutes(router *gin.Engine) { //注册路由
	router.GET("/ws", handleWebSocket)
	router.GET("/ws/data", handleWebSocketData)
}

func handleWebSocketData(ctx *gin.Context) {
	handleWebSocket(ctx)
}

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

/*
topic: 主题 根据这个来区分数据类型
jsonText: JSON字符串， 传解析，转换之后的值
*/
func PublishWebSocketJSON(topic string, jsonText string) error { //
	topic = normalizeTopic(topic, webSocketTopicData)
	raw, err := normalizeWebSocketJSON(jsonText)
	if err != nil {
		return err
	}

	frame, err := newWebSocketDataFrame(topic, raw) //把数据打包成json字符串
	if err != nil {
		return err
	}

	defaultWebSocketHub.publish(topic, frame) //发布数据并广播
	return nil
}

func (h *webSocketHub) start() { //启动
	h.once.Do(func() {
		go h.run()
	})
}

func (h *webSocketHub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = struct{}{}
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case payload := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- payload:
				default:
					delete(h.clients, client)
					close(client.send)
				}
			}
		}
	}
}

func (h *webSocketHub) publish(topic string, payload []byte) { //广播
	h.start()

	select {
	case h.broadcast <- payload: //发送
	default:
		setCTCPServerLastMessage("WebSocket broadcast queue full, dropped topic=%s", topic)
	}
}

func (c *webSocketClient) readPump() { //读取前端发送的数据
	defer func() {
		c.hub.unregister <- c
		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(webSocketMaxMessageSize)                  //设置读取最大值
	_ = c.conn.SetReadDeadline(time.Now().Add(webSocketPongWait)) //判断是否超时
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(webSocketPongWait))
	})

	for {
		messageType, payload, err := c.conn.ReadMessage() //读取数据
		if err != nil {
			return
		}
		if messageType != websocket.TextMessage { //判断一下payload 的类型 看一下是不是文本类型
			continue
		}
		c.handleIncoming(payload)
	}
}

func (c *webSocketClient) writePump() { // 主动推消息给前端
	ticker := time.NewTicker(webSocketPingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case payload, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(webSocketWriteWait)) //超时时间
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{}) //发送一个关闭信号
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, payload); err != nil { //发送数据
				return
			}
		case <-ticker.C: //定时器到了
			_ = c.conn.SetWriteDeadline(time.Now().Add(webSocketWriteWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *webSocketClient) handleIncoming(payload []byte) { //处理前端发送的数据s
	text := strings.TrimSpace(string(payload))
	if strings.EqualFold(text, "ping") { //如果前端发了个ping 就回复个pong 这个是心跳机制的一部分
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

	case "close_client":
		fmt.Println("123")

	case "dropdata":
		c.handleDropData(control)

	case "saveLevelData":
		c.handleGradeInfoData("saveLevelData", cTCPHCGradeInfo, control)
	case "saveQualityData":
		c.handleGradeInfoData("saveQualityData", cTCPHCColorGradeInfo, control)

	}

}

func parseWebSocketControlMessage(text string) (webSocketControlMessage, bool) {
	if !json.Valid([]byte(text)) {
		return webSocketControlMessage{}, false
	}

	var message webSocketControlMessage
	if err := json.Unmarshal([]byte(text), &message); err != nil {
		setCTCPServerLastMessage("WebSocket control JSON 解析失败: %v", err)
		return webSocketControlMessage{}, false
	}
	message.Type = strings.TrimSpace(message.Type)
	return message, message.Type != ""
}

func (c *webSocketClient) handleRequestStGlobal() {
	// 前端 WebSocket 连接成功后发 requestStGlobal，表示前端已经准备接收数据。
	// 这里异步触发 CTCP 客户端发送 DISPLAY_ON，避免阻塞 WebSocket 的读循环。
	go func() {
		if result := RequestStGlobalFromDefaultFSM(); result != 0 {
			setCTCPServerLastMessage("WebSocket requestStGlobal failed: result=%d", result)
		}
	}()
}

func (c *webSocketClient) handleDropData(control webSocketControlMessage) {
	go func() {
		result, destID, payloadBytes := DragLevelData(control)
		c.sendCommandAck("dropdata", cTCPHCGradeInfo, destID, payloadBytes, result)
	}()
}

func (c *webSocketClient) handleGradeInfoData(topic string, commandID int32, control webSocketControlMessage) {
	go func() {
		result, destID, payloadBytes := SendGradeInfoData(topic, commandID, control)
		c.sendCommandAck(topic, commandID, destID, payloadBytes, result)
	}()
}

func (c *webSocketClient) sendCommandAck(topic string, commandID int32, destID int32, payloadBytes int, result int) {
	c.sendFrame(webSocketFrame{
		Type:  "commandAck",
		Topic: topic,
		Data: rawJSONFromValue(map[string]any{
			"result":       result,
			"ok":           result == 0,
			"command":      topic,
			"cmdId":        commandID,
			"destId":       destID,
			"payloadBytes": payloadBytes,
			"message":      commandAckMessage(result),
		}),
	})
}

func commandAckMessage(result int) string {
	if result == 0 {
		return "sent"
	}
	return "send failed"
}

func (c *webSocketClient) sendFrame(frame webSocketFrame) { //发送数据
	if frame.At == 0 {
		frame.At = time.Now().UnixMilli()
	}
	payload, err := json.Marshal(frame)
	if err != nil {
		return
	}
	c.trySend(payload)
}

func (c *webSocketClient) trySend(payload []byte) bool { //尝试发送
	select {
	case c.send <- payload:
		return true
	default:
		return false
	}
}

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

func normalizeWebSocketJSON(jsonText string) (json.RawMessage, error) { //标准化JSON
	text := strings.TrimSpace(jsonText)
	if text == "" {
		return nil, errEmptyWebSocketJSON
	}

	raw := json.RawMessage(text) //raw = 初始的text数据
	if !json.Valid(raw) {
		return nil, errInvalidWebSocketJSON
	}
	return cloneRawMessage(raw), nil
}

func normalizeTopic(topic string, defaultTopic string) string { //标准化主题
	topic = strings.ToLower(strings.TrimSpace(topic))
	if topic == "" {
		return defaultTopic
	}
	return topic
}

func cloneRawMessage(raw json.RawMessage) json.RawMessage { //克隆当前数据
	if raw == nil {
		return nil
	}
	out := make(json.RawMessage, len(raw))
	copy(out, raw)
	return out
}

func rawJSONFromValue(value any) json.RawMessage { //将任意值转换为原始JSON消息
	payload, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	return json.RawMessage(payload)
}

// 处理拖拽数据下发
func DragLevelData(control webSocketControlMessage) (int, int32, int) {
	if control.Grade != nil {
		return SendGradeInfoData("dropdata", cTCPHCGradeInfo, control)
	}

	destID := normalizeDropDataDestID(control)
	if len(control.GradeExits) == 0 {
		setCTCPServerLastMessage("WebSocket dropdata failed: empty gradeExits, dest=0x%04X", uint32(destID))
		return -1, destID, 0
	}

	grade, ok := latestGradeInfo(destID)
	if !ok {
		setCTCPServerLastMessage("WebSocket dropdata failed: no cached StGradeInfo, dest=0x%04X", uint32(destID))
		return -1, destID, 0
	}

	if err := applyGradeExitMapping(&grade, control.GradeExits); err != nil {
		setCTCPServerLastMessage("WebSocket dropdata failed: %v", err)
		return -1, destID, 0
	}

	payload, err := encodeGradeInfoPayload(grade)
	if err != nil {
		setCTCPServerLastMessage("WebSocket dropdata failed: encode StGradeInfo: %v", err)
		return -1, destID, 0
	}

	targetIP, targetPort := resolveCTCPTarget(destID, cTCPHCGradeInfo, "", 0)
	setCTCPServerLastMessage(
		"WebSocket dropdata: sending HC_CMD_GRADE_INFO(0x%04X), dest=0x%04X, target=%s:%d, payload=%d bytes, activeExits=%s",
		uint32(cTCPHCGradeInfo),
		uint32(destID),
		targetIP,
		targetPort,
		len(payload),
		summarizeGradeExitMappings(grade),
	)

	result := StartCTCPClient(targetIP, targetPort, destID, cTCPHCGradeInfo, payload)
	if result != 0 {
		setCTCPServerLastMessage("WebSocket dropdata failed: HC_CMD_GRADE_INFO result=%d", result)
		return result, destID, len(payload)
	}

	cacheLatestGradeInfo(destID, grade)
	setCTCPServerLastMessage("WebSocket dropdata success: HC_CMD_GRADE_INFO sent, dest=0x%04X", uint32(destID))
	return 0, destID, len(payload)
}

func SendGradeInfoData(topic string, commandID int32, control webSocketControlMessage) (int, int32, int) {
	destID := normalizeDropDataDestID(control)
	if control.Grade == nil {
		setCTCPServerLastMessage("WebSocket %s failed: empty StGradeInfo, dest=0x%04X", topic, uint32(destID))
		return -1, destID, 0
	}

	grade := *control.Grade
	payload, err := encodeGradeInfoPayload(grade)
	if err != nil {
		setCTCPServerLastMessage("WebSocket %s failed: encode StGradeInfo: %v", topic, err)
		return -1, destID, 0
	}

	targetIP, targetPort := resolveCTCPTarget(destID, commandID, "", 0)
	setCTCPServerLastMessage(
		"WebSocket %s: sending cmd=0x%04X, dest=0x%04X, target=%s:%d, payload=%d bytes, sizeGrade=%d, qualityGrade=%d, activeExits=%s",
		topic,
		uint32(commandID),
		uint32(destID),
		targetIP,
		targetPort,
		len(payload),
		grade.NSizeGradeNum,
		grade.NQualityGradeNum,
		summarizeGradeExitMappings(grade),
	)

	result := StartCTCPClient(targetIP, targetPort, destID, commandID, payload)
	if result != 0 {
		setCTCPServerLastMessage("WebSocket %s failed: cmd=0x%04X result=%d", topic, uint32(commandID), result)
		return result, destID, len(payload)
	}

	cacheLatestGradeInfo(destID, grade)
	setCTCPServerLastMessage("WebSocket %s success: cmd=0x%04X sent, dest=0x%04X", topic, uint32(commandID), uint32(destID))
	return 0, destID, len(payload)
}

func cacheLatestGradeInfo(destID int32, grade StGradeInfo) {
	setLastStGradeInfoSnapshot(grade)
	if destID == 0 {
		return
	}
	gradeInfoCacheMu.Lock()
	gradeInfoCache[destID] = grade
	gradeInfoCacheMu.Unlock()
}

func latestGradeInfo(destID int32) (StGradeInfo, bool) {
	gradeInfoCacheMu.RLock()
	grade, ok := gradeInfoCache[destID]
	gradeInfoCacheMu.RUnlock()
	return grade, ok
}

func summarizeGradeExitMappings(grade StGradeInfo) string {
	const maxSummaryItems = 12

	parts := make([]string, 0, maxSummaryItems)
	total := 0
	for index, item := range grade.Grades {
		if item.ExitLow == 0 && item.ExitHigh == 0 {
			continue
		}
		total++
		if len(parts) < maxSummaryItems {
			parts = append(parts, fmt.Sprintf(
				"%d:low=0x%08X,high=0x%08X,exit=%d",
				index,
				item.ExitLow,
				item.ExitHigh,
				item.Exit(),
			))
		}
	}
	if total == 0 {
		return "none"
	}
	if total > len(parts) {
		return fmt.Sprintf("%s (+%d more)", strings.Join(parts, ","), total-len(parts))
	}
	return strings.Join(parts, ",")
}

func normalizeDropDataDestID(control webSocketControlMessage) int32 { //标准化目标ID
	if control.DestID != 0 {
		return control.DestID
	}
	if control.FSMID != 0 {
		return control.FSMID
	}
	return cTCPDefaultFSMID
}

func applyGradeExitMapping(grade *StGradeInfo, exits []webSocketGradeExit) error {
	for _, item := range exits {
		if item.Index < 0 || item.Index >= len(grade.Grades) {
			return fmt.Errorf("grade index out of range: %d", item.Index)
		}

		exitMask := uint64(item.Exit)
		grade.Grades[item.Index].ExitLow = uint32(exitMask)
		grade.Grades[item.Index].ExitHigh = uint32(exitMask >> 32)
	}
	return nil
}

func encodeGradeInfoPayload(grade StGradeInfo) ([]byte, error) {
	size := int(unsafe.Sizeof(grade))
	if size != cTCP48StGradeInfoWireSize {
		return nil, fmt.Errorf("sizeof(StGradeInfo)=%d, expected=%d", size, cTCP48StGradeInfoWireSize)
	}

	payload := make([]byte, size)
	src := unsafe.Slice((*byte)(unsafe.Pointer(&grade)), size)
	copy(payload, src)
	return payload, nil
}
