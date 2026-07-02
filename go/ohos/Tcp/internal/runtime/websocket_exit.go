package tcp

import (
	"fmt"
	"strings"
)

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
	result := 0
	message := "sent"
	if control.ExitDisplay == nil {
		setCTCPServerLastMessage("WebSocket saveExitDisplay failed: empty ExitDisplayInfo, fsmId=0x%04X", uint32(control.FSMID))
		result = -1
		message = "empty ExitDisplayInfo"
	} else {
		baseInfo := defaultExitDisplayInfo()
		if _, cachedInfo, ok := latestExitDisplayInfoSnapshot(); ok {
			baseInfo = cachedInfo
		}
		info := applyExitDisplayUpdate(baseInfo, *control.ExitDisplay)
		if err := SaveExitDisplayInfoToLocalConfig(control.FSMID, info); err != nil {
			result = -1
			message = "save failed"
		} else {
			setLastExitDisplayInfoSnapshot(control.FSMID, info)
			publishExitDisplayData(control.FSMID, info)
			if isExitDisplayBroadcastEnabled() {
				BroadcastExitDisplayOnce()
			}
		}
	}
	c.sendCommandAckDetail("saveExitDisplay", 0, control.FSMID, 0, result, message, control.RequestID)
}

func (c *webSocketClient) handleExitDisplayEnabled(control webSocketControlMessage) {
	result := 0
	message := "sent"
	if control.ExitDisplayEnabled == nil {
		result = -1
		message = "empty exitDisplayEnabled"
		setCTCPServerLastMessage("WebSocket setExitDisplayEnabled failed: empty enabled")
	} else if err := SaveExitDisplayBroadcastEnabledToLocalConfig(*control.ExitDisplayEnabled); err != nil {
		result = -1
		message = "save failed"
		setCTCPServerLastMessage("WebSocket setExitDisplayEnabled failed: enabled=%t, err=%v", *control.ExitDisplayEnabled, err)
	} else {
		if *control.ExitDisplayEnabled {
			BroadcastExitDisplayOnce()
		}
		setCTCPServerLastMessage("WebSocket setExitDisplayEnabled success: enabled=%t", *control.ExitDisplayEnabled)
	}
	c.sendCommandAckDetail("setExitDisplayEnabled", 0, 0, 0, result, message, control.RequestID)
	publishExitDisplayData(0, latestExitDisplayInfoOrDefault())
}

func (c *webSocketClient) handleExitScreenSettings(control webSocketControlMessage) {
	result := 0
	message := "sent"
	if control.ExitScreenSettings == nil {
		result = -1
		message = "empty exitScreenSettings"
		setCTCPServerLastMessage("WebSocket saveExitScreenSettings failed: empty settings, fsmId=0x%04X", uint32(control.FSMID))
	} else {
		displayInfo, additionalInfo := buildExitScreenSettingsInfo(*control.ExitScreenSettings)
		if err := SaveExitDisplayInfoToLocalConfig(control.FSMID, displayInfo); err != nil {
			result = -1
			message = "save display failed"
		} else if err := SaveExitAdditionalTextInfoToLocalConfig(control.FSMID, additionalInfo); err != nil {
			result = -1
			message = "save additional text failed"
		} else {
			setLastExitDisplayInfoSnapshot(control.FSMID, displayInfo)
			setLastExitAdditionalTextInfoSnapshot(control.FSMID, additionalInfo)
			publishExitDisplayData(control.FSMID, displayInfo)
			publishExitAdditionalTextData(control.FSMID, additionalInfo)
			if isExitDisplayBroadcastEnabled() {
				BroadcastExitDisplayOnce()
			}
			setCTCPServerLastMessage(
				"WebSocket saveExitScreenSettings success: fsmId=0x%04X, configs=%d, displayType=%d",
				uint32(control.FSMID),
				len(control.ExitScreenSettings.Configs),
				displayInfo.DisplayType,
			)
		}
	}
	c.sendCommandAckDetail("saveExitScreenSettings", 0, control.FSMID, 0, result, message, control.RequestID)
}

func buildExitScreenSettingsInfo(settings webSocketExitScreenSettingsControl) (ExitDisplayInfo, ExitAdditionalTextInfo) {
	displayInfo := defaultExitDisplayInfo()
	additionalInfo := defaultExitAdditionalTextInfo()
	for _, config := range settings.Configs {
		exitIndex := config.ExitNo - 1
		if exitIndex < 0 || exitIndex >= cTCP48MaxExitNum {
			continue
		}

		useAutoName := true
		if config.UseAutoName != nil {
			useAutoName = *config.UseAutoName
		}
		displayInfo.DisplayType = setExitDisplayNameEnabled(displayInfo.DisplayType, exitIndex, !useAutoName)
		displayInfo.DisplayNames[exitIndex] = strings.TrimSpace(config.CustomName)
		additionalInfo.AdditionalTexts[exitIndex] = strings.TrimSpace(config.AdditionalInfo)
	}
	return displayInfo, additionalInfo
}

func buildExitScreenSettingsControl(displayInfo ExitDisplayInfo, additionalInfo ExitAdditionalTextInfo) webSocketExitScreenSettingsControl {
	settings := webSocketExitScreenSettingsControl{
		Configs: make([]webSocketExitScreenConfigControl, 0, cTCP48MaxExitNum),
	}
	for i := 0; i < cTCP48MaxExitNum; i++ {
		useAutoName := !exitDisplayNameEnabled(displayInfo.DisplayType, i)
		settings.Configs = append(settings.Configs, webSocketExitScreenConfigControl{
			ExitNo:         i + 1,
			CustomName:     displayInfo.DisplayNames[i],
			AdditionalInfo: additionalInfo.AdditionalTexts[i],
			UseAutoName:    &useAutoName,
		})
	}
	return settings
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
		Data: rawJSONFromValue(webSocketExitDisplayData{
			FSMID:              fsmID,
			Enabled:            isExitDisplayBroadcastEnabled(),
			ExitDisplay:        info,
			ExitScreenSettings: buildExitScreenSettingsControl(info, latestExitAdditionalTextInfoOrDefault()),
		}),
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
	raw := rawJSONFromValue(webSocketExitDisplayData{
		FSMID:              fsmID,
		Enabled:            isExitDisplayBroadcastEnabled(),
		ExitDisplay:        info,
		ExitScreenSettings: buildExitScreenSettingsControl(info, latestExitAdditionalTextInfoOrDefault()),
	})
	frame, err := newWebSocketDataFrame(webSocketTopicExitDisplay, raw)
	if err != nil {
		setCTCPServerLastMessage("WebSocket publish exitDisplay failed: %v", err)
		return
	}
	defaultWebSocketHub.publish(webSocketTopicExitDisplay, frame)
}

func latestExitDisplayInfoOrDefault() ExitDisplayInfo {
	if _, info, ok := latestExitDisplayInfoSnapshot(); ok {
		return info
	}
	return defaultExitDisplayInfo()
}

func latestExitAdditionalTextInfoOrDefault() ExitAdditionalTextInfo {
	if _, info, ok := latestExitAdditionalTextInfoSnapshot(); ok {
		return info
	}
	return defaultExitAdditionalTextInfo()
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
