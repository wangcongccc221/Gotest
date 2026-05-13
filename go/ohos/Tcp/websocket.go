package tcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	webSocketTopicAll        = "all"      // 订阅时表示订阅所有主题；发布时不使用，默认主题为 "data"
	webSocketTopicData       = "data"     // 默认主题，表示一般数据更新；也可用于订阅/发布时的显式主题
	webSocketTopicStGlobal   = "stglobal" // StGlobal 全量数据；发布时表示发送 StGlobal 全量 JSON；订阅时表示订阅 StGlobal 全量 JSON 更新
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

type webSocketRequest struct {
	Type      string          `json:"type"`
	RequestID string          `json:"requestId,omitempty"`
	Topic     string          `json:"topic,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

type webSocketFrame struct { //数据帧
	Type      string          `json:"type"`
	RequestID string          `json:"requestId,omitempty"`
	Topic     string          `json:"topic,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Message   string          `json:"message,omitempty"`
	Error     string          `json:"error,omitempty"`
	At        int64           `json:"at"` //时间戳
}

type webSocketBroadcast struct {
	topic   string
	payload []byte
}

type webSocketHub struct {
	once       sync.Once
	register   chan *webSocketClient
	unregister chan *webSocketClient
	broadcast  chan webSocketBroadcast
	clients    map[*webSocketClient]struct{}

	snapshotMu sync.RWMutex
	snapshots  map[string]json.RawMessage
}

type webSocketClient struct {
	hub  *webSocketHub
	conn *websocket.Conn
	send chan []byte

	subMu         sync.RWMutex
	subscriptions map[string]struct{}
}

func newWebSocketHub() *webSocketHub { //管理中心
	return &webSocketHub{
		register:   make(chan *webSocketClient),
		unregister: make(chan *webSocketClient),
		broadcast:  make(chan webSocketBroadcast, webSocketBroadcastBufferSize),
		clients:    make(map[*webSocketClient]struct{}),
		snapshots:  make(map[string]json.RawMessage),
	}
}

func registerWebSocketRoutes(router *gin.Engine) { //注册路由
	router.GET("/ws", handleWebSocket)
	router.GET("/ws/data", handleWebSocketData)
}

func handleWebSocketEcho(ctx *gin.Context) { //回显测试
	handleWebSocket(ctx)
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
		subscriptions: map[string]struct{}{
			webSocketTopicAll: {},
		},
	}

	client.hub.register <- client
	go client.writePump()

	client.sendFrame(webSocketFrame{
		Type:    "ready",
		Message: "websocket connected",
		Data: rawJSONFromValue(map[string]any{
			"topics":          client.hub.topicList(),
			"maxMessageBytes": webSocketMaxMessageSize,
		}),
	})
	client.readPump()
}

func PublishWebSocketJSON(topic string, jsonText string) error {
	topic = normalizeWebSocketTopic(topic)
	raw, err := normalizeWebSocketJSON(jsonText)
	if err != nil {
		return err
	}

	frame, err := newWebSocketDataFrame(topic, raw)
	if err != nil {
		return err
	}

	defaultWebSocketHub.publish(topic, raw, frame)
	return nil
}

func (h *webSocketHub) start() {
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
		case message := <-h.broadcast:
			for client := range h.clients {
				if !client.accepts(message.topic) {
					continue
				}
				select {
				case client.send <- message.payload:
				default:
					delete(h.clients, client)
					close(client.send)
				}
			}
		}
	}
}

func (h *webSocketHub) publish(topic string, data json.RawMessage, payload []byte) { //广播
	h.start()
	h.setSnapshot(topic, data) //保存当前数据

	select {
	case h.broadcast <- webSocketBroadcast{topic: topic, payload: payload}: //发送
	default:
		setCTCPServerLastMessage("WebSocket broadcast queue full, dropped topic=%s", topic)
	}
}

func (h *webSocketHub) setSnapshot(topic string, data json.RawMessage) { //保存转换好的结构体字符串  SAVE
	h.snapshotMu.Lock()
	h.snapshots[topic] = cloneRawMessage(data)
	h.snapshotMu.Unlock()
}

func (h *webSocketHub) snapshot(topic string) (json.RawMessage, bool) { //读取当前数据 READ
	h.snapshotMu.RLock()
	data, ok := h.snapshots[topic]
	h.snapshotMu.RUnlock()
	if !ok {
		return nil, false
	}
	return cloneRawMessage(data), true
}

func (h *webSocketHub) topicList() []string { //列举所有主题主要用于调试
	h.snapshotMu.RLock()
	topics := make([]string, 0, len(h.snapshots))
	for topic := range h.snapshots {
		topics = append(topics, topic)
	}
	h.snapshotMu.RUnlock()
	sort.Strings(topics)
	return topics
}

func (h *webSocketHub) allSnapshots() map[string]json.RawMessage { //读取所有快照 用于测试
	h.snapshotMu.RLock()
	out := make(map[string]json.RawMessage, len(h.snapshots))
	for topic, data := range h.snapshots {
		out[topic] = cloneRawMessage(data)
	}
	h.snapshotMu.RUnlock()
	return out
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

	/*
		messageType: 这个payload的类型
	*/
	for {
		messageType, payload, err := c.conn.ReadMessage() //读取数据
		if err != nil {
			return
		}
		if messageType != websocket.TextMessage { //判断一下payload 的类型 看一下是不是文本类型
			c.sendError("", "", "only text websocket messages are supported")
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
	text := strings.TrimSpace(string(payload)) // 去掉首位空白 可注释 用不到这个
	if text == "" {
		c.sendError("", "", "empty websocket message")
		return
	}
	if strings.EqualFold(text, "ping") { //如果前端发了个ping 就回复个pong 这个是心跳机制的一部分
		c.sendFrame(webSocketFrame{Type: "pong"})
		return
	}
	if !json.Valid([]byte(text)) { //如果不是合法的json 就发个错误回去
		c.trySend([]byte("发送失败非法JSON: " + text))
		return
	}

	var request webSocketRequest
	if err := json.Unmarshal([]byte(text), &request); err != nil {
		c.sendError("", "", fmt.Sprintf("invalid websocket request: %v", err))
		return
	}

	requestType := strings.ToLower(strings.TrimSpace(request.Type))
	switch requestType {
	case "":
		c.sendFrame(webSocketFrame{
			Type:      "echo",
			RequestID: request.RequestID,
			Data:      cloneRawMessage(json.RawMessage(text)),
		})
	case "echo":
		c.sendFrame(webSocketFrame{
			Type:      "echo",
			RequestID: request.RequestID,
			Topic:     normalizeOptionalWebSocketTopic(request.Topic),
			Payload:   cloneRawMessage(request.Payload),
		})
	case "ping":
		c.sendFrame(webSocketFrame{
			Type:      "pong",
			RequestID: request.RequestID,
		})
	case "status":
		c.sendStatus(request.RequestID)
	case "snapshot":
		c.sendSnapshot(request.RequestID, request.Topic)
	case "subscribe":
		topic := c.subscribe(request.Topic)
		c.sendFrame(webSocketFrame{
			Type:      "subscribed",
			RequestID: request.RequestID,
			Topic:     topic,
		})
	case "unsubscribe":
		topic := c.unsubscribe(request.Topic)
		c.sendFrame(webSocketFrame{
			Type:      "unsubscribed",
			RequestID: request.RequestID,
			Topic:     topic,
		})
	default:
		c.sendError(request.RequestID, request.Topic, fmt.Sprintf("unsupported websocket message type %q", request.Type))
	}
}

func (c *webSocketClient) sendStatus(requestID string) {
	c.sendFrame(webSocketFrame{
		Type:      "status",
		RequestID: requestID,
		Data: rawJSONFromValue(map[string]any{
			"topics":          c.hub.topicList(),
			"maxMessageBytes": webSocketMaxMessageSize,
		}),
	})
}

func (c *webSocketClient) sendSnapshot(requestID string, topic string) {
	if strings.TrimSpace(topic) == "" || strings.EqualFold(strings.TrimSpace(topic), webSocketTopicAll) {
		snapshots := c.hub.allSnapshots()
		if len(snapshots) == 0 {
			c.sendError(requestID, webSocketTopicAll, "no websocket snapshot available")
			return
		}

		topics := make([]string, 0, len(snapshots))
		for topic := range snapshots {
			topics = append(topics, topic)
		}
		sort.Strings(topics)
		for _, topic := range topics {
			c.sendFrame(webSocketFrame{
				Type:      "data",
				RequestID: requestID,
				Topic:     topic,
				Data:      snapshots[topic],
			})
		}
		return
	}

	topic = normalizeWebSocketTopic(topic)
	data, ok := c.hub.snapshot(topic)
	if !ok {
		c.sendError(requestID, topic, "no websocket snapshot available")
		return
	}
	c.sendFrame(webSocketFrame{
		Type:      "data",
		RequestID: requestID,
		Topic:     topic,
		Data:      data,
	})
}

func (c *webSocketClient) sendFrame(frame webSocketFrame) {
	if frame.At == 0 {
		frame.At = time.Now().UnixMilli()
	}
	payload, err := json.Marshal(frame)
	if err != nil {
		return
	}
	c.trySend(payload)
}

func (c *webSocketClient) sendError(requestID string, topic string, message string) {
	c.sendFrame(webSocketFrame{
		Type:      "error",
		RequestID: requestID,
		Topic:     normalizeOptionalWebSocketTopic(topic),
		Error:     message,
	})
}

func (c *webSocketClient) trySend(payload []byte) bool {
	select {
	case c.send <- payload:
		return true
	default:
		return false
	}
}

func (c *webSocketClient) accepts(topic string) bool {
	c.subMu.RLock()
	_, all := c.subscriptions[webSocketTopicAll]
	_, exact := c.subscriptions[topic]
	c.subMu.RUnlock()
	return all || exact
}

func (c *webSocketClient) subscribe(topic string) string {
	topic = normalizeSubscriptionTopic(topic)
	c.subMu.Lock()
	if topic == webSocketTopicAll {
		c.subscriptions = map[string]struct{}{webSocketTopicAll: {}}
	} else {
		delete(c.subscriptions, webSocketTopicAll)
		c.subscriptions[topic] = struct{}{}
	}
	c.subMu.Unlock()
	return topic
}

func (c *webSocketClient) unsubscribe(topic string) string {
	topic = normalizeSubscriptionTopic(topic)
	c.subMu.Lock()
	if topic == webSocketTopicAll {
		c.subscriptions = map[string]struct{}{webSocketTopicAll: {}}
	} else {
		delete(c.subscriptions, topic)
		if len(c.subscriptions) == 0 {
			c.subscriptions[webSocketTopicAll] = struct{}{}
		}
	}
	c.subMu.Unlock()
	return topic
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

func normalizeWebSocketJSON(jsonText string) (json.RawMessage, error) {
	text := strings.TrimSpace(jsonText)
	if text == "" {
		return nil, errEmptyWebSocketJSON
	}

	raw := json.RawMessage(text)
	if !json.Valid(raw) {
		return nil, errInvalidWebSocketJSON
	}
	return cloneRawMessage(raw), nil
}

func normalizeWebSocketTopic(topic string) string {
	topic = strings.ToLower(strings.TrimSpace(topic))
	if topic == "" {
		return webSocketTopicData
	}
	return topic
}

func normalizeOptionalWebSocketTopic(topic string) string {
	topic = strings.ToLower(strings.TrimSpace(topic))
	return topic
}

func normalizeSubscriptionTopic(topic string) string {
	topic = strings.ToLower(strings.TrimSpace(topic))
	if topic == "" {
		return webSocketTopicAll
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

func rawJSONFromValue(value any) json.RawMessage {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	return json.RawMessage(payload)
}
