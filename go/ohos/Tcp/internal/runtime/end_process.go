package tcp

import (
	"fmt"
	"strings"
	"time"

	"gotest/ohos/database"
)

const webSocketTopicEndProcess = "endProcess"

type webSocketEndProcessData struct {
	CustomerID     int     `json:"customerId"`
	FruitName      string  `json:"fruitName"`
	StartTime      string  `json:"startTime"`
	EndTime        string  `json:"endTime"`
	BatchNumber    uint64  `json:"batchNumber"`
	BatchWeight    float64 `json:"batchWeight"`
	CompletedState string  `json:"completedState"`
	CommandID      int32   `json:"cmdId"`
	DestIDs        []int32 `json:"destIds"`
	Action         string  `json:"action"`
	Database       string  `json:"database"`
}

func (c *webSocketClient) handleEndProcess(control webSocketControlMessage) {
	go func() {
		result, commandID, destID, payloadBytes := SendEndProcessCommand(control)
		c.sendCommandAck(webSocketTopicEndProcess, commandID, destID, payloadBytes, result)
	}()
}

func SendEndProcessCommand(control webSocketControlMessage) (int, int32, int32, int) {
	now := cTCPNow()
	commandID, action := endProcessCommandFromControl(control)
	destIDs := endProcessDestIDs(control)
	ackDestID := int32(-1)
	if len(destIDs) == 1 {
		ackDestID = destIDs[0]
	}

	setCTCPServerLastMessage(
		"WebSocket endProcess: action=%s, cmd=0x%04X, dests=%s",
		action,
		uint32(commandID),
		formatEndProcessDestIDs(destIDs),
	)

	if err := sendEndProcessDeviceCommand(commandID, destIDs); err != nil {
		setCTCPServerLastMessage("WebSocket endProcess failed: send device command: %v", err)
		return -1, commandID, ackDestID, 0
	}

	if err := saveRealtimeStatisticsBeforeEndProcess(now); err != nil {
		setCTCPServerLastMessage("WebSocket endProcess failed: realtime save before end: %v", err)
		return -2, commandID, ackDestID, 0
	}

	if result, err := database.ReportSortRunningStateAt(database.SortRunningStateRequest{IsEndProcess: true}, now); err != nil {
		setCTCPServerLastMessage("WebSocket endProcess failed: close sort running time: %v", err)
		return -3, commandID, ackDestID, 0
	} else {
		setCTCPServerLastMessage(
			"WebSocket endProcess sort running time: action=%s, closedId=%d",
			result.Action,
			result.ClosedRecordID,
		)
	}

	result, err := database.EndCurrentFruitProcess(now)
	if err != nil {
		setCTCPServerLastMessage("WebSocket endProcess failed: update database: %v", err)
		return -3, commandID, ackDestID, 0
	}

	markRealtimeSaveProcessEnded()
	publishEndProcessResult(result, commandID, destIDs, action)
	setCTCPServerLastMessage(
		"WebSocket endProcess success: CustomerID=%d, EndTime=%s, CompletedState=%s, database=%s",
		result.CustomerID,
		result.EndTime,
		result.CompletedState,
		database.RealtimeSaveDatabaseForLog(),
	)
	return 0, commandID, ackDestID, 0
}

func endProcessCommandFromControl(control webSocketControlMessage) (int32, string) {
	action := strings.ToLower(strings.TrimSpace(control.Action))
	switch action {
	case "savecurrent", "savecurrentdata", "keep", "keepexit", "keepexitboxes", "no":
		return cTCPHCSaveCurrentData, "saveCurrentData"
	default:
		return cTCPHCClearData, "clearData"
	}
}

func endProcessDestIDs(control webSocketControlMessage) []int32 {
	if control.DestID != 0 {
		return []int32{control.DestID}
	}
	if control.FSMID != 0 {
		return []int32{control.FSMID}
	}

	realtimeSaveMu.Lock()
	stg := realtimeSaveLatestGlobal
	hasGlobal := realtimeSaveHasGlobal
	realtimeSaveMu.Unlock()
	if hasGlobal {
		channelInfo := make([]uint8, len(stg.Sys.NChannelInfo))
		copy(channelInfo, stg.Sys.NChannelInfo[:])
		destIDs := getAllSysID(channelInfo)
		if len(destIDs) > 0 {
			return destIDs
		}
	}
	return []int32{cTCPDefaultFSMID}
}

func sendEndProcessDeviceCommand(commandID int32, destIDs []int32) error {
	if len(destIDs) == 0 {
		return fmt.Errorf("empty dest ids")
	}

	for _, destID := range destIDs {
		targetIP, targetPort := resolveCTCPTarget(destID, commandID, "", 0)
		setCTCPServerLastMessage(
			"WebSocket endProcess: sending cmd=0x%04X, dest=0x%04X, target=%s:%d, payload=0 bytes",
			uint32(commandID),
			uint32(destID),
			targetIP,
			targetPort,
		)
		result := StartCTCPClient(targetIP, targetPort, destID, commandID, nil)
		if result != 0 {
			return fmt.Errorf("dest=0x%04X result=%d", uint32(destID), result)
		}
	}
	return nil
}

func saveRealtimeStatisticsBeforeEndProcess(now time.Time) error {
	input, ok, reason := buildRealtimeSaveInput(now)
	if !ok {
		setCTCPServerLastMessage("WebSocket endProcess: realtime save before end skipped: %s", reason)
		return nil
	}

	customerID, err := database.SaveRealtimeFruitInfo(input)
	if err != nil {
		return err
	}
	setCTCPServerLastMessage("WebSocket endProcess: realtime save before end ok: CustomerID=%d", customerID)
	return nil
}

func publishEndProcessResult(result database.EndProcessResult, commandID int32, destIDs []int32, action string) {
	raw := rawJSONFromValue(webSocketEndProcessData{
		CustomerID:     result.CustomerID,
		FruitName:      result.FruitName,
		StartTime:      result.StartTime,
		EndTime:        result.EndTime,
		BatchNumber:    result.BatchNumber,
		BatchWeight:    result.BatchWeight,
		CompletedState: result.CompletedState,
		CommandID:      commandID,
		DestIDs:        destIDs,
		Action:         action,
		Database:       database.RealtimeSaveDatabaseForLog(),
	})
	frame, err := newWebSocketDataFrame(webSocketTopicEndProcess, raw)
	if err != nil {
		setCTCPServerLastMessage("WebSocket endProcess publish failed: %v", err)
		return
	}
	defaultWebSocketHub.publish(webSocketTopicEndProcess, frame)
}

func formatEndProcessDestIDs(destIDs []int32) string {
	if len(destIDs) == 0 {
		return "none"
	}
	items := make([]string, 0, len(destIDs))
	for _, destID := range destIDs {
		items = append(items, fmt.Sprintf("0x%04X", uint32(destID)))
	}
	return strings.Join(items, ",")
}
