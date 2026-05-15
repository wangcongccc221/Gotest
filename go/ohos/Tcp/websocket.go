package tcp

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

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
	Type string `json:"type"`
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
	}
}

func parseWebSocketControlMessage(text string) (webSocketControlMessage, bool) {
	if !json.Valid([]byte(text)) {
		return webSocketControlMessage{}, false
	}

	var message webSocketControlMessage
	if err := json.Unmarshal([]byte(text), &message); err != nil {
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
