package tcp

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"gotest/ohos/Tcp/events"
	"gotest/ohos/database"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func init() {
	events.SetPublisherFunc(publishWebSocketJSON)
}

const (
	webSocketTopicData               = "data"     //
	webSocketTopicStGlobal           = "stglobal" // StGlobal 全量数据
	webSocketTopicStatistics         = "statistics"
	webSocketTopicHomeStats          = "homeStats"
	webSocketTopicGrade              = "grade"
	webSocketTopicExitInfos          = "exitInfos"
	webSocketTopicExitDisplay        = "exitDisplay"
	webSocketTopicExitAdditionalText = "exitAdditionalText"
	webSocketTopicLevelAuxConfig     = "levelAuxConfig"
	webSocketTopicFruitTypeConfig    = "fruitTypeConfig"
	webSocketTopicProjectSchemeFile  = "projectSettingsSchemeFile"
	webSocketTopicWeightGlobal       = "weightGlobal"
	webSocketTopicWeightInfo         = "weightInfo"
	webSocketTopicWaveInfo           = "waveInfo"

	webSocketWriteWait             = 5 * time.Second  //写入等待
	webSocketPongWait              = 70 * time.Second // Pong 等待，比客户端心跳周期略长，允许偶尔的网络抖动
	webSocketPingPeriod            = 30 * time.Second
	webSocketMaxMessageSize        = 2056 * 2056 // 4MB 放数据用的
	webSocketSendBufferSize        = 32          //最多发送32条消息
	webSocketBroadcastBufferSize   = 64          // 广播最多64条消息
	webSocketSysConfigRefreshDelay = 300 * time.Millisecond
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
	Type      string          `json:"type"`
	Topic     string          `json:"topic,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	RequestID string          `json:"requestId,omitempty"`
	At        int64           `json:"at"` //时间戳
}

type webSocketControlMessage struct {
	Type                       string                              `json:"type"`
	RequestID                  string                              `json:"requestId,omitempty"`
	FSMID                      int32                               `json:"fsmId,omitempty"`
	DestID                     int32                               `json:"destId,omitempty"`
	SysConfig                  *StSysConfig                        `json:"sysConfig,omitempty"`
	Grade                      *StGradeInfo                        `json:"grade,omitempty"`
	Paras                      *StParas                            `json:"paras,omitempty"`
	GlobalExitInfo             *StGlobalExitInfo                   `json:"globalExitInfo,omitempty"`
	ExitInfo                   *StExitInfo                         `json:"exitInfo,omitempty"`
	ExitInfos                  *webSocketExitInfosControl          `json:"exitInfos,omitempty"`
	ExitDisplay                *webSocketExitDisplayControl        `json:"exitDisplay,omitempty"`
	ExitDisplayEnabled         *bool                               `json:"exitDisplayEnabled,omitempty"`
	ExitScreenSettings         *webSocketExitScreenSettingsControl `json:"exitScreenSettings,omitempty"`
	ExitAdditionalText         *webSocketExitAdditionalTextControl `json:"exitAdditionalText,omitempty"`
	LevelAuxConfig             *webSocketLevelAuxConfigControl     `json:"levelAuxConfig,omitempty"`
	PreserveCachedGradeExits   *bool                               `json:"preserveCachedGradeExits,omitempty"`
	GradeNameTexts             *webSocketGradeNameTexts            `json:"gradeNameTexts,omitempty"`
	FruitTypeConfig            *webSocketFruitTypeConfigControl    `json:"fruitTypeConfig,omitempty"`
	ProjectScheme              *webSocketProjectSchemeControl      `json:"projectScheme,omitempty"`
	DensityInfo                *StAnalogDensity                    `json:"densityInfo,omitempty"`
	Motor                      *StMotorInfo                        `json:"motor,omitempty"`
	GradeExits                 []webSocketGradeExit                `json:"gradeExits,omitempty"`
	Action                     string                              `json:"action,omitempty"`
	ExitNo                     int                                 `json:"exitNo,omitempty"`
	DropGrades                 []webSocketDropGrade                `json:"grades,omitempty"`
	CameraNum                  *int                                `json:"cameraNum,omitempty"`
	EvenShow                   *bool                               `json:"evenShow,omitempty"`
	WhiteBalancePayload        []int                               `json:"whiteBalancePayload,omitempty"`
	IP                         string                              `json:"ip,omitempty"`
	MAC                        string                              `json:"mac,omitempty"`
	Phone                      string                              `json:"phone,omitempty"`
	ValidateCode               string                              `json:"validateCode,omitempty"`
	ChannelIndex               *int                                `json:"channelIndex,omitempty"`
	ExitIndex                  *int                                `json:"exitIndex,omitempty"`
	ResetADValue               *int                                `json:"resetADValue,omitempty"`
	WeightInfo                 *StWeightBaseInfo                   `json:"weightInfo,omitempty"`
	GlobalWeightInfo           *StGlobalWeightBaseInfo             `json:"globalWeightInfo,omitempty"`
	FruitCustomerInfo          *webSocketFruitCustomerInfoControl  `json:"fruitCustomerInfo,omitempty"`
	FruitInfoQuery             *database.FruitInfoQueryRequest     `json:"fruitInfoQuery,omitempty"`
	SysFruitInfoQuery          *database.SysFruitInfoQueryRequest  `json:"sysFruitInfoQuery,omitempty"`
	SortLogQuery               *database.SortLogQueryRequest       `json:"sortLogQuery,omitempty"`
	FruitInfoDeleteCustomerIDs []int                               `json:"fruitInfoDeleteCustomerIds,omitempty"`
}

type webSocketFruitCustomerInfoControl struct {
	CustomerID   int     `json:"CustomerID,omitempty"`
	CustomerName *string `json:"CustomerName,omitempty"`
	FarmName     *string `json:"FarmName,omitempty"`
	FruitName    *string `json:"FruitName,omitempty"`
	SortBaseName *string `json:"SortBaseName,omitempty"`
	ProgramName  *string `json:"ProgramName,omitempty"`
	FBatchNo     *string `json:"FBatchNo,omitempty"`
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

type webSocketExitScreenConfigControl struct {
	ExitNo         int    `json:"exitNo,omitempty"`
	CustomName     string `json:"customName,omitempty"`
	AdditionalInfo string `json:"additionalInfo,omitempty"`
	UseAutoName    *bool  `json:"useAutoName,omitempty"`
}

type webSocketExitScreenSettingsControl struct {
	Configs []webSocketExitScreenConfigControl `json:"configs,omitempty"`
}

type webSocketExitDisplayData struct {
	FSMID              int32                              `json:"fsmId"`
	Enabled            bool                               `json:"enabled"`
	ExitDisplay        ExitDisplayInfo                    `json:"exitDisplay"`
	ExitScreenSettings webSocketExitScreenSettingsControl `json:"exitScreenSettings"`
}

type webSocketExitAdditionalTextControl struct {
	AdditionalTexts []string `json:"additionalTexts,omitempty"`
}

type webSocketExitAdditionalTextData struct {
	FSMID              int32                  `json:"fsmId"`
	ExitAdditionalText ExitAdditionalTextInfo `json:"exitAdditionalText"`
}

type webSocketProjectSchemeControl struct {
	Name     string `json:"name,omitempty"`
	JSONText string `json:"jsonText,omitempty"`
}

type webSocketProjectSchemeFileData struct {
	FileName string `json:"fileName"`
	JSONText string `json:"jsonText"`
}

type webSocketLevelAuxConfigControl struct {
	SelectedFruitTypes *string   `json:"selectedFruitTypes,omitempty"`
	GradeAccuracy      *int      `json:"gradeAccuracy,omitempty"`
	ExitAlarmThreshold *int      `json:"exitAlarmThreshold,omitempty"`
	PackingWeight1     *float64  `json:"packingWeight1,omitempty"`
	PackingWeight2     *float64  `json:"packingWeight2,omitempty"`
	WeightStandards    *[]int    `json:"weightStandards,omitempty"`
	LabelerNames       *[]string `json:"labelerNames,omitempty"`
}

type webSocketLevelAuxConfigData struct {
	FSMID          int32              `json:"fsmId"`
	LevelAuxConfig LevelAuxConfigInfo `json:"levelAuxConfig"`
}

type webSocketFruitTypeConfigControl struct {
	MajorTypes         *string            `json:"majorTypes,omitempty"`
	MajorTypesEn       *string            `json:"majorTypesEn,omitempty"`
	SelectedFruitTypes *string            `json:"selectedFruitTypes,omitempty"`
	SubTypeConfigs     *map[string]string `json:"subTypeConfigs,omitempty"`
}

type webSocketFruitTypeConfigData struct {
	FSMID           int32               `json:"fsmId"`
	FruitTypeConfig FruitTypeConfigInfo `json:"fruitTypeConfig"`
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

	resetHomeStatsEfficiencyWindow("WebSocket client connected")

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
	return events.PublishJSON(topic, jsonText)
}

func publishWebSocketJSON(topic string, jsonText string) error {
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
	case "ping":
		c.sendFrame(webSocketFrame{
			Type:      "pong",
			RequestID: control.RequestID,
		})

	case "requestStGlobal":
		c.handleRequestStGlobal()

	case "close_client":

	case "dropdata":
		c.handleDropData(control)

	case "clearExitGrades", "clearAllExitGrades": //清空等级出口数据
		c.handleClearGradeExitData(control)

	case "clearData": //数据清零
		c.handleSimpleFSMCommand("clearData", cTCPHCClearData, control)
	case "saveParasToFlash":
		c.handleSimpleFSMCommand("saveParasToFlash", cTCPHCSaveParas, control)
	case "endProcess":
		c.handleEndProcess(control)
	case "updateFruitCustomerInfo":
		c.handleFruitCustomerInfoUpdate(control)
	case "queryFruitInfo":
		c.handleFruitInfoQuery(control)
	case "querySysFruitInfo":
		c.handleSysFruitInfoQuery(control)
	case "querySortLog":
		c.handleSortLogQuery(control)
	case "deleteFruitInfo":
		c.handleFruitInfoDelete(control)
	case "createProjectPassword":
		c.handleCreateProjectPassword(control)
	case "validateProjectPassword":
		c.handleValidateProjectPassword(control)
	case "fsmTestCupOn":
		c.handleSimpleFSMCommand("fsmTestCupOn", cTCPHCTestCupOn, control)
	case "fsmTestCupOff":
		c.handleSimpleFSMCommand("fsmTestCupOff", cTCPHCTestCupOff, control)
	case "fruitGradeInfoOn":
		c.handleSimpleFSMCommand("fruitGradeInfoOn", cTCPHCFruitGradeOn, control)
	case "fruitGradeInfoOff":
		c.handleSimpleFSMCommand("fruitGradeInfoOff", cTCPHCFruitGradeOff, control)
	case "motorEnable":
		c.handleSimpleFSMCommand("motorEnable", cTCPHCMotorEnable, control)
	case "exitClear":
		c.handleSimpleFSMCommand("exitClear", cTCPHCExitClear, control)
	case "testValve":
		c.handleTestValveData("testValve", cTCPHCTestVolve, control)
	case "testAllLaneValve":
		c.handleTestValveData("testAllLaneValve", cTCPHCTestAllLaneVolve, control)

	case "wamGetInfo":
		c.handleSimpleWAMCommand("wamGetInfo", cTCPHCWAMWeightOn, control)
	case "wamWeightReset":
		c.handleSimpleWAMCommand("wamWeightReset", cTCPHCWAMWeightReset, control)
	case "resetCup":
		c.handleResetCupCommand("resetCup", cTCPHCWAMWeightReset, control)
	case "cupStateReset":
		c.handleResetCupCommand("cupStateReset", cTCPHCWAMCupStateReset, control)
	case "wamSimulatedPulseOn":
		c.handleSimpleWAMCommand("wamSimulatedPulseOn", cTCPHCWAMSimulatedPulseOn, control)
	case "wamSimulatedPulseOff":
		c.handleSimpleWAMCommand("wamSimulatedPulseOff", cTCPHCWAMSimulatedPulseOff, control)
	case "wamTestCupOn":
		c.handleSimpleWAMCommand("wamTestCupOn", cTCPHCWAMTestCupOn, control)
	case "wamTestCupOff":
		c.handleSimpleWAMCommand("wamTestCupOff", cTCPHCWAMTestCupOff, control)
	case "wamWaveFormOn":
		c.handleSimpleWAMChannelCommand("wamWaveFormOn", cTCPHCWAMWaveFormOn, control)
	case "wamWaveFormOff":
		c.handleSimpleWAMChannelCommand("wamWaveFormOff", cTCPHCWAMWaveFormOff, control)
	case "wamDataTrackingOn":
		c.handleSimpleWAMChannelCommand("wamDataTrackingOn", cTCPHCWAMDataTrackingOn, control)
	case "wamDataTrackingOff":
		c.handleSimpleWAMChannelCommand("wamDataTrackingOff", cTCPHCWAMDataTrackingOff, control)
	case "wamResetAd":
		c.handleWAMResetAD(control)
	case "saveWamWeightInfo":
		c.handleWAMWeightInfoData(control)
	case "saveWamGlobalWeightInfo":
		c.handleWAMGlobalWeightInfoData(control)

	case "saveLevelData":
		c.handleGradeInfoData("saveLevelData", cTCPHCGradeInfo, control)
	case "saveQualityData":
		c.handleGradeInfoData("saveQualityData", cTCPHCColorGradeInfo, control)
	case "saveDensityInfo":
		c.handleDensityInfoData(control)
	case "saveSysConfig":
		c.handleSysConfigData(control)
	case "saveParasInfo":
		c.handleParasInfoData(control)
	case "saveGlobalExitInfo":
		c.handleGlobalExitInfoData(control)
	case "saveExitInfo":
		c.handleExitInfoData(control)
	case "ipmSingleSample":
		c.handleIpmCameraCommand("ipmSingleSample", cTCPHCSingleSample, control)
	case "ipmContinuousSampleOn":
		c.handleIpmCameraCommand("ipmContinuousSampleOn", cTCPHCContinuousSampleOn, control)
	case "ipmContinuousSampleOff":
		c.handleIpmCameraCommand("ipmContinuousSampleOff", cTCPHCContinuousSampleOff, control)
	case "ipmShowBlobOn":
		c.handleIpmCameraCommand("ipmShowBlobOn", cTCPHCShowBlobOn, control)
	case "ipmAutoBalanceOnCamera":
		c.handleIpmCameraCommand("ipmAutoBalanceOnCamera", cTCPHCAutoBalanceOnCamera, control)
	case "ipmAutoBalanceOn":
		c.handleIpmCameraCommand("ipmAutoBalanceOn", cTCPHCAutoBalanceOn, control)
	case "ipmSingleSampleSpot":
		c.handleIpmCameraCommand("ipmSingleSampleSpot", cTCPHCSingleSampleSpot, control)
	case "ipmShutdown":
		c.handleIpmCameraCommand("ipmShutdown", cTCPHCIpmShutdown, control)
	case "ipmShutterAdjustOn":
		c.handleIpmCameraCommand("ipmShutterAdjustOn", cTCPHCShutterAdjustOn, control)
	case "ipmShutterAdjustOff":
		c.handleIpmCameraCommand("ipmShutterAdjustOff", cTCPHCShutterAdjustOff, control)
	case "ipmGetMac":
		c.handleIpmGetMAC(control)
	case "ipmWakeOnLan":
		c.handleIpmWakeOnLAN(control)
	case "saveExitInfos":
		c.handleExitInfosLog(control)
	case "saveExitDisplay":
		c.handleExitDisplayInfo(control)
	case "setExitDisplayEnabled":
		c.handleExitDisplayEnabled(control)
	case "saveExitScreenSettings":
		c.handleExitScreenSettings(control)
	case "saveExitAdditionalText":
		c.handleExitAdditionalTextInfo(control)
	case "saveLevelAuxConfig":
		c.handleLevelAuxConfigInfo(control)
	case "saveFruitTypeConfig":
		c.handleFruitTypeConfigInfo(control)
	case "saveProjectSettingsScheme":
		c.handleSaveProjectSettingsScheme(control)
	case "importProjectSettingsScheme":
		c.handleImportProjectSettingsScheme(control)
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

func (c *webSocketClient) sendCommandAck(topic string, commandID int32, destID int32, payloadBytes int, result int) {
	c.sendCommandAckDetail(topic, commandID, destID, payloadBytes, result, commandAckMessage(result), "")
}

func (c *webSocketClient) sendCommandAckDetail(topic string, commandID int32, destID int32, payloadBytes int, result int, message string, requestID string) {
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
			"message":      message,
			"requestId":    requestID,
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
