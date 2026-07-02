package tcp

import (
	"fmt"
	"strings"
	"time"
)

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
		c.sendCommandAckDetail(
			"clearExitGrades",
			cTCPHCGradeInfo,
			destID,
			payloadBytes,
			result,
			commandAckMessage(result),
			control.RequestID,
		)
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

func (c *webSocketClient) handleTestValveData(topic string, commandID int32, control webSocketControlMessage) {
	go func() {
		result, destID, payloadBytes := SendTestValveData(topic, commandID, control)
		c.sendCommandAckDetail(topic, commandID, destID, payloadBytes, result, commandAckMessage(result), control.RequestID)
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
		c.sendCommandAckDetail(topic, commandID, destID, payloadBytes, result, commandAckMessage(result), control.RequestID)
	}()
}

func (c *webSocketClient) handleIpmGetMAC(control webSocketControlMessage) {
	go func() {
		ip := strings.TrimSpace(control.IP)
		setCTCPServerLastMessage("WebSocket ipmGetMac: lookup begin, ip=%s", ip)
		mac, err := lookupRemoteMAC(ip)
		result := 0
		message := "mac found"
		if err != nil {
			result = -1
			message = err.Error()
			setCTCPServerLastMessage("WebSocket ipmGetMac failed: ip=%s, error=%v", ip, err)
		} else {
			setCTCPServerLastMessage("WebSocket ipmGetMac success: ip=%s, mac=%s", ip, mac)
		}
		c.sendFrame(webSocketFrame{
			Type:  "commandAck",
			Topic: "ipmgetmac",
			Data: rawJSONFromValue(map[string]any{
				"result":       result,
				"ok":           result == 0,
				"command":      "ipmGetMac",
				"cmdId":        0,
				"destId":       0,
				"payloadBytes": 0,
				"message":      message,
				"requestId":    control.RequestID,
				"ip":           ip,
				"mac":          mac,
			}),
		})
	}()
}

func (c *webSocketClient) handleIpmWakeOnLAN(control webSocketControlMessage) {
	go func() {
		ip := strings.TrimSpace(control.IP)
		mac := strings.TrimSpace(control.MAC)
		setCTCPServerLastMessage("WebSocket ipmWakeOnLan: sending WOL begin, ip=%s, mac=%s", ip, mac)
		sentCount, err := sendWakeOnLAN(ip, mac)
		result := 0
		message := fmt.Sprintf("wol sent: %d", sentCount)
		if err != nil {
			result = -1
			message = err.Error()
			setCTCPServerLastMessage("WebSocket ipmWakeOnLan failed: ip=%s, mac=%s, error=%v", ip, mac, err)
		} else {
			setCTCPServerLastMessage("WebSocket ipmWakeOnLan success: ip=%s, mac=%s, sentPackets=%d", ip, mac, sentCount)
		}
		c.sendFrame(webSocketFrame{
			Type:  "commandAck",
			Topic: "ipmwakeonlan",
			Data: rawJSONFromValue(map[string]any{
				"result":       result,
				"ok":           result == 0,
				"command":      "ipmWakeOnLan",
				"cmdId":        0,
				"destId":       0,
				"payloadBytes": 0,
				"message":      message,
				"requestId":    control.RequestID,
				"ip":           ip,
				"mac":          mac,
			}),
		})
	}()
}

func (c *webSocketClient) handleSimpleWAMCommand(topic string, commandID int32, control webSocketControlMessage) {
	result, destID, payloadBytes := SendSimpleWAMCommand(topic, commandID, control)
	c.sendCommandAck(topic, commandID, destID, payloadBytes, result)
}

func (c *webSocketClient) handleResetCupCommand(topic string, commandID int32, control webSocketControlMessage) {
	go func() {
		result, destID, payloadBytes := SendResetCupWAMCommand(topic, commandID, control)
		c.sendCommandAck(topic, commandID, destID, payloadBytes, result)
	}()
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
