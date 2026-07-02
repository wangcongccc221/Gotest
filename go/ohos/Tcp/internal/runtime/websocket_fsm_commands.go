package tcp

import (
	"fmt"
	"time"
)

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

func SendTestValveData(topic string, commandID int32, control webSocketControlMessage) (int, int32, int) {
	destID := normalizeValveTestDestID(control)
	exitIndex := 255
	if control.ExitIndex != nil {
		exitIndex = *control.ExitIndex
	}
	if exitIndex < 0 || exitIndex > 255 {
		setCTCPServerLastMessage("WebSocket %s failed: exitIndex=%d out of range, dest=0x%04X", topic, exitIndex, uint32(destID))
		return -1, destID, 0
	}

	volveTest := StVolveTest{ExitId: uint8(exitIndex)}
	payload := encodeVolveTestPayload(volveTest)
	targetIP, targetPort := resolveCTCPTarget(destID, commandID, "", 0)
	channelText := "<nil>"
	if control.ChannelIndex != nil {
		channelText = fmt.Sprintf("%d", *control.ChannelIndex)
	}
	setCTCPServerLastMessage(
		"WebSocket %s: sending cmd=0x%04X, dest=0x%04X, target=%s:%d, payload=%d bytes, channel=%s, StVolveTest{ExitId=%d, hex=0x%02X}",
		topic,
		uint32(commandID),
		uint32(destID),
		targetIP,
		targetPort,
		len(payload),
		channelText,
		volveTest.ExitId,
		volveTest.ExitId,
	)

	result := StartCTCPClient(targetIP, targetPort, destID, commandID, payload)
	if result != 0 {
		setCTCPServerLastMessage("WebSocket %s failed: cmd=0x%04X result=%d", topic, uint32(commandID), result)
		return result, destID, len(payload)
	}

	setCTCPServerLastMessage("WebSocket %s success: cmd=0x%04X sent, dest=0x%04X, exit=%d", topic, uint32(commandID), uint32(destID), exitIndex)
	return 0, destID, len(payload)
}

func encodeVolveTestPayload(volveTest StVolveTest) []byte {
	return []byte{volveTest.ExitId}
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
