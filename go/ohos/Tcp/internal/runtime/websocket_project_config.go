package tcp

import (
	"strings"
)

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
		c.sendCommandAckDetail("saveFruitTypeConfig", 0, control.FSMID, 0, -1, "fruitTypeConfig is required", control.RequestID)
		return
	}
	if err := saveFruitTypeConfigFromControl(control.FSMID, *control.FruitTypeConfig); err != nil {
		c.sendCommandAckDetail("saveFruitTypeConfig", 0, control.FSMID, 0, -1, err.Error(), control.RequestID)
		return
	}
	c.sendCommandAckDetail("saveFruitTypeConfig", 0, control.FSMID, 0, 0, "database saved and verified", control.RequestID)
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
