package tcp

import (
	"strings"

	"gotest/ohos/database"
)

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

func (c *webSocketClient) handleSysFruitInfoQuery(control webSocketControlMessage) {
	go func() {
		query := control.SysFruitInfoQuery
		if query == nil {
			c.sendCommandAckDetail("querySysFruitInfo", 0, 0, 0, -1, "sysFruitInfoQuery is required", control.RequestID)
			return
		}

		result, err := database.QuerySysFruitInfo(*query)
		payloadBytes := len(rawJSONFromValue(*query))
		if err != nil {
			setCTCPServerLastMessage("WebSocket querySysFruitInfo failed: err=%v", err)
			c.sendCommandAckDetail("querySysFruitInfo", 0, 0, payloadBytes, -1, err.Error(), control.RequestID)
			return
		}

		setCTCPServerLastMessage(
			"WebSocket querySysFruitInfo success: rows=%d, total=%d, page=%d, size=%d, database=%s",
			len(result.Items),
			result.TotalCount,
			result.PageIndex,
			result.PageSize,
			database.RealtimeSaveDatabaseForLog(),
		)
		c.sendFrame(webSocketFrame{
			Type:  "commandAck",
			Topic: "querySysFruitInfo",
			Data: rawJSONFromValue(map[string]any{
				"result":       0,
				"ok":           true,
				"command":      "querySysFruitInfo",
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

func (c *webSocketClient) handleSortLogQuery(control webSocketControlMessage) {
	go func() {
		query := control.SortLogQuery
		if query == nil {
			c.sendCommandAckDetail("querySortLog", 0, 0, 0, -1, "sortLogQuery is required", control.RequestID)
			return
		}

		result, err := database.QuerySortLog(*query)
		payloadBytes := len(rawJSONFromValue(*query))
		if err != nil {
			setCTCPServerLastMessage("WebSocket querySortLog failed: err=%v", err)
			c.sendCommandAckDetail("querySortLog", 0, 0, payloadBytes, -1, err.Error(), control.RequestID)
			return
		}

		setCTCPServerLastMessage(
			"WebSocket querySortLog success: begin=%s, end=%s, days=%d, totalMinutes=%d, database=%s",
			firstNonEmptyLog(query.StartDate, query.StartTime),
			firstNonEmptyLog(query.EndDate, query.EndTime),
			len(result.Items),
			result.TotalMinutes,
			database.RealtimeSaveDatabaseForLog(),
		)
		c.sendFrame(webSocketFrame{
			Type:  "commandAck",
			Topic: "querySortLog",
			Data: rawJSONFromValue(map[string]any{
				"result":       0,
				"ok":           true,
				"command":      "querySortLog",
				"cmdId":        0,
				"destId":       0,
				"payloadBytes": payloadBytes,
				"message":      "database queried",
				"requestId":    control.RequestID,
				"SortLogItems": result.Items,
				"TotalMinutes": result.TotalMinutes,
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

func firstNonEmptyLog(values ...string) string {
	for _, value := range values {
		text := strings.TrimSpace(value)
		if text != "" {
			return text
		}
	}
	return ""
}
