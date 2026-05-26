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
	webSocketTopicData               = "data"     //
	webSocketTopicStGlobal           = "stglobal" // StGlobal 全量数据
	webSocketTopicStatistics         = "statistics"
	webSocketTopicGrade              = "grade"
	webSocketTopicExitInfos          = "exitInfos"
	webSocketTopicExitDisplay        = "exitDisplay"
	webSocketTopicExitAdditionalText = "exitAdditionalText"

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
	Type               string                              `json:"type"`
	FSMID              int32                               `json:"fsmId,omitempty"`
	DestID             int32                               `json:"destId,omitempty"`
	Grade              *StGradeInfo                        `json:"grade,omitempty"`
	ExitInfos          *webSocketExitInfosControl          `json:"exitInfos,omitempty"`
	ExitDisplay        *webSocketExitDisplayControl        `json:"exitDisplay,omitempty"`
	ExitAdditionalText *webSocketExitAdditionalTextControl `json:"exitAdditionalText,omitempty"`
	Motor              *StMotorInfo                        `json:"motor,omitempty"`
	GradeExits         []webSocketGradeExit                `json:"gradeExits,omitempty"`
	Action             string                              `json:"action,omitempty"`
	ExitNo             int                                 `json:"exitNo,omitempty"`
	DropGrades         []webSocketDropGrade                `json:"grades,omitempty"`
}

type webSocketGradeExit struct {
	Index int     `json:"index"`
	Exit  float64 `json:"exit"`
}

type webSocketDropGrade struct {
	Row   *int   `json:"row,omitempty"`
	Col   *int   `json:"col,omitempty"`
	Index *int   `json:"index,omitempty"`
	Name  string `json:"name,omitempty"`
}

type webSocketExitInfosData struct {
	FSMID     int32       `json:"fsmId"`
	ExitInfos StExitInfos `json:"exitInfos"`
}

type webSocketExitInfosControl struct {
	ExitName        []uint8 `json:"ExitName,omitempty"`
	ExitControlMode []uint8 `json:"ExitControlMode,omitempty"`
	ExitBoxType     []uint8 `json:"ExitBoxType,omitempty"`
}

type webSocketExitDisplayControl struct {
	DisplayType  *int64   `json:"displayType,omitempty"`
	DisplayNames []string `json:"displayNames,omitempty"`
}

type webSocketExitDisplayData struct {
	FSMID       int32           `json:"fsmId"`
	ExitDisplay ExitDisplayInfo `json:"exitDisplay"`
}

type webSocketExitAdditionalTextControl struct {
	AdditionalTexts []string `json:"additionalTexts,omitempty"`
}

type webSocketExitAdditionalTextData struct {
	FSMID              int32                  `json:"fsmId"`
	ExitAdditionalText ExitAdditionalTextInfo `json:"exitAdditionalText"`
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
		fmt.Println("123") //发送一个display_off 还有一个关闭啥来着 忘记了

	case "dropdata":
		c.handleDropData(control)

	case "clearExitGrades", "clearAllExitGrades": //清空等级出口数据
		c.handleClearGradeExitData(control)

	case "clearData": //数据清零
		c.handleSimpleFSMCommand("clearData", cTCPHCClearData, control)

	case "saveLevelData":
		c.handleGradeInfoData("saveLevelData", cTCPHCGradeInfo, control)
	case "saveQualityData":
		c.handleGradeInfoData("saveQualityData", cTCPHCColorGradeInfo, control)
	case "saveExitInfos":
		c.handleExitInfosLog(control)
	case "saveExitDisplay":
		c.handleExitDisplayInfo(control)
	case "saveExitAdditionalText":
		c.handleExitAdditionalTextInfo(control)
	case "saveMotorInfo":
		c.handleMotorInfoData(control)
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
	c.sendLatestExitInfosData()
	c.sendLatestExitDisplayData()
	c.sendLatestExitAdditionalTextData()
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

func (c *webSocketClient) handleClearGradeExitData(control webSocketControlMessage) {
	go func() {
		result, destID, payloadBytes := ClearGradeExitData(control)
		c.sendCommandAck("clearExitGrades", cTCPHCGradeInfo, destID, payloadBytes, result)
	}()
}

func (c *webSocketClient) handleGradeInfoData(topic string, commandID int32, control webSocketControlMessage) {
	go func() {
		result, destID, payloadBytes := SendGradeInfoData(topic, commandID, control)
		c.sendCommandAck(topic, commandID, destID, payloadBytes, result)
	}()
}

func (c *webSocketClient) handleMotorInfoData(control webSocketControlMessage) {
	go func() {
		result, destID, payloadBytes := SendMotorInfoData(control)
		c.sendCommandAck("saveMotorInfo", cTCPHCMotorInfo, destID, payloadBytes, result)
	}()
}

func (c *webSocketClient) handleExitInfosLog(control webSocketControlMessage) {
	if control.ExitInfos == nil {
		setCTCPServerLastMessage("WebSocket saveExitInfos failed: empty StExitInfos, fsmId=0x%04X", uint32(control.FSMID))
		return
	}

	baseExitInfos := defaultStExitInfos()
	if _, cachedExitInfos, ok := latestStExitInfosSnapshot(); ok {
		baseExitInfos = cachedExitInfos
	} else if storedExitInfos, ok, err := ReadStExitInfosFromLocalConfig(); err != nil {
		setCTCPServerLastMessage("WebSocket saveExitInfos failed: read local config: %v", err)
		return
	} else if ok {
		baseExitInfos = storedExitInfos
	}
	exitInfos, err := applyStExitInfosUpdate(baseExitInfos, *control.ExitInfos)
	if err != nil {
		setCTCPServerLastMessage("WebSocket saveExitInfos failed: %v", err)
		return
	}
	if err := SaveStExitInfosToLocalConfig(control.FSMID, exitInfos); err != nil {
		return
	}
	if storedExitInfos, ok, err := ReadStExitInfosFromLocalConfig(); err != nil {
		setCTCPServerLastMessage("WebSocket saveExitInfos saved but reload failed: %v", err)
	} else if ok {
		exitInfos = storedExitInfos
	}
	setLastStExitInfosSnapshot(control.FSMID, exitInfos)
	publishExitInfosData(control.FSMID, exitInfos)
}

func applyStExitInfosUpdate(base StExitInfos, update webSocketExitInfosControl) (StExitInfos, error) {
	next := base
	if update.ExitName != nil {
		if err := copyExactUint8Field("ExitName", next.ExitName[:], update.ExitName); err != nil {
			return StExitInfos{}, err
		}
	}
	if update.ExitControlMode != nil {
		if err := copyExactUint8Field("ExitControlMode", next.ExitControlMode[:], update.ExitControlMode); err != nil {
			return StExitInfos{}, err
		}
	}
	if update.ExitBoxType != nil {
		if err := copyExactUint8Field("ExitBoxType", next.ExitBoxType[:], update.ExitBoxType); err != nil {
			return StExitInfos{}, err
		}
	}
	return next, nil
}

func copyExactUint8Field(field string, dst []uint8, src []uint8) error {
	if len(src) != len(dst) {
		return fmt.Errorf("%s length=%d, want %d", field, len(src), len(dst))
	}
	copy(dst, src)
	return nil
}

func (c *webSocketClient) handleExitDisplayInfo(control webSocketControlMessage) {
	if control.ExitDisplay == nil {
		setCTCPServerLastMessage("WebSocket saveExitDisplay failed: empty ExitDisplayInfo, fsmId=0x%04X", uint32(control.FSMID))
		return
	}

	baseInfo := defaultExitDisplayInfo()
	if _, cachedInfo, ok := latestExitDisplayInfoSnapshot(); ok {
		baseInfo = cachedInfo
	}
	info := applyExitDisplayUpdate(baseInfo, *control.ExitDisplay)
	if err := SaveExitDisplayInfoToLocalConfig(control.FSMID, info); err != nil {
		return
	}
	setLastExitDisplayInfoSnapshot(control.FSMID, info)
	publishExitDisplayData(control.FSMID, info)
}

func applyExitDisplayUpdate(base ExitDisplayInfo, update webSocketExitDisplayControl) ExitDisplayInfo {
	next := base
	if update.DisplayType != nil {
		next.DisplayType = *update.DisplayType
	}
	for i, name := range update.DisplayNames {
		if i >= len(next.DisplayNames) {
			break
		}
		next.DisplayNames[i] = name
	}
	return next
}

func (c *webSocketClient) handleExitAdditionalTextInfo(control webSocketControlMessage) {
	if control.ExitAdditionalText == nil {
		setCTCPServerLastMessage("WebSocket saveExitAdditionalText failed: empty ExitAdditionalTextInfo, fsmId=0x%04X", uint32(control.FSMID))
		return
	}

	baseInfo := defaultExitAdditionalTextInfo()
	if _, cachedInfo, ok := latestExitAdditionalTextInfoSnapshot(); ok {
		baseInfo = cachedInfo
	}
	info := applyExitAdditionalTextUpdate(baseInfo, *control.ExitAdditionalText)
	if err := SaveExitAdditionalTextInfoToLocalConfig(control.FSMID, info); err != nil {
		return
	}
	setLastExitAdditionalTextInfoSnapshot(control.FSMID, info)
	publishExitAdditionalTextData(control.FSMID, info)
}

func applyExitAdditionalTextUpdate(base ExitAdditionalTextInfo, update webSocketExitAdditionalTextControl) ExitAdditionalTextInfo {
	next := base
	for i, text := range update.AdditionalTexts {
		if i >= len(next.AdditionalTexts) {
			break
		}
		next.AdditionalTexts[i] = text
	}
	return next
}

func (c *webSocketClient) sendLatestExitInfosData() {
	fsmID, exitInfos, ok := latestStExitInfosSnapshot()
	if !ok {
		storedExitInfos, storedOK, err := ReadStExitInfosFromLocalConfig()
		if err != nil {
			setCTCPServerLastMessage("WebSocket send exitInfos failed: read local config: %v", err)
			return
		}
		if !storedOK {
			return
		}
		fsmID = 0
		exitInfos = storedExitInfos
		setLastStExitInfosSnapshot(fsmID, exitInfos)
	}
	c.sendExitInfosData(fsmID, exitInfos)
}

func (c *webSocketClient) sendLatestExitDisplayData() {
	fsmID, info, ok := latestExitDisplayInfoSnapshot()
	if !ok {
		return
	}
	c.sendExitDisplayData(fsmID, info)
}

func (c *webSocketClient) sendLatestExitAdditionalTextData() {
	fsmID, info, ok := latestExitAdditionalTextInfoSnapshot()
	if !ok {
		return
	}
	c.sendExitAdditionalTextData(fsmID, info)
}

func (c *webSocketClient) sendExitInfosData(fsmID int32, exitInfos StExitInfos) {
	c.sendFrame(webSocketFrame{
		Type:  webSocketTopicData,
		Topic: webSocketTopicExitInfos,
		Data:  rawJSONFromValue(webSocketExitInfosData{FSMID: fsmID, ExitInfos: exitInfos}),
	})
}

func (c *webSocketClient) sendExitDisplayData(fsmID int32, info ExitDisplayInfo) {
	c.sendFrame(webSocketFrame{
		Type:  webSocketTopicData,
		Topic: webSocketTopicExitDisplay,
		Data:  rawJSONFromValue(webSocketExitDisplayData{FSMID: fsmID, ExitDisplay: info}),
	})
}

func (c *webSocketClient) sendExitAdditionalTextData(fsmID int32, info ExitAdditionalTextInfo) {
	c.sendFrame(webSocketFrame{
		Type:  webSocketTopicData,
		Topic: webSocketTopicExitAdditionalText,
		Data:  rawJSONFromValue(webSocketExitAdditionalTextData{FSMID: fsmID, ExitAdditionalText: info}),
	})
}

func publishExitInfosData(fsmID int32, exitInfos StExitInfos) {
	raw := rawJSONFromValue(webSocketExitInfosData{FSMID: fsmID, ExitInfos: exitInfos})
	frame, err := newWebSocketDataFrame(webSocketTopicExitInfos, raw)
	if err != nil {
		setCTCPServerLastMessage("WebSocket publish exitInfos failed: %v", err)
		return
	}
	defaultWebSocketHub.publish(webSocketTopicExitInfos, frame)
}

func publishExitDisplayData(fsmID int32, info ExitDisplayInfo) {
	raw := rawJSONFromValue(webSocketExitDisplayData{FSMID: fsmID, ExitDisplay: info})
	frame, err := newWebSocketDataFrame(webSocketTopicExitDisplay, raw)
	if err != nil {
		setCTCPServerLastMessage("WebSocket publish exitDisplay failed: %v", err)
		return
	}
	defaultWebSocketHub.publish(webSocketTopicExitDisplay, frame)
}

func publishExitAdditionalTextData(fsmID int32, info ExitAdditionalTextInfo) {
	raw := rawJSONFromValue(webSocketExitAdditionalTextData{FSMID: fsmID, ExitAdditionalText: info})
	frame, err := newWebSocketDataFrame(webSocketTopicExitAdditionalText, raw)
	if err != nil {
		setCTCPServerLastMessage("WebSocket publish exitAdditionalText failed: %v", err)
		return
	}
	defaultWebSocketHub.publish(webSocketTopicExitAdditionalText, frame)
}

func (c *webSocketClient) handleSimpleFSMCommand(topic string, commandID int32, control webSocketControlMessage) {
	go func() {
		result, destID, payloadBytes := SendSimpleFSMCommand(topic, commandID, control)
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
	if len(control.DropGrades) == 0 && len(control.GradeExits) == 0 {
		setCTCPServerLastMessage("WebSocket dropdata failed: empty gradeExits, dest=0x%04X", uint32(destID))
		return -1, destID, 0
	}

	grade, ok := latestGradeInfo(destID)
	if !ok {
		setCTCPServerLastMessage("WebSocket dropdata failed: no cached StGradeInfo, dest=0x%04X", uint32(destID))
		return -1, destID, 0
	}

	if len(control.DropGrades) > 0 {
		if err := applyGradeDropAction(&grade, control.Action, control.ExitNo, control.DropGrades); err != nil {
			setCTCPServerLastMessage("WebSocket dropdata failed: %v", err)
			return -1, destID, 0
		}
	} else if err := applyGradeExitMapping(&grade, control.GradeExits); err != nil {
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

func ClearGradeExitData(control webSocketControlMessage) (int, int32, int) {
	destID := normalizeDropDataDestID(control)
	grade, ok := latestGradeInfo(destID)
	if !ok {
		setCTCPServerLastMessage("WebSocket clearExitGrades failed: no cached StGradeInfo, dest=0x%04X", uint32(destID))
		return -1, destID, 0
	}

	before := summarizeGradeExitMappings(grade)
	clearGradeExitMappings(&grade)

	payload, err := encodeGradeInfoPayload(grade)
	if err != nil {
		setCTCPServerLastMessage("WebSocket clearExitGrades failed: encode StGradeInfo: %v", err)
		return -1, destID, 0
	}

	targetIP, targetPort := resolveCTCPTarget(destID, cTCPHCGradeInfo, "", 0)
	setCTCPServerLastMessage(
		"WebSocket clearExitGrades: sending HC_CMD_GRADE_INFO(0x%04X), dest=0x%04X, target=%s:%d, payload=%d bytes, activeExitsBefore=%s, activeExitsAfter=%s",
		uint32(cTCPHCGradeInfo),
		uint32(destID),
		targetIP,
		targetPort,
		len(payload),
		before,
		summarizeGradeExitMappings(grade),
	)

	result := StartCTCPClient(targetIP, targetPort, destID, cTCPHCGradeInfo, payload)
	if result != 0 {
		setCTCPServerLastMessage("WebSocket clearExitGrades failed: HC_CMD_GRADE_INFO result=%d", result)
		return result, destID, len(payload)
	}

	cacheLatestGradeInfo(destID, grade)
	setCTCPServerLastMessage("WebSocket clearExitGrades success: HC_CMD_GRADE_INFO sent, dest=0x%04X", uint32(destID))
	return 0, destID, len(payload)
}

func SendGradeInfoData(topic string, commandID int32, control webSocketControlMessage) (int, int32, int) {
	destID := normalizeDropDataDestID(control)
	if control.Grade == nil {
		setCTCPServerLastMessage("WebSocket %s failed: empty StGradeInfo, dest=0x%04X", topic, uint32(destID))
		return -1, destID, 0
	}

	grade := *control.Grade
	if commandID == cTCPHCGradeInfo {
		if cached, ok := latestGradeInfo(destID); ok {
			if shouldPreserveCachedGradeExits(grade, cached) {
				mergeGradeExitState(&grade, cached)
			}
		}
	}

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

func SendMotorInfoData(control webSocketControlMessage) (int, int32, int) {
	destID := normalizeDropDataDestID(control)
	if control.Motor == nil {
		setCTCPServerLastMessage("WebSocket saveMotorInfo failed: empty StMotorInfo, dest=0x%04X", uint32(destID))
		return -1, destID, 0
	}

	motor := *control.Motor
	if int(motor.BExitId) >= cTCP48MaxExitNum {
		setCTCPServerLastMessage("WebSocket saveMotorInfo failed: exit=%d out of range, dest=0x%04X", motor.BExitId, uint32(destID))
		return -1, destID, 0
	}
	payload, err := encodeMotorInfoPayload(motor)
	if err != nil {
		setCTCPServerLastMessage("WebSocket saveMotorInfo failed: encode StMotorInfo: %v", err)
		return -1, destID, 0
	}
	setLastStMotorInfoSnapshot(destID, motor)

	targetIP, targetPort := resolveCTCPTarget(destID, cTCPHCMotorInfo, "", 0)
	setCTCPServerLastMessage(
		"WebSocket saveMotorInfo: sending HC_CMD_MOTOR_INFO(0x%04X), dest=0x%04X, target=%s:%d, payload=%d bytes, exit=%d, mode=%d, num=%d, weight=%d, delay=%.1f, hold=%.1f",
		uint32(cTCPHCMotorInfo),
		uint32(destID),
		targetIP,
		targetPort,
		len(payload),
		motor.BExitId,
		motor.BMotorSwitch,
		motor.NMotorEnableSwitchNum,
		motor.NMotorEnableSwitchWeight,
		motor.FDelayTime,
		motor.FHoldTime,
	)

	result := StartCTCPClient(targetIP, targetPort, destID, cTCPHCMotorInfo, payload)
	if result != 0 {
		setCTCPServerLastMessage("WebSocket saveMotorInfo failed: HC_CMD_MOTOR_INFO result=%d", result)
		return result, destID, len(payload)
	}

	setCTCPServerLastMessage("WebSocket saveMotorInfo success: HC_CMD_MOTOR_INFO sent, dest=0x%04X, exit=%d", uint32(destID), motor.BExitId)
	return 0, destID, len(payload)
}

func SendSimpleFSMCommand(topic string, commandID int32, control webSocketControlMessage) (int, int32, int) {
	destID := normalizeDropDataDestID(control)
	targetIP, targetPort := resolveCTCPTarget(destID, commandID, "", 0)
	setCTCPServerLastMessage(
		"WebSocket %s: sending cmd=0x%04X, dest=0x%04X, target=%s:%d, payload=0 bytes",
		topic,
		uint32(commandID),
		uint32(destID),
		targetIP,
		targetPort,
	)

	result := StartCTCPClient(targetIP, targetPort, destID, commandID, nil)
	if result != 0 {
		setCTCPServerLastMessage("WebSocket %s failed: cmd=0x%04X result=%d", topic, uint32(commandID), result)
		return result, destID, 0
	}

	setCTCPServerLastMessage("WebSocket %s success: cmd=0x%04X sent, dest=0x%04X", topic, uint32(commandID), uint32(destID))
	return 0, destID, 0
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

func applyGradeDropAction(grade *StGradeInfo, action string, exitNo int, grades []webSocketDropGrade) error {
	add, err := normalizeDropAction(action)
	if err != nil {
		return err
	}
	if exitNo < 1 || exitNo > 64 {
		return fmt.Errorf("exitNo out of range: %d", exitNo)
	}
	if len(grades) == 0 {
		return errors.New("empty drop grades")
	}

	for _, item := range grades {
		index, err := resolveDropGradeIndex(item)
		if err != nil {
			return err
		}
		applyGradeExitBit(&grade.Grades[index], exitNo, add)
	}
	if add {
		enableGradeExit(grade, exitNo)
	}
	return nil
}

func normalizeDropAction(action string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "", "add", "append", "up":
		return true, nil
	case "remove", "delete", "del", "down":
		return false, nil
	default:
		return false, fmt.Errorf("unknown drop action: %s", action)
	}
}

func resolveDropGradeIndex(item webSocketDropGrade) (int, error) {
	if item.Row != nil || item.Col != nil {
		if item.Row == nil || item.Col == nil {
			return 0, errors.New("grade row/col must be provided together")
		}
		if *item.Row < 0 || *item.Row >= cTCPServerMaxQualityGradeNum {
			return 0, fmt.Errorf("grade row out of range: %d", *item.Row)
		}
		if *item.Col < 0 || *item.Col >= cTCPServerMaxSizeGradeNum {
			return 0, fmt.Errorf("grade col out of range: %d", *item.Col)
		}
		return *item.Row*cTCPServerMaxSizeGradeNum + *item.Col, nil
	}
	if item.Index != nil {
		if *item.Index < 0 || *item.Index >= cTCPServerMaxQualityGradeNum*cTCPServerMaxSizeGradeNum {
			return 0, fmt.Errorf("grade index out of range: %d", *item.Index)
		}
		return *item.Index, nil
	}
	return 0, errors.New("grade row/col or index is required")
}

func applyGradeExitBit(item *StGradeItemInfo, exitNo int, add bool) {
	mask := uint64(1) << uint(exitNo-1)
	exit64 := uint64(item.ExitLow) | (uint64(item.ExitHigh) << 32)
	if add {
		exit64 |= mask
	} else {
		exit64 &^= mask
	}
	item.ExitLow = uint32(exit64)
	item.ExitHigh = uint32(exit64 >> 32)
}

func clearGradeExitMappings(grade *StGradeInfo) {
	if grade == nil {
		return
	}
	for i := range grade.Grades {
		grade.Grades[i].ExitLow = 0
		grade.Grades[i].ExitHigh = 0
	}
}

func mergeGradeExitState(target *StGradeInfo, cached StGradeInfo) {
	for i := 0; i < len(target.Grades) && i < len(cached.Grades); i++ {
		target.Grades[i].ExitLow |= cached.Grades[i].ExitLow
		target.Grades[i].ExitHigh |= cached.Grades[i].ExitHigh
	}
	for i := 0; i < len(target.ExitEnabled) && i < len(cached.ExitEnabled); i++ {
		target.ExitEnabled[i] = int32(uint32(target.ExitEnabled[i]) | uint32(cached.ExitEnabled[i]))
	}
}

func shouldPreserveCachedGradeExits(target StGradeInfo, cached StGradeInfo) bool {
	return target.NSizeGradeNum == cached.NSizeGradeNum &&
		target.NQualityGradeNum == cached.NQualityGradeNum &&
		target.NClassifyType == cached.NClassifyType
}

func enableGradeExit(grade *StGradeInfo, exitNo int) {
	if exitNo < 1 || exitNo > 64 {
		return
	}
	bucket := 0
	shift := exitNo - 1
	if shift >= 32 {
		bucket = 1
		shift -= 32
	}
	grade.ExitEnabled[bucket] = int32(uint32(grade.ExitEnabled[bucket]) | (uint32(1) << uint(shift)))
}

func applyGradeExitMapping(grade *StGradeInfo, exits []webSocketGradeExit) error {
	for _, item := range exits {
		if item.Index < 0 || item.Index >= len(grade.Grades) {
			return fmt.Errorf("grade index out of range: %d", item.Index)
		}

		exitMask := uint64(item.Exit)
		current := grade.Grades[item.Index].Exit()
		next := current | exitMask
		grade.Grades[item.Index].ExitLow = uint32(next)
		grade.Grades[item.Index].ExitHigh = uint32(next >> 32)
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

func encodeMotorInfoPayload(motor StMotorInfo) ([]byte, error) {
	size := int(unsafe.Sizeof(motor))
	if size != cTCP48StMotorInfoWireSize {
		return nil, fmt.Errorf("sizeof(StMotorInfo)=%d, expected=%d", size, cTCP48StMotorInfoWireSize)
	}

	payload := make([]byte, size)
	src := unsafe.Slice((*byte)(unsafe.Pointer(&motor)), size)
	copy(payload, src)
	return payload, nil
}
