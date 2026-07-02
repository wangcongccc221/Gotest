package tcp

import (
	"fmt"
	"strings"
)

func SendSimpleWAMCommand(topic string, commandID int32, control webSocketControlMessage) (int, int32, int) {
	destID := normalizeWAMDestID(control)
	return sendWAMCommand(topic, commandID, destID, nil)
}

func SendResetCupWAMCommand(topic string, commandID int32, control webSocketControlMessage) (int, int32, int) {
	targetID := control.FSMID
	if control.DestID != 0 {
		targetID = control.DestID
	}
	dests := buildWAMDestIDsForResetCup(targetID)
	if len(dests) == 0 {
		setCTCPServerLastMessage("WebSocket %s failed: empty WAM dests, fsm=0x%04X, dest=0x%04X", topic, uint32(control.FSMID), uint32(control.DestID))
		return -1, 0, 0
	}

	result := 0
	for _, destID := range dests {
		if sendResult, _, _ := sendWAMCommand(topic, commandID, destID, nil); sendResult != 0 {
			result = sendResult
		}
	}
	setCTCPServerLastMessage(
		"WebSocket %s summary: cmd=0x%04X, wamDests=%s, result=%d",
		topic,
		uint32(commandID),
		formatInt32HexList(dests),
		result,
	)
	return result, dests[0], 0
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

func buildWAMDestIDsForResetCup(fsmID int32) []int32 {
	if fsmID <= 0 {
		return []int32{encodeWAMID(0)}
	}
	subsysIndex := getSubsysIndex(fsmID)
	if subsysIndex < 0 {
		subsysIndex = 0
	}
	return []int32{encodeWAMID(subsysIndex)}
}

func formatInt32HexList(values []int32) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprintf("0x%04X", uint32(value)))
	}
	return "[" + strings.Join(parts, ",") + "]"
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
