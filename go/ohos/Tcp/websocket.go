package tcp

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
	"unsafe"

	"gotest/ohos/database"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

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
	ExitAdditionalText         *webSocketExitAdditionalTextControl `json:"exitAdditionalText,omitempty"`
	LevelAuxConfig             *webSocketLevelAuxConfigControl     `json:"levelAuxConfig,omitempty"`
	PreserveCachedGradeExits   *bool                               `json:"preserveCachedGradeExits,omitempty"`
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
	ChannelIndex               *int                                `json:"channelIndex,omitempty"`
	ResetADValue               *int                                `json:"resetADValue,omitempty"`
	WeightInfo                 *StWeightBaseInfo                   `json:"weightInfo,omitempty"`
	GlobalWeightInfo           *StGlobalWeightBaseInfo             `json:"globalWeightInfo,omitempty"`
	FruitCustomerInfo          *webSocketFruitCustomerInfoControl  `json:"fruitCustomerInfo,omitempty"`
	FruitInfoQuery             *database.FruitInfoQueryRequest     `json:"fruitInfoQuery,omitempty"`
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
		fmt.Println("123") //发送一个display_off 还有一个关闭啥来着 忘记了

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
	case "deleteFruitInfo":
		c.handleFruitInfoDelete(control)
	case "fsmTestCupOn":
		c.handleSimpleFSMCommand("fsmTestCupOn", cTCPHCTestCupOn, control)
	case "fsmTestCupOff":
		c.handleSimpleFSMCommand("fsmTestCupOff", cTCPHCTestCupOff, control)
	case "fruitGradeInfoOn":
		c.handleSimpleFSMCommand("fruitGradeInfoOn", cTCPHCFruitGradeOn, control)
	case "fruitGradeInfoOff":
		c.handleSimpleFSMCommand("fruitGradeInfoOff", cTCPHCFruitGradeOff, control)

	case "wamGetInfo":
		c.handleSimpleWAMCommand("wamGetInfo", cTCPHCWAMWeightOn, control)
	case "wamWeightReset":
		c.handleSimpleWAMCommand("wamWeightReset", cTCPHCWAMWeightReset, control)
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
	case "ipmShutterAdjustOn":
		c.handleIpmCameraCommand("ipmShutterAdjustOn", cTCPHCShutterAdjustOn, control)
	case "ipmShutterAdjustOff":
		c.handleIpmCameraCommand("ipmShutterAdjustOff", cTCPHCShutterAdjustOff, control)
	case "saveExitInfos":
		c.handleExitInfosLog(control)
	case "saveExitDisplay":
		c.handleExitDisplayInfo(control)
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

func (c *webSocketClient) handleRequestStGlobal() {
	c.sendLatestExitInfosData()
	c.sendLatestExitDisplayData()
	c.sendLatestExitAdditionalTextData()
	c.sendLatestLevelAuxConfigData()
	c.sendLatestFruitTypeConfigData()
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
		// 透传 requestId：前端依赖它匹配 ack，确认等级配置是否真正转发到下位机
		c.sendCommandAckDetail(topic, commandID, destID, payloadBytes, result, commandAckMessage(result), control.RequestID)
	}()
}

func (c *webSocketClient) handleDensityInfoData(control webSocketControlMessage) {
	go func() {
		result, destID, payloadBytes := SendDensityInfoData(control)
		c.sendCommandAck("saveDensityInfo", cTCPHCDensityInfo, destID, payloadBytes, result)
	}()
}

func (c *webSocketClient) handleMotorInfoData(control webSocketControlMessage) {
	go func() {
		result, destID, payloadBytes := SendMotorInfoData(control)
		c.sendCommandAck("saveMotorInfo", cTCPHCMotorInfo, destID, payloadBytes, result)
	}()
}

func (c *webSocketClient) handleSysConfigData(control webSocketControlMessage) {
	go func() {
		result, destID, payloadBytes := SendSysConfigData(control)
		c.sendCommandAck("saveSysConfig", cTCPHCSysConfig, destID, payloadBytes, result)
	}()
}

func (c *webSocketClient) handleParasInfoData(control webSocketControlMessage) {
	go func() {
		result, destID, payloadBytes := SendParasInfoData(control)
		c.sendCommandAck("saveParasInfo", cTCPHCParasInfo, destID, payloadBytes, result)
	}()
}

func (c *webSocketClient) handleGlobalExitInfoData(control webSocketControlMessage) {
	go func() {
		result, destID, payloadBytes := SendGlobalExitInfoData(control)
		c.sendCommandAck("saveGlobalExitInfo", cTCPHCGlobalExitInfo, destID, payloadBytes, result)
	}()
}

func (c *webSocketClient) handleExitInfoData(control webSocketControlMessage) {
	go func() {
		result, destID, payloadBytes := SendExitInfoData(control)
		c.sendCommandAck("saveExitInfo", cTCPHCExitInfo, destID, payloadBytes, result)
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

func (c *webSocketClient) handleLevelAuxConfigInfo(control webSocketControlMessage) {
	if control.LevelAuxConfig == nil {
		setCTCPServerLastMessage("WebSocket saveLevelAuxConfig failed: empty levelAuxConfig, fsmId=0x%04X", uint32(control.FSMID))
		return
	}
	if err := saveLevelAuxConfigFromControl(control.FSMID, *control.LevelAuxConfig); err != nil {
		return
	}
}

func (c *webSocketClient) handleFruitTypeConfigInfo(control webSocketControlMessage) {
	if control.FruitTypeConfig == nil {
		setCTCPServerLastMessage("WebSocket saveFruitTypeConfig failed: empty fruitTypeConfig, fsmId=0x%04X", uint32(control.FSMID))
		return
	}
	if err := saveFruitTypeConfigFromControl(control.FSMID, *control.FruitTypeConfig); err != nil {
		return
	}
}

func (c *webSocketClient) handleSaveProjectSettingsScheme(control webSocketControlMessage) {
	if control.ProjectScheme == nil {
		setCTCPServerLastMessage("WebSocket saveProjectSettingsScheme failed: empty projectScheme, fsmId=0x%04X", uint32(control.FSMID))
		c.sendCommandAck("saveProjectSettingsScheme", 0, control.FSMID, 0, -1)
		return
	}
	scheme, err := BuildCurrentProjectSettingsSchemeForLocalFile(control.FSMID, control.ProjectScheme.Name)
	if err != nil {
		setCTCPServerLastMessage("WebSocket saveProjectSettingsScheme failed: %v", err)
		c.sendCommandAck("saveProjectSettingsScheme", 0, control.FSMID, 0, -1)
		return
	}
	c.sendProjectSettingsSchemeFileData(scheme)
	c.sendCommandAck("saveProjectSettingsScheme", 0, control.FSMID, len(scheme.Name), 0)
}

func (c *webSocketClient) handleImportProjectSettingsScheme(control webSocketControlMessage) {
	if control.ProjectScheme == nil {
		setCTCPServerLastMessage("WebSocket importProjectSettingsScheme failed: empty projectScheme, fsmId=0x%04X", uint32(control.FSMID))
		c.sendCommandAck("importProjectSettingsScheme", 0, control.FSMID, 0, -1)
		return
	}
	scheme, err := ApplyProjectSettingsSchemeJSON(control.FSMID, control.ProjectScheme.JSONText)
	if err != nil {
		setCTCPServerLastMessage("WebSocket importProjectSettingsScheme failed: %v", err)
		c.sendCommandAck("importProjectSettingsScheme", 0, control.FSMID, len(control.ProjectScheme.JSONText), -1)
		return
	}
	c.sendCommandAck("importProjectSettingsScheme", 0, control.FSMID, len(scheme.Name), 0)
}

func saveFruitTypeConfigFromControl(fsmID int32, update webSocketFruitTypeConfigControl) error {
	baseInfo := defaultFruitTypeConfigInfo()
	if storedInfo, err := ReadFruitTypeConfigInfoFromLocalConfig(); err != nil {
		setCTCPServerLastMessage("WebSocket saveFruitTypeConfig failed: read local config: %v", err)
		return err
	} else {
		baseInfo = storedInfo
	}

	info := applyFruitTypeConfigUpdate(baseInfo, update)
	if err := SaveFruitTypeConfigInfoToLocalConfig(fsmID, info); err != nil {
		return err
	}
	setLastFruitTypeConfigInfoSnapshot(fsmID, info)
	publishFruitTypeConfigData(fsmID, info)
	syncLevelAuxSelectedFruitTypesFromFruitConfig(fsmID, info.SelectedFruitTypes)
	return nil
}

func applyFruitTypeConfigUpdate(base FruitTypeConfigInfo, update webSocketFruitTypeConfigControl) FruitTypeConfigInfo {
	next := base
	if update.MajorTypes != nil {
		next.MajorTypes = *update.MajorTypes
	}
	if update.MajorTypesEn != nil {
		next.MajorTypesEn = *update.MajorTypesEn
	}
	if update.SelectedFruitTypes != nil {
		next.SelectedFruitTypes = *update.SelectedFruitTypes
	}
	if update.SubTypeConfigs != nil {
		next.SubTypeConfigs = make(map[string]string, len(*update.SubTypeConfigs))
		for key, value := range *update.SubTypeConfigs {
			next.SubTypeConfigs[key] = value
		}
	}
	return normalizeFruitTypeConfigInfo(next)
}

func syncLevelAuxSelectedFruitTypesFromFruitConfig(fsmID int32, selectedFruitTypes string) {
	selectedFruitTypes = strings.TrimSpace(selectedFruitTypes)
	if selectedFruitTypes == "" {
		return
	}
	baseInfo := defaultLevelAuxConfigInfo()
	if storedInfo, err := ReadLevelAuxConfigInfoFromLocalConfig(); err != nil {
		setCTCPServerLastMessage("WebSocket saveFruitTypeConfig: levelAux read failed, skip selected fruit push: %v", err)
		return
	} else {
		baseInfo = storedInfo
	}
	update := webSocketLevelAuxConfigControl{SelectedFruitTypes: &selectedFruitTypes}
	info := applyLevelAuxConfigUpdate(baseInfo, update)
	if err := SaveLevelAuxConfigInfoToLocalConfig(fsmID, info); err != nil {
		return
	}
	setLastLevelAuxConfigInfoSnapshot(fsmID, info)
	publishLevelAuxConfigData(fsmID, info)
}

func saveLevelAuxConfigFromControl(fsmID int32, update webSocketLevelAuxConfigControl) error {
	baseInfo := defaultLevelAuxConfigInfo()
	if storedInfo, err := ReadLevelAuxConfigInfoFromLocalConfig(); err != nil {
		setCTCPServerLastMessage("WebSocket saveLevelAuxConfig failed: read local config: %v", err)
		return err
	} else {
		baseInfo = storedInfo
	}
	if _, cachedInfo, ok := latestLevelAuxConfigInfoSnapshot(); ok && !hasNonEmptyString(baseInfo.LabelerNames) {
		baseInfo.LabelerNames = cachedInfo.LabelerNames
	}

	info := applyLevelAuxConfigUpdate(baseInfo, update)
	if err := SaveLevelAuxConfigInfoToLocalConfig(fsmID, info); err != nil {
		return err
	}
	setLastLevelAuxConfigInfoSnapshot(fsmID, info)
	publishLevelAuxConfigData(fsmID, info)
	return nil
}

func applyLevelAuxConfigUpdate(base LevelAuxConfigInfo, update webSocketLevelAuxConfigControl) LevelAuxConfigInfo {
	next := base
	if update.SelectedFruitTypes != nil {
		next.SelectedFruitTypes = *update.SelectedFruitTypes
	}
	if update.GradeAccuracy != nil {
		next.GradeAccuracy = *update.GradeAccuracy
	}
	if update.ExitAlarmThreshold != nil {
		next.ExitAlarmThreshold = *update.ExitAlarmThreshold
	}
	if update.PackingWeight1 != nil {
		next.PackingWeight1 = *update.PackingWeight1
	}
	if update.PackingWeight2 != nil {
		next.PackingWeight2 = *update.PackingWeight2
	}
	if update.WeightStandards != nil {
		next.WeightStandards = append([]int(nil), (*update.WeightStandards)...)
	}
	if update.LabelerNames != nil {
		names := normalizeLabelerNames(*update.LabelerNames)
		if hasNonEmptyString(names) {
			next.LabelerNames = names
		}
	}
	return normalizeLevelAuxConfigInfo(next)
}

func hasNonEmptyString(values []string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
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

func (c *webSocketClient) sendLatestLevelAuxConfigData() {
	fsmID := int32(0)
	if cachedFSMID, _, ok := latestLevelAuxConfigInfoSnapshot(); ok {
		fsmID = cachedFSMID
	}
	info, err := ReadLevelAuxConfigInfoFromLocalConfig()
	if err != nil {
		if _, cachedInfo, ok := latestLevelAuxConfigInfoSnapshot(); ok {
			setCTCPServerLastMessage("WebSocket send levelAuxConfig read local config failed, fallback cache: %v", err)
			c.sendLevelAuxConfigData(fsmID, cachedInfo)
			return
		}
		setCTCPServerLastMessage("WebSocket send levelAuxConfig failed: read local config: %v", err)
		return
	}
	setLastLevelAuxConfigInfoSnapshot(fsmID, info)
	setCTCPServerLastMessage(
		"WebSocket send levelAuxConfig: fsmId=0x%04X, selectedFruitTypesLen=%d, labelerNames=%s",
		uint32(fsmID),
		len(info.SelectedFruitTypes),
		formatStringListForLog(info.LabelerNames),
	)
	c.sendLevelAuxConfigData(fsmID, info)
}

func (c *webSocketClient) sendLatestFruitTypeConfigData() {
	fsmID := int32(0)
	if cachedFSMID, _, ok := latestFruitTypeConfigInfoSnapshot(); ok {
		fsmID = cachedFSMID
	}
	info, err := ReadFruitTypeConfigInfoFromLocalConfig()
	if err != nil {
		if _, cachedInfo, ok := latestFruitTypeConfigInfoSnapshot(); ok {
			setCTCPServerLastMessage("WebSocket send fruitTypeConfig read local config failed, fallback cache: %v", err)
			c.sendFruitTypeConfigData(fsmID, cachedInfo)
			return
		}
		setCTCPServerLastMessage("WebSocket send fruitTypeConfig failed: read local config: %v", err)
		return
	}
	setLastFruitTypeConfigInfoSnapshot(fsmID, info)
	setCTCPServerLastMessage(
		"WebSocket send fruitTypeConfig: fsmId=0x%04X, majorTypes=%d, selectedFruitTypesLen=%d, subTypeKeys=%d",
		uint32(fsmID),
		len(splitSemicolonConfig(info.MajorTypes)),
		len(info.SelectedFruitTypes),
		len(info.SubTypeConfigs),
	)
	c.sendFruitTypeConfigData(fsmID, info)
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

func (c *webSocketClient) sendLevelAuxConfigData(fsmID int32, info LevelAuxConfigInfo) {
	c.sendFrame(webSocketFrame{
		Type:  webSocketTopicData,
		Topic: webSocketTopicLevelAuxConfig,
		Data:  rawJSONFromValue(webSocketLevelAuxConfigData{FSMID: fsmID, LevelAuxConfig: normalizeLevelAuxConfigInfo(info)}),
	})
}

func (c *webSocketClient) sendFruitTypeConfigData(fsmID int32, info FruitTypeConfigInfo) {
	c.sendFrame(webSocketFrame{
		Type:  webSocketTopicData,
		Topic: webSocketTopicFruitTypeConfig,
		Data:  rawJSONFromValue(webSocketFruitTypeConfigData{FSMID: fsmID, FruitTypeConfig: normalizeFruitTypeConfigInfo(info)}),
	})
}

func (c *webSocketClient) sendProjectSettingsSchemeFileData(scheme ProjectSettingsScheme) {
	jsonText, err := formatProjectSettingsSchemeJSON(scheme)
	if err != nil {
		setCTCPServerLastMessage("WebSocket send projectSettingsSchemeFile failed: %v", err)
		return
	}
	c.sendFrame(webSocketFrame{
		Type:  webSocketTopicData,
		Topic: webSocketTopicProjectSchemeFile,
		Data: rawJSONFromValue(webSocketProjectSchemeFileData{
			FileName: projectSettingsSchemeFileName(scheme),
			JSONText: jsonText,
		}),
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

func publishLevelAuxConfigData(fsmID int32, info LevelAuxConfigInfo) {
	raw := rawJSONFromValue(webSocketLevelAuxConfigData{FSMID: fsmID, LevelAuxConfig: normalizeLevelAuxConfigInfo(info)})
	frame, err := newWebSocketDataFrame(webSocketTopicLevelAuxConfig, raw)
	if err != nil {
		setCTCPServerLastMessage("WebSocket publish levelAuxConfig failed: %v", err)
		return
	}
	defaultWebSocketHub.publish(webSocketTopicLevelAuxConfig, frame)
}

func publishFruitTypeConfigData(fsmID int32, info FruitTypeConfigInfo) {
	raw := rawJSONFromValue(webSocketFruitTypeConfigData{FSMID: fsmID, FruitTypeConfig: normalizeFruitTypeConfigInfo(info)})
	frame, err := newWebSocketDataFrame(webSocketTopicFruitTypeConfig, raw)
	if err != nil {
		setCTCPServerLastMessage("WebSocket publish fruitTypeConfig failed: %v", err)
		return
	}
	defaultWebSocketHub.publish(webSocketTopicFruitTypeConfig, frame)
}

func (c *webSocketClient) handleFruitCustomerInfoUpdate(control webSocketControlMessage) {
	go func() {
		info := control.FruitCustomerInfo
		if info == nil {
			c.sendCommandAckDetail("updateFruitCustomerInfo", 0, 0, 0, -1, "fruitCustomerInfo is required", control.RequestID)
			return
		}
		err := database.UpdateFruitCustomerInfo(database.UpdateFruitCustomerInfoInput{
			CustomerID:   info.CustomerID,
			CustomerName: info.CustomerName,
			FarmName:     info.FarmName,
			FruitName:    info.FruitName,
			SortBaseName: info.SortBaseName,
			ProgramName:  info.ProgramName,
			FBatchNo:     info.FBatchNo,
		})
		if err != nil {
			setCTCPServerLastMessage("WebSocket updateFruitCustomerInfo failed: CustomerID=%d, err=%v", info.CustomerID, err)
			c.sendCommandAckDetail("updateFruitCustomerInfo", 0, int32(info.CustomerID), 0, -1, err.Error(), control.RequestID)
			return
		}
		snapshot, readErr := database.GetFruitCustomerInfoSnapshot(info.CustomerID)
		if readErr != nil {
			setCTCPServerLastMessage(
				"WebSocket updateFruitCustomerInfo success but readback failed: CustomerID=%d, database=%s, err=%v",
				info.CustomerID,
				database.RealtimeSaveDatabaseForLog(),
				readErr,
			)
			c.sendCommandAckDetail("updateFruitCustomerInfo", 0, int32(info.CustomerID), 0, -1, readErr.Error(), control.RequestID)
			return
		}
		setCTCPServerLastMessage(
			"WebSocket updateFruitCustomerInfo success: CustomerID=%d, database=%s, readback CustomerName=%s, FarmName=%s, FruitName=%s",
			snapshot.CustomerID,
			database.RealtimeSaveDatabaseForLog(),
			snapshot.CustomerName,
			snapshot.FarmName,
			snapshot.FruitName,
		)
		c.sendCommandAckDetail("updateFruitCustomerInfo", 0, int32(info.CustomerID), 0, 0, "database updated and verified", control.RequestID)
	}()
}

func (c *webSocketClient) handleFruitInfoQuery(control webSocketControlMessage) {
	go func() {
		query := control.FruitInfoQuery
		if query == nil {
			c.sendCommandAckDetail("queryFruitInfo", 0, 0, 0, -1, "fruitInfoQuery is required", control.RequestID)
			return
		}

		result, err := database.QueryFruitInfoPage(*query)
		payloadBytes := len(rawJSONFromValue(*query))
		if err != nil {
			setCTCPServerLastMessage("WebSocket queryFruitInfo failed: err=%v", err)
			c.sendCommandAckDetail("queryFruitInfo", 0, 0, payloadBytes, -1, err.Error(), control.RequestID)
			return
		}

		setCTCPServerLastMessage(
			"WebSocket queryFruitInfo success: rows=%d, total=%d, page=%d, size=%d, database=%s",
			len(result.Items),
			result.TotalCount,
			result.PageIndex,
			result.PageSize,
			database.RealtimeSaveDatabaseForLog(),
		)
		c.sendFrame(webSocketFrame{
			Type:  "commandAck",
			Topic: "queryFruitInfo",
			Data: rawJSONFromValue(map[string]any{
				"result":       0,
				"ok":           true,
				"command":      "queryFruitInfo",
				"cmdId":        0,
				"destId":       0,
				"payloadBytes": payloadBytes,
				"message":      "database queried",
				"requestId":    control.RequestID,
				"Items":        result.Items,
				"TotalCount":   result.TotalCount,
				"PageIndex":    result.PageIndex,
				"PageSize":     result.PageSize,
			}),
		})
	}()
}

func (c *webSocketClient) handleFruitInfoDelete(control webSocketControlMessage) {
	go func() {
		customerIDs := control.FruitInfoDeleteCustomerIDs
		payloadBytes := len(rawJSONFromValue(customerIDs))
		deletedRows, err := database.DeleteFruitInfoByCustomerIDs(customerIDs)
		if err != nil {
			setCTCPServerLastMessage("WebSocket deleteFruitInfo failed: customerIds=%v, err=%v", customerIDs, err)
			c.sendCommandAckDetail("deleteFruitInfo", 0, 0, payloadBytes, -1, err.Error(), control.RequestID)
			return
		}

		setCTCPServerLastMessage(
			"WebSocket deleteFruitInfo success: customerIds=%v, deletedRows=%d, database=%s",
			customerIDs,
			deletedRows,
			database.RealtimeSaveDatabaseForLog(),
		)
		c.sendFrame(webSocketFrame{
			Type:  "commandAck",
			Topic: "deleteFruitInfo",
			Data: rawJSONFromValue(map[string]any{
				"result":       0,
				"ok":           true,
				"command":      "deleteFruitInfo",
				"cmdId":        0,
				"destId":       0,
				"payloadBytes": payloadBytes,
				"message":      "database soft deleted",
				"requestId":    control.RequestID,
				"customerIds":  customerIDs,
				"deletedRows":  deletedRows,
			}),
		})
	}()
}

func (c *webSocketClient) handleSimpleFSMCommand(topic string, commandID int32, control webSocketControlMessage) {
	go func() {
		result, destID, payloadBytes := SendSimpleFSMCommand(topic, commandID, control)
		c.sendCommandAckDetail(
			topic,
			commandID,
			destID,
			payloadBytes,
			result,
			commandAckMessage(result),
			control.RequestID,
		)
	}()
}

func (c *webSocketClient) handleIpmCameraCommand(topic string, commandID int32, control webSocketControlMessage) {
	go func() {
		result, destID, payloadBytes := SendIpmCameraCommand(topic, commandID, control)
		c.sendCommandAck(topic, commandID, destID, payloadBytes, result)
	}()
}

func (c *webSocketClient) handleSimpleWAMCommand(topic string, commandID int32, control webSocketControlMessage) {
	result, destID, payloadBytes := SendSimpleWAMCommand(topic, commandID, control)
	c.sendCommandAck(topic, commandID, destID, payloadBytes, result)
}

func (c *webSocketClient) handleSimpleWAMChannelCommand(topic string, commandID int32, control webSocketControlMessage) {
	result, destID, payloadBytes := SendSimpleWAMChannelCommand(topic, commandID, control)
	c.sendCommandAck(topic, commandID, destID, payloadBytes, result)
}

func (c *webSocketClient) handleWAMResetAD(control webSocketControlMessage) {
	result, destID, payloadBytes := SendWAMResetAD(control)
	c.sendCommandAck("wamResetAd", cTCPHCWAMResetAD, destID, payloadBytes, result)
}

func (c *webSocketClient) handleWAMWeightInfoData(control webSocketControlMessage) {
	destID := normalizeWAMChannelDestID(control)
	channelIndexText := "<nil>"
	if control.ChannelIndex != nil {
		channelIndexText = fmt.Sprintf("%d", *control.ChannelIndex)
	}
	if control.WeightInfo == nil {
		setCTCPServerLastMessage("WebSocket 收到 saveWamWeightInfo: dest=0x%04X, channelIndex=%s, payload=<empty>", uint32(destID), channelIndexText)
	} else {
		weightInfo := *control.WeightInfo
		setCTCPServerLastMessage(
			"WebSocket 收到 saveWamWeightInfo: dest=0x%04X, channelIndex=%s, StWeightBaseInfo{FGADParam=[%.6f %.6f], FTemperatureParams=%.6f, WaveInterval=[%d %d]}",
			uint32(destID),
			channelIndexText,
			weightInfo.FGADParam[0],
			weightInfo.FGADParam[1],
			weightInfo.FTemperatureParams,
			weightInfo.WaveInterval[0],
			weightInfo.WaveInterval[1],
		)
	}
	result, destID, payloadBytes := SendWAMWeightInfoData(control)
	c.sendCommandAck("saveWamWeightInfo", cTCPHCWAMWeightInfo, destID, payloadBytes, result)
}

func (c *webSocketClient) handleWAMGlobalWeightInfoData(control webSocketControlMessage) {
	destID := normalizeWAMDestID(control)
	if control.GlobalWeightInfo == nil {
		setCTCPServerLastMessage("WebSocket 收到 saveWamGlobalWeightInfo: dest=0x%04X, payload=<empty>", uint32(destID))
	} else {
		globalWeightInfo := *control.GlobalWeightInfo
		setCTCPServerLastMessage(
			"WebSocket 收到 saveWamGlobalWeightInfo: dest=0x%04X, StGlobalWeightBaseInfo{FFilterParam=%.6f, AD_Filter_ALG=%d, NEffectCupThreshold=%d, NMinGradeThreshold=%d, NCupDeviationThreshold=%d, NCupBreakageThreshold=%d, NBaseCupNum=%d, NTotalCupNums=%v, RefWeight=%d, WeightTh=%d}",
			uint32(destID),
			globalWeightInfo.FFilterParam,
			globalWeightInfo.AD_Filter_ALG,
			globalWeightInfo.NEffectCupThreshold,
			globalWeightInfo.NMinGradeThreshold,
			globalWeightInfo.NCupDeviationThreshold,
			globalWeightInfo.NCupBreakageThreshold,
			globalWeightInfo.NBaseCupNum,
			globalWeightInfo.NTotalCupNums,
			globalWeightInfo.RefWeight,
			globalWeightInfo.WeightTh,
		)
	}
	setCTCPServerLastMessage("WebSocket saveWamGlobalWeightInfo: wait 1000ms before WAM global write")
	time.Sleep(1000 * time.Millisecond)
	result, destID, payloadBytes := SendWAMGlobalWeightInfoData(control)
	c.sendCommandAck("saveWamGlobalWeightInfo", cTCPHCWAMGlobalWeightInfo, destID, payloadBytes, result)
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
	setCTCPServerLastMessage(
		"WebSocket dropdata exitLOW/exitHight after apply: dest=0x%04X, action=%s, exitNo=%d, dropGrades=%d, gradeExits=%d, exitBits=%s",
		uint32(destID),
		control.Action,
		control.ExitNo,
		len(control.DropGrades),
		len(control.GradeExits),
		summarizeGradeExitLowHigh(grade),
	)

	payload, err := encodeGradeInfoPayload(grade)
	if err != nil {
		setCTCPServerLastMessage("WebSocket dropdata failed: encode StGradeInfo: %v", err)
		return -1, destID, 0
	}

	targetIP, targetPort := resolveCTCPTarget(destID, cTCPHCGradeInfo, "", 0)
	setCTCPServerLastMessage(
		"WebSocket dropdata: sending HC_CMD_GRADE_INFO(0x%04X), dest=0x%04X, target=%s:%d, payload=%d bytes, activeExits=%s, byExit=%s",
		uint32(cTCPHCGradeInfo),
		uint32(destID),
		targetIP,
		targetPort,
		len(payload),
		summarizeGradeExitMappings(grade),
		summarizeGradeExitMappingsByExit(grade),
	)

	result := StartCTCPClient(targetIP, targetPort, destID, cTCPHCGradeInfo, payload)
	if result != 0 {
		setCTCPServerLastMessage("WebSocket dropdata failed: HC_CMD_GRADE_INFO result=%d", result)
		return result, destID, len(payload)
	}

	cacheLatestGradeInfo(destID, grade)
	requestStGlobalAfterConfigCommand("dropdata", destID)
	setCTCPServerLastMessage("WebSocket dropdata success: HC_CMD_GRADE_INFO sent, dest=0x%04X, refresh StGlobal scheduled", uint32(destID))
	return 0, destID, len(payload)
}

func ClearGradeExitData(control webSocketControlMessage) (int, int32, int) { //清空数据
	destID := normalizeDropDataDestID(control)
	grade, ok := latestGradeInfo(destID)
	if !ok {
		setCTCPServerLastMessage("WebSocket clearExitGrades failed: no cached StGradeInfo, dest=0x%04X", uint32(destID))
		return -1, destID, 0
	}

	before := summarizeGradeExitMappings(grade)
	beforeByExit := summarizeGradeExitMappingsByExit(grade)
	beforeLowHigh := summarizeGradeExitLowHigh(grade)
	clearGradeExitMappings(&grade)
	setCTCPServerLastMessage(
		"WebSocket clearExitGrades exitLOW/exitHight: dest=0x%04X, before=%s, after=%s",
		uint32(destID),
		beforeLowHigh,
		summarizeGradeExitLowHigh(grade),
	)

	payload, err := encodeGradeInfoPayload(grade)
	if err != nil {
		setCTCPServerLastMessage("WebSocket clearExitGrades failed: encode StGradeInfo: %v", err)
		return -1, destID, 0
	}

	targetIP, targetPort := resolveCTCPTarget(destID, cTCPHCGradeInfo, "", 0)
	setCTCPServerLastMessage(
		"WebSocket clearExitGrades: sending HC_CMD_GRADE_INFO(0x%04X), dest=0x%04X, target=%s:%d, payload=%d bytes, activeExitsBefore=%s, byExitBefore=%s, activeExitsAfter=%s, byExitAfter=%s",
		uint32(cTCPHCGradeInfo),
		uint32(destID),
		targetIP,
		targetPort,
		len(payload),
		before,
		beforeByExit,
		summarizeGradeExitMappings(grade),
		summarizeGradeExitMappingsByExit(grade),
	)

	result := StartCTCPClient(targetIP, targetPort, destID, cTCPHCGradeInfo, payload)
	if result != 0 {
		setCTCPServerLastMessage("WebSocket clearExitGrades failed: HC_CMD_GRADE_INFO result=%d", result)
		return result, destID, len(payload)
	}

	cacheLatestGradeInfo(destID, grade)
	requestStGlobalAfterConfigCommand("clearExitGrades", destID)
	setCTCPServerLastMessage("WebSocket clearExitGrades success: HC_CMD_GRADE_INFO sent, dest=0x%04X, refresh StGlobal scheduled", uint32(destID))
	return 0, destID, len(payload)
}

func SendGradeInfoData(topic string, commandID int32, control webSocketControlMessage) (int, int32, int) { //发送等级信息数据
	destID := normalizeDropDataDestID(control)
	if commandID == cTCPHCGradeInfo && control.LevelAuxConfig != nil {
		if err := saveLevelAuxConfigFromControl(control.FSMID, *control.LevelAuxConfig); err != nil {
			setCTCPServerLastMessage("WebSocket %s failed: save levelAuxConfig: %v", topic, err)
			return -1, destID, 0
		}
	}
	if control.Grade == nil {
		setCTCPServerLastMessage("WebSocket %s failed: empty StGradeInfo, dest=0x%04X", topic, uint32(destID))
		return -1, destID, 0
	}

	grade := *control.Grade
	setCTCPServerLastMessage(
		"WebSocket %s received StGradeInfo: dest=0x%04X, sizeGrade=%d, qualityGrade=%d, activeExits=%s, byExit=%s",
		topic,
		uint32(destID),
		grade.NSizeGradeNum,
		grade.NQualityGradeNum,
		summarizeGradeExitMappings(grade),
		summarizeGradeExitMappingsByExit(grade),
	)
	setCTCPServerLastMessage(
		"WebSocket %s received exitLOW/exitHight: dest=0x%04X, sizeGrade=%d, qualityGrade=%d, exitBits=%s",
		topic,
		uint32(destID),
		grade.NSizeGradeNum,
		grade.NQualityGradeNum,
		summarizeGradeExitLowHigh(grade),
	)
	if commandID == cTCPHCGradeInfo {
		if cached, ok := latestGradeInfo(destID); ok {
			if shouldPreserveCachedGradeExits(grade, cached, control.PreserveCachedGradeExits) {
				mergeGradeExitState(&grade, cached)
				setCTCPServerLastMessage(
					"WebSocket %s merged cached exit state: dest=0x%04X, activeExits=%s, byExit=%s",
					topic,
					uint32(destID),
					summarizeGradeExitMappings(grade),
					summarizeGradeExitMappingsByExit(grade),
				)
				setCTCPServerLastMessage(
					"WebSocket %s merged cached exitLOW/exitHight: dest=0x%04X, exitBits=%s",
					topic,
					uint32(destID),
					summarizeGradeExitLowHigh(grade),
				)
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
		"WebSocket %s: sending cmd=0x%04X, dest=0x%04X, target=%s:%d, payload=%d bytes, sizeGrade=%d, qualityGrade=%d, activeExits=%s, byExit=%s",
		topic,
		uint32(commandID),
		uint32(destID),
		targetIP,
		targetPort,
		len(payload),
		grade.NSizeGradeNum,
		grade.NQualityGradeNum,
		summarizeGradeExitMappings(grade),
		summarizeGradeExitMappingsByExit(grade),
	)

	result := StartCTCPClient(targetIP, targetPort, destID, commandID, payload)
	if result != 0 {
		setCTCPServerLastMessage("WebSocket %s failed: cmd=0x%04X result=%d", topic, uint32(commandID), result)
		return result, destID, len(payload)
	}

	cacheLatestGradeInfo(destID, grade)
	requestStGlobalAfterConfigCommand(topic, destID)
	setCTCPServerLastMessage("WebSocket %s success: cmd=0x%04X sent, dest=0x%04X, refresh StGlobal scheduled", topic, uint32(commandID), uint32(destID))
	return 0, destID, len(payload)
}

func SendParasInfoData(control webSocketControlMessage) (int, int32, int) {
	destID := normalizeDropDataDestID(control)
	if control.Paras == nil {
		setCTCPServerLastMessage("WebSocket saveParasInfo failed: empty StParas, dest=0x%04X", uint32(destID))
		return -1, destID, 0
	}

	paras := *control.Paras
	payload, err := encodeParasInfoPayload(paras)
	if err != nil {
		setCTCPServerLastMessage("WebSocket saveParasInfo failed: encode StParas: %v", err)
		return -1, destID, 0
	}

	targetIP, targetPort := resolveCTCPTarget(destID, cTCPHCParasInfo, "", 0)
	setCTCPServerLastMessage(
		"WebSocket saveParasInfo: sending HC_CMD_PARAS_INFO(0x%04X), dest=0x%04X, target=%s:%d, payload=%d bytes, cupNum=%d, colorCameras=%d, nirCameras=%d",
		uint32(cTCPHCParasInfo),
		uint32(destID),
		targetIP,
		targetPort,
		len(payload),
		paras.NCupNum,
		len(paras.CameraParas),
		len(paras.IRCameraParas),
	)

	result := StartCTCPClient(targetIP, targetPort, destID, cTCPHCParasInfo, payload)
	if result != 0 {
		setCTCPServerLastMessage("WebSocket saveParasInfo failed: HC_CMD_PARAS_INFO result=%d", result)
		return result, destID, len(payload)
	}

	requestStGlobalAfterParasInfo(destID)
	setCTCPServerLastMessage("WebSocket saveParasInfo success: HC_CMD_PARAS_INFO sent, dest=0x%04X, refresh StGlobal scheduled", uint32(destID))
	return 0, destID, len(payload)
}

func SendGlobalExitInfoData(control webSocketControlMessage) (int, int32, int) {
	destID := normalizeDropDataDestID(control)
	if control.GlobalExitInfo == nil {
		setCTCPServerLastMessage("WebSocket saveGlobalExitInfo failed: empty StGlobalExitInfo, dest=0x%04X", uint32(destID))
		return -1, destID, 0
	}

	globalExitInfo := *control.GlobalExitInfo
	payload, err := encodeGlobalExitInfoPayload(globalExitInfo)
	if err != nil {
		setCTCPServerLastMessage("WebSocket saveGlobalExitInfo failed: encode StGlobalExitInfo: %v", err)
		return -1, destID, 0
	}

	targetIP, targetPort := resolveCTCPTarget(destID, cTCPHCGlobalExitInfo, "", 0)
	setCTCPServerLastMessage(
		"WebSocket saveGlobalExitInfo: sending HC_CMD_GLOBAL_EXIT_INFO(0x%04X), dest=0x%04X, target=%s:%d, payload=%d bytes, pulse=%d, labelPulse=%d, driver0=%d",
		uint32(cTCPHCGlobalExitInfo),
		uint32(destID),
		targetIP,
		targetPort,
		len(payload),
		globalExitInfo.NPulse,
		globalExitInfo.NLabelPulse,
		globalExitInfo.NDriverPin[0],
	)
	setCTCPServerLastMessage("WebSocket saveGlobalExitInfo payload全字段: %s", formatStGlobalExitInfoFields(globalExitInfo))

	result := StartCTCPClient(targetIP, targetPort, destID, cTCPHCGlobalExitInfo, payload)
	if result != 0 {
		setCTCPServerLastMessage("WebSocket saveGlobalExitInfo failed: HC_CMD_GLOBAL_EXIT_INFO result=%d", result)
		return result, destID, len(payload)
	}

	requestStGlobalAfterConfigCommand("saveGlobalExitInfo", destID)
	setCTCPServerLastMessage("WebSocket saveGlobalExitInfo success: HC_CMD_GLOBAL_EXIT_INFO sent, dest=0x%04X, refresh StGlobal scheduled", uint32(destID))
	return 0, destID, len(payload)
}

func SendExitInfoData(control webSocketControlMessage) (int, int32, int) {
	destID := normalizeDropDataDestID(control)
	if control.ExitInfo == nil {
		setCTCPServerLastMessage("WebSocket saveExitInfo failed: empty StExitInfo, dest=0x%04X", uint32(destID))
		return -1, destID, 0
	}

	exitInfo := *control.ExitInfo
	payload, err := encodeExitInfoPayload(exitInfo)
	if err != nil {
		setCTCPServerLastMessage("WebSocket saveExitInfo failed: encode StExitInfo: %v", err)
		return -1, destID, 0
	}

	targetIP, targetPort := resolveCTCPTarget(destID, cTCPHCExitInfo, "", 0)
	setCTCPServerLastMessage(
		"WebSocket saveExitInfo: sending HC_CMD_EXIT_INFO(0x%04X), dest=0x%04X, target=%s:%d, payload=%d bytes, label0Dis=%d, exit0Dis=%d, exit0Offset=%d",
		uint32(cTCPHCExitInfo),
		uint32(destID),
		targetIP,
		targetPort,
		len(payload),
		exitInfo.LabelExit[0].NDis,
		exitInfo.Exits[0].NDis,
		exitInfo.Exits[0].NOffset,
	)

	result := StartCTCPClient(targetIP, targetPort, destID, cTCPHCExitInfo, payload)
	if result != 0 {
		setCTCPServerLastMessage("WebSocket saveExitInfo failed: HC_CMD_EXIT_INFO result=%d", result)
		return result, destID, len(payload)
	}

	requestStGlobalAfterConfigCommand("saveExitInfo", destID)
	setCTCPServerLastMessage("WebSocket saveExitInfo success: HC_CMD_EXIT_INFO sent, dest=0x%04X, refresh StGlobal scheduled", uint32(destID))
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

func SendIpmCameraCommand(topic string, commandID int32, control webSocketControlMessage) (int, int32, int) {
	destID := normalizeDropDataDestID(control)
	payload, cameraNum, err := encodeIpmCameraCommandPayload(commandID, control)
	if err != nil {
		setCTCPServerLastMessage("WebSocket %s failed: %v, dest=0x%04X", topic, err, uint32(destID))
		return -1, destID, 0
	}

	targetIP, targetPort := resolveCTCPTarget(destID, commandID, "", 0)
	setCTCPServerLastMessage(
		"WebSocket %s: sending cmd=0x%04X, dest=0x%04X, target=%s:%d, payload=%d bytes, camera=%d",
		topic,
		uint32(commandID),
		uint32(destID),
		targetIP,
		targetPort,
		len(payload),
		cameraNum,
	)

	result := StartCTCPClient(targetIP, targetPort, destID, commandID, payload)
	if result != 0 {
		setCTCPServerLastMessage("WebSocket %s failed: cmd=0x%04X result=%d", topic, uint32(commandID), result)
		return result, destID, len(payload)
	}

	setCTCPServerLastMessage("WebSocket %s success: cmd=0x%04X sent, dest=0x%04X, camera=%d", topic, uint32(commandID), uint32(destID), cameraNum)
	return 0, destID, len(payload)
}

func encodeIpmCameraCommandPayload(commandID int32, control webSocketControlMessage) ([]byte, int, error) {
	switch commandID {
	case cTCPHCSingleSample, cTCPHCSingleSampleSpot, cTCPHCShutterAdjustOn, cTCPHCShutterAdjustOff:
		return nil, -1, nil
	case cTCPHCAutoBalanceOnCamera, cTCPHCAutoBalanceOn:
		payload, err := encodeIpmWhiteBalancePayload(control.WhiteBalancePayload)
		if err != nil {
			return nil, -1, err
		}
		return payload, int(payload[20]), nil
	case cTCPHCContinuousSampleOn:
		cameraNum, err := requireIpmCameraNum(control)
		if err != nil {
			return nil, cameraNum, err
		}
		payload := []byte{byte(cameraNum), 0}
		if control.EvenShow != nil && *control.EvenShow {
			payload[1] = 1
		}
		return payload, cameraNum, nil
	case cTCPHCContinuousSampleOff:
		cameraNum, err := requireIpmCameraNum(control)
		if err != nil {
			return nil, cameraNum, err
		}
		return []byte{byte(cameraNum)}, cameraNum, nil
	case cTCPHCShowBlobOn:
		cameraNum, err := requireIpmCameraNum(control)
		if err != nil {
			return nil, cameraNum, err
		}
		payload := make([]byte, 4)
		binary.LittleEndian.PutUint32(payload, uint32(cameraNum))
		return payload, cameraNum, nil
	default:
		return nil, -1, fmt.Errorf("unsupported IPM camera command: 0x%04X", uint32(commandID))
	}
}

func requireIpmCameraNum(control webSocketControlMessage) (int, error) {
	if control.CameraNum == nil {
		return 0, errors.New("empty cameraNum")
	}
	cameraNum := *control.CameraNum
	if cameraNum < 0 || cameraNum >= cTCPServerMaxCameraNum {
		return cameraNum, fmt.Errorf("cameraNum out of range: %d", cameraNum)
	}
	return cameraNum, nil
}

func encodeIpmWhiteBalancePayload(values []int) ([]byte, error) {
	const whiteBalancePayloadBytes = 24
	if len(values) != whiteBalancePayloadBytes {
		return nil, fmt.Errorf("whiteBalancePayload length=%d, want %d", len(values), whiteBalancePayloadBytes)
	}
	payload := make([]byte, whiteBalancePayloadBytes)
	for i, value := range values {
		if value < 0 || value > 255 {
			return nil, fmt.Errorf("whiteBalancePayload[%d] out of byte range: %d", i, value)
		}
		payload[i] = byte(value)
	}
	return payload, nil
}

func SendDensityInfoData(control webSocketControlMessage) (int, int32, int) {
	destID := normalizeDropDataDestID(control)
	if control.DensityInfo == nil {
		setCTCPServerLastMessage("WebSocket saveDensityInfo failed: empty StAnalogDensity, dest=0x%04X", uint32(destID))
		return -1, destID, 0
	}

	densityInfo := *control.DensityInfo
	payload, err := encodeAnalogDensityPayload(densityInfo)
	if err != nil {
		setCTCPServerLastMessage("WebSocket saveDensityInfo failed: encode StAnalogDensity: %v", err)
		return -1, destID, 0
	}

	targetIP, targetPort := resolveCTCPTarget(destID, cTCPHCDensityInfo, "", 0)
	setCTCPServerLastMessage(
		"WebSocket saveDensityInfo: sending HC_CMD_DENSITY_INFO(0x%04X), dest=0x%04X, target=%s:%d, payload=%d bytes, density0=%.3f, density1=%.3f, density2=%.3f",
		uint32(cTCPHCDensityInfo),
		uint32(destID),
		targetIP,
		targetPort,
		len(payload),
		densityInfo.UAnalogDensity[0],
		densityInfo.UAnalogDensity[1],
		densityInfo.UAnalogDensity[2],
	)

	result := StartCTCPClient(targetIP, targetPort, destID, cTCPHCDensityInfo, payload)
	if result != 0 {
		setCTCPServerLastMessage("WebSocket saveDensityInfo failed: HC_CMD_DENSITY_INFO result=%d", result)
		return result, destID, len(payload)
	}

	requestStGlobalAfterConfigCommand("saveDensityInfo", destID)
	setCTCPServerLastMessage("WebSocket saveDensityInfo success: HC_CMD_DENSITY_INFO sent, dest=0x%04X, refresh StGlobal scheduled", uint32(destID))
	return 0, destID, len(payload)
}

func SendSysConfigData(control webSocketControlMessage) (int, int32, int) { // 发送系统数据
	destID := normalizeDropDataDestID(control)
	if control.SysConfig == nil {
		setCTCPServerLastMessage("WebSocket saveSysConfig failed: empty StSysConfig, dest=0x%04X", uint32(destID))
		return -1, destID, 0
	}

	sysConfig := *control.SysConfig
	payload, err := encodeSysConfigPayload(sysConfig)
	if err != nil {
		setCTCPServerLastMessage("WebSocket saveSysConfig failed: encode StSysConfig: %v", err)
		return -1, destID, 0
	}

	targetIP, targetPort := resolveCTCPTarget(destID, cTCPHCSysConfig, "", 0)
	setCTCPServerLastMessage(
		"WebSocket saveSysConfig: sending HC_CMD_SYS_CONFIG(0x%04X), dest=0x%04X, target=%s:%d, payload=%d bytes, subsys=%d, exits=%d, systemInfo=0x%04X low9=%09b, classify=0x%02X, cir=0x%02X, uv=0x%02X, weight=0x%02X, internal=0x%02X, ultrasonic=0x%02X",
		uint32(cTCPHCSysConfig),
		uint32(destID),
		targetIP,
		targetPort,
		len(payload),
		sysConfig.NSubsysNum,
		sysConfig.NExitNum,
		uint16(sysConfig.NSystemInfo),
		uint16(sysConfig.NSystemInfo&0x01FF),
		sysConfig.NClassificationInfo,
		sysConfig.CIRClassifyType,
		sysConfig.UVClassifyType,
		sysConfig.WeightClassifyTpye,
		sysConfig.InternalClassifyType,
		sysConfig.UltrasonicClassifyType,
	)

	result := StartCTCPClient(targetIP, targetPort, destID, cTCPHCSysConfig, payload)
	if result != 0 {
		setCTCPServerLastMessage("WebSocket saveSysConfig failed: HC_CMD_SYS_CONFIG result=%d", result)
		return result, destID, len(payload)
	}

	requestStGlobalAfterSysConfig(destID)
	setCTCPServerLastMessage("WebSocket saveSysConfig success: HC_CMD_SYS_CONFIG sent, dest=0x%04X, refresh StGlobal scheduled", uint32(destID))
	return 0, destID, len(payload)
}

func requestStGlobalAfterSysConfig(destID int32) {
	go func() {
		time.Sleep(webSocketSysConfigRefreshDelay)
		if result := RequestStGlobalFromFSM(destID); result != 0 {
			setCTCPServerLastMessage("WebSocket saveSysConfig refresh StGlobal failed: dest=0x%04X, result=%d", uint32(destID), result)
			return
		}
		setCTCPServerLastMessage("WebSocket saveSysConfig refresh StGlobal requested: dest=0x%04X", uint32(destID))
	}()
}

func requestStGlobalAfterParasInfo(destID int32) {
	go func() {
		time.Sleep(webSocketSysConfigRefreshDelay)
		fsmID := encodeSubsys(getSubsysIndex(destID))
		if result := RequestStGlobalFromFSM(fsmID); result != 0 {
			setCTCPServerLastMessage("WebSocket saveParasInfo refresh StGlobal failed: fsm=0x%04X, dest=0x%04X, result=%d", uint32(fsmID), uint32(destID), result)
			return
		}
		setCTCPServerLastMessage("WebSocket saveParasInfo refresh StGlobal requested: fsm=0x%04X, dest=0x%04X", uint32(fsmID), uint32(destID))
	}()
}

func requestStGlobalAfterConfigCommand(topic string, destID int32) {
	go func() {
		fsmID := encodeSubsys(getSubsysIndex(destID))
		delays := []time.Duration{
			webSocketSysConfigRefreshDelay,
			1500 * time.Millisecond,
			3 * time.Second,
		}
		for index, delay := range delays {
			time.Sleep(delay)
			if result := RequestStGlobalFromFSM(fsmID); result != 0 {
				setCTCPServerLastMessage("WebSocket %s refresh StGlobal failed: attempt=%d, fsm=0x%04X, dest=0x%04X, result=%d", topic, index+1, uint32(fsmID), uint32(destID), result)
				return
			}
			setCTCPServerLastMessage("WebSocket %s refresh StGlobal requested: attempt=%d, fsm=0x%04X, dest=0x%04X", topic, index+1, uint32(fsmID), uint32(destID))
		}
	}()
}

func SendSimpleFSMCommand(topic string, commandID int32, control webSocketControlMessage) (int, int32, int) {
	destID := normalizeDropDataDestID(control)
	targetDestID := destID
	if destID < 0 {
		if control.FSMID != 0 {
			targetDestID = control.FSMID
		} else {
			targetDestID = cTCPDefaultFSMID
		}
	}
	targetIP, targetPort := resolveCTCPTarget(targetDestID, commandID, "", 0)
	setCTCPServerLastMessage(
		"WebSocket %s: sending cmd=0x%04X, dest=0x%04X, targetDest=0x%04X, target=%s:%d, payload=0 bytes",
		topic,
		uint32(commandID),
		uint32(destID),
		uint32(targetDestID),
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

func SendSimpleWAMCommand(topic string, commandID int32, control webSocketControlMessage) (int, int32, int) {
	destID := normalizeWAMDestID(control)
	return sendWAMCommand(topic, commandID, destID, nil)
}

func SendSimpleWAMChannelCommand(topic string, commandID int32, control webSocketControlMessage) (int, int32, int) {
	destID := normalizeWAMChannelDestID(control)
	return sendWAMCommand(topic, commandID, destID, nil)
}

func SendWAMResetAD(control webSocketControlMessage) (int, int32, int) {
	resetAD := StResetAD{}
	if control.ResetADValue != nil && *control.ResetADValue != 0 {
		resetAD.Value = 1
	}
	payload, err := encodeResetADPayload(resetAD)
	destID := normalizeWAMChannelDestID(control)
	if err != nil {
		setCTCPServerLastMessage("WebSocket wamResetAd failed: encode StResetAD: %v", err)
		return -1, destID, 0
	}
	return sendWAMCommand("wamResetAd", cTCPHCWAMResetAD, destID, payload)
}

func SendWAMWeightInfoData(control webSocketControlMessage) (int, int32, int) {
	destID := normalizeWAMChannelDestID(control)
	if control.WeightInfo == nil {
		setCTCPServerLastMessage("WebSocket saveWamWeightInfo failed: empty StWeightBaseInfo, dest=0x%04X", uint32(destID))
		return -1, destID, 0
	}
	payload, err := encodeWeightBaseInfoPayload(*control.WeightInfo)
	if err != nil {
		setCTCPServerLastMessage("WebSocket saveWamWeightInfo failed: encode StWeightBaseInfo: %v", err)
		return -1, destID, 0
	}
	return sendWAMCommand("saveWamWeightInfo", cTCPHCWAMWeightInfo, destID, payload)
}

func SendWAMGlobalWeightInfoData(control webSocketControlMessage) (int, int32, int) {
	destID := normalizeWAMDestID(control)
	if control.GlobalWeightInfo == nil {
		setCTCPServerLastMessage("WebSocket saveWamGlobalWeightInfo failed: empty StGlobalWeightBaseInfo, dest=0x%04X", uint32(destID))
		return -1, destID, 0
	}
	payload, err := encodeGlobalWeightBaseInfoPayload(*control.GlobalWeightInfo)
	if err != nil {
		setCTCPServerLastMessage("WebSocket saveWamGlobalWeightInfo failed: encode StGlobalWeightBaseInfo: %v", err)
		return -1, destID, 0
	}
	return sendWAMCommand("saveWamGlobalWeightInfo", cTCPHCWAMGlobalWeightInfo, destID, payload)
}

func sendWAMCommand(topic string, commandID int32, destID int32, payload []byte) (int, int32, int) {
	targetIP, targetPort := resolveCTCPTarget(destID, commandID, "", 0)
	setCTCPServerLastMessage(
		"WebSocket %s: sending WAM cmd=0x%04X, dest=0x%04X, target=%s:%d, payload=%d bytes",
		topic,
		uint32(commandID),
		uint32(destID),
		targetIP,
		targetPort,
		len(payload),
	)

	result := StartCTCPClient(targetIP, targetPort, destID, commandID, payload)
	if result != 0 {
		setCTCPServerLastMessage("WebSocket %s failed: WAM cmd=0x%04X result=%d", topic, uint32(commandID), result)
		return result, destID, len(payload)
	}

	setCTCPServerLastMessage("WebSocket %s success: WAM cmd=0x%04X sent, dest=0x%04X", topic, uint32(commandID), uint32(destID))
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

func summarizeGradeExitLowHigh(grade StGradeInfo) string {
	const maxSummaryItems = 16

	parts := make([]string, 0, maxSummaryItems)
	total := 0
	for index, item := range grade.Grades {
		if item.ExitLow == 0 && item.ExitHigh == 0 {
			continue
		}
		total++
		if len(parts) < maxSummaryItems {
			parts = append(parts, fmt.Sprintf(
				"%d:exitLOW=0x%08X,exitHight=0x%08X,exit64=%d",
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

func summarizeGradeExitMappingsByExit(grade StGradeInfo) string {
	const maxExitSummaryItems = 12
	const maxGradeSummaryItems = 16

	parts := make([]string, 0, maxExitSummaryItems)
	for exitNo := 1; exitNo <= 48; exitNo++ {
		mask := uint64(1) << uint(exitNo-1)
		gradeIndexes := make([]string, 0, maxGradeSummaryItems)
		total := 0
		for gradeIndex, item := range grade.Grades {
			if item.Exit()&mask == 0 {
				continue
			}
			total++
			if len(gradeIndexes) < maxGradeSummaryItems {
				gradeIndexes = append(gradeIndexes, fmt.Sprintf("%d", gradeIndex))
			}
		}
		if total == 0 {
			continue
		}
		if total > len(gradeIndexes) {
			gradeIndexes = append(gradeIndexes, fmt.Sprintf("+%d", total-len(gradeIndexes)))
		}
		parts = append(parts, fmt.Sprintf("exit%d=[%s]", exitNo, strings.Join(gradeIndexes, "|")))
		if len(parts) >= maxExitSummaryItems {
			break
		}
	}
	if len(parts) == 0 {
		return "none"
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

func normalizeWAMDestID(control webSocketControlMessage) int32 {
	destID := control.DestID
	if destID == 0 {
		destID = control.FSMID
	}
	if destID == 0 {
		destID = cTCPDefaultFSMID
	}
	subsysBits := destID & 0x0F00
	if subsysBits == 0 {
		subsysBits = cTCPDefaultFSMID & 0x0F00
	}
	return subsysBits | 0x00D0
}

func normalizeWAMChannelDestID(control webSocketControlMessage) int32 {
	if control.DestID != 0 && (control.DestID&0x000F) != 0 {
		return control.DestID
	}
	channelIndex := 0
	if control.ChannelIndex != nil {
		channelIndex = *control.ChannelIndex
	}
	if channelIndex < 0 {
		channelIndex = 0
	}
	if channelIndex >= cTCPMaxChannelNum {
		channelIndex = cTCPMaxChannelNum - 1
	}
	return normalizeWAMDestID(control) | int32(channelIndex+1)
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

	op := "add"
	if !add {
		op = "remove"
	}
	for _, item := range grades {
		index, err := resolveDropGradeIndex(item)
		if err != nil {
			return err
		}
		beforeLow := grade.Grades[index].ExitLow
		beforeHigh := grade.Grades[index].ExitHigh
		applyGradeExitBit(&grade.Grades[index], exitNo, add)
		setCTCPServerLastMessage(
			"WebSocket dropdata exitLOW/exitHight item: action=%s, exitNo=%d, gradeIndex=%d, %s, name=%q, beforeLOW=0x%08X, beforeHight=0x%08X, afterLOW=0x%08X, afterHight=0x%08X",
			op,
			exitNo,
			index,
			formatDropGradeCoord(item),
			item.Name,
			beforeLow,
			beforeHigh,
			grade.Grades[index].ExitLow,
			grade.Grades[index].ExitHigh,
		)
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

func formatDropGradeCoord(item webSocketDropGrade) string {
	if item.Row != nil && item.Col != nil {
		return fmt.Sprintf("row=%d, col=%d", *item.Row, *item.Col)
	}
	if item.Index != nil {
		return fmt.Sprintf("index=%d", *item.Index)
	}
	return "row=-, col=-"
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

func shouldPreserveCachedGradeExits(target StGradeInfo, cached StGradeInfo, explicitPreserve *bool) bool {
	if explicitPreserve != nil {
		return *explicitPreserve
	}
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
		beforeLow := grade.Grades[item.Index].ExitLow
		beforeHigh := grade.Grades[item.Index].ExitHigh
		grade.Grades[item.Index].ExitLow = uint32(next)
		grade.Grades[item.Index].ExitHigh = uint32(next >> 32)
		setCTCPServerLastMessage(
			"WebSocket gradeExits exitLOW/exitHight item: gradeIndex=%d, exitMask=%d, beforeLOW=0x%08X, beforeHight=0x%08X, afterLOW=0x%08X, afterHight=0x%08X",
			item.Index,
			exitMask,
			beforeLow,
			beforeHigh,
			grade.Grades[item.Index].ExitLow,
			grade.Grades[item.Index].ExitHigh,
		)
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

func encodeParasInfoPayload(paras StParas) ([]byte, error) {
	size := int(unsafe.Sizeof(paras))
	if size != cTCP48StParasWireSize {
		return nil, fmt.Errorf("sizeof(StParas)=%d, expected=%d", size, cTCP48StParasWireSize)
	}

	payload := make([]byte, size)
	src := unsafe.Slice((*byte)(unsafe.Pointer(&paras)), size)
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

func encodeGlobalExitInfoPayload(globalExitInfo StGlobalExitInfo) ([]byte, error) {
	size := int(unsafe.Sizeof(globalExitInfo))
	if size != cTCP48StGlobalExitInfoSize {
		return nil, fmt.Errorf("sizeof(StGlobalExitInfo)=%d, expected=%d", size, cTCP48StGlobalExitInfoSize)
	}

	payload := make([]byte, size)
	src := unsafe.Slice((*byte)(unsafe.Pointer(&globalExitInfo)), size)
	copy(payload, src)
	return payload, nil
}

func encodeAnalogDensityPayload(densityInfo StAnalogDensity) ([]byte, error) {
	size := int(unsafe.Sizeof(densityInfo))
	expected := cTCP48AnalogDensitySlots * 4
	if size != expected {
		return nil, fmt.Errorf("sizeof(StAnalogDensity)=%d, expected=%d", size, expected)
	}

	payload := make([]byte, size)
	src := unsafe.Slice((*byte)(unsafe.Pointer(&densityInfo)), size)
	copy(payload, src)
	return payload, nil
}

func encodeWeightBaseInfoPayload(weightInfo StWeightBaseInfo) ([]byte, error) {
	size := int(unsafe.Sizeof(weightInfo))
	payload := make([]byte, size)
	src := unsafe.Slice((*byte)(unsafe.Pointer(&weightInfo)), size)
	copy(payload, src)
	return payload, nil
}

func encodeGlobalWeightBaseInfoPayload(globalWeightInfo StGlobalWeightBaseInfo) ([]byte, error) {
	size := int(unsafe.Sizeof(globalWeightInfo))
	payload := make([]byte, size)
	src := unsafe.Slice((*byte)(unsafe.Pointer(&globalWeightInfo)), size)
	copy(payload, src)
	return payload, nil
}

func encodeResetADPayload(resetAD StResetAD) ([]byte, error) {
	size := int(unsafe.Sizeof(resetAD))
	payload := make([]byte, size)
	src := unsafe.Slice((*byte)(unsafe.Pointer(&resetAD)), size)
	copy(payload, src)
	return payload, nil
}

func encodeExitInfoPayload(exitInfo StExitInfo) ([]byte, error) {
	size := int(unsafe.Sizeof(exitInfo))
	if size != cTCP48StExitInfoWireSize {
		return nil, fmt.Errorf("sizeof(StExitInfo)=%d, expected=%d", size, cTCP48StExitInfoWireSize)
	}

	payload := make([]byte, size)
	src := unsafe.Slice((*byte)(unsafe.Pointer(&exitInfo)), size)
	copy(payload, src)
	return payload, nil
}

func encodeSysConfigPayload(sysConfig StSysConfig) ([]byte, error) {
	size := int(unsafe.Sizeof(sysConfig))
	if size != cTCP48StSysConfigWireSize {
		return nil, fmt.Errorf("sizeof(StSysConfig)=%d, expected=%d", size, cTCP48StSysConfigWireSize)
	}

	payload := make([]byte, size)
	src := unsafe.Slice((*byte)(unsafe.Pointer(&sysConfig)), size)
	copy(payload, src)
	return payload, nil
}
