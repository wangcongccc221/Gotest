package tcp

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"gotest/ohos/database"
)

const (
	cTCPSelectedFruitTypesConfigName = "已选水果种类"
	cTCPGradeAccuracyConfigName      = "GradeAccracy"
	cTCPExitAlarmThresholdConfigName = "出口报警阈值"

	cTCPDefaultSelectedFruitTypes = "1.1-新鲜脐橙;3.1-安岳柠檬;12.1-苹果;"
	cTCPDefaultExitAlarmThreshold = 30
)

type LevelAuxConfigInfo struct {
	SelectedFruitTypes string `json:"selectedFruitTypes"`
	GradeAccuracy      int    `json:"gradeAccuracy"`
	ExitAlarmThreshold int    `json:"exitAlarmThreshold"`
}

var (
	lastLevelAuxConfigInfoSnapshot   LevelAuxConfigInfo
	lastLevelAuxConfigInfoFSMID      int32
	lastLevelAuxConfigInfoSnapshotOK bool
	lastLevelAuxConfigInfoSnapshotMu sync.RWMutex
)

func LoadLevelAuxConfigInfoFromLocalConfig() {
	info, err := ReadLevelAuxConfigInfoFromLocalConfig()
	if err != nil {
		setCTCPServerLastMessage("等级辅助配置读取失败: %v", err)
		return
	}

	setLastLevelAuxConfigInfoSnapshot(0, info)
	setCTCPServerLastMessage(
		"等级辅助配置已加载: selectedFruitTypesLen=%d, gradeAccuracy=%d, exitAlarmThreshold=%d",
		len(info.SelectedFruitTypes),
		info.GradeAccuracy,
		info.ExitAlarmThreshold,
	)
}

func ReadLevelAuxConfigInfoFromLocalConfig() (LevelAuxConfigInfo, error) {
	info := defaultLevelAuxConfigInfo()

	selectedFruitTypes, err := database.GetConfigValue(cTCPSelectedFruitTypesConfigName)
	if err != nil {
		return LevelAuxConfigInfo{}, fmt.Errorf("read %s: %w", cTCPSelectedFruitTypesConfigName, err)
	}
	if strings.TrimSpace(selectedFruitTypes) != "" {
		info.SelectedFruitTypes = selectedFruitTypes
	}

	gradeAccuracyText, err := database.GetConfigValue(cTCPGradeAccuracyConfigName)
	if err != nil {
		return LevelAuxConfigInfo{}, fmt.Errorf("read %s: %w", cTCPGradeAccuracyConfigName, err)
	}
	gradeAccuracy, err := parseIntConfigValue(cTCPGradeAccuracyConfigName, gradeAccuracyText, info.GradeAccuracy)
	if err != nil {
		return LevelAuxConfigInfo{}, err
	}
	info.GradeAccuracy = clampInt(gradeAccuracy, 0, 6)

	exitAlarmThresholdText, err := database.GetConfigValue(cTCPExitAlarmThresholdConfigName)
	if err != nil {
		return LevelAuxConfigInfo{}, fmt.Errorf("read %s: %w", cTCPExitAlarmThresholdConfigName, err)
	}
	exitAlarmThreshold, err := parseIntConfigValue(cTCPExitAlarmThresholdConfigName, exitAlarmThresholdText, info.ExitAlarmThreshold)
	if err != nil {
		return LevelAuxConfigInfo{}, err
	}
	info.ExitAlarmThreshold = clampInt(exitAlarmThreshold, 0, 100)

	return info, nil
}

func SaveLevelAuxConfigInfoToLocalConfig(fsmID int32, info LevelAuxConfigInfo) error {
	info = normalizeLevelAuxConfigInfo(info)

	if err := database.SaveConfigValue(cTCPSelectedFruitTypesConfigName, info.SelectedFruitTypes); err != nil {
		setCTCPServerLastMessage("已选水果种类保存失败: %v", err)
		return err
	}
	if err := database.SaveConfigValue(cTCPGradeAccuracyConfigName, strconv.Itoa(info.GradeAccuracy)); err != nil {
		setCTCPServerLastMessage("等级重量精度保存失败: %v", err)
		return err
	}
	if err := database.SaveConfigValue(cTCPExitAlarmThresholdConfigName, strconv.Itoa(info.ExitAlarmThreshold)); err != nil {
		setCTCPServerLastMessage("出口报警阈值保存失败: %v", err)
		return err
	}

	setCTCPServerLastMessage(
		"等级辅助配置已保存: fsmId=0x%04X, selectedFruitTypesLen=%d, gradeAccuracy=%d, exitAlarmThreshold=%d",
		uint32(fsmID),
		len(info.SelectedFruitTypes),
		info.GradeAccuracy,
		info.ExitAlarmThreshold,
	)
	return nil
}

func defaultLevelAuxConfigInfo() LevelAuxConfigInfo {
	return LevelAuxConfigInfo{
		SelectedFruitTypes: cTCPDefaultSelectedFruitTypes,
		GradeAccuracy:      0,
		ExitAlarmThreshold: cTCPDefaultExitAlarmThreshold,
	}
}

func normalizeLevelAuxConfigInfo(info LevelAuxConfigInfo) LevelAuxConfigInfo {
	info.SelectedFruitTypes = strings.TrimSpace(info.SelectedFruitTypes)
	if info.SelectedFruitTypes == "" {
		info.SelectedFruitTypes = cTCPDefaultSelectedFruitTypes
	}
	info.GradeAccuracy = clampInt(info.GradeAccuracy, 0, 6)
	info.ExitAlarmThreshold = clampInt(info.ExitAlarmThreshold, 0, 100)
	return info
}

func parseIntConfigValue(key string, text string, defaultValue int) (int, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return defaultValue, nil
	}
	value, err := strconv.Atoi(text)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return value, nil
}

func clampInt(value int, min int, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func setLastLevelAuxConfigInfoSnapshot(fsmID int32, info LevelAuxConfigInfo) {
	lastLevelAuxConfigInfoSnapshotMu.Lock()
	lastLevelAuxConfigInfoSnapshot = normalizeLevelAuxConfigInfo(info)
	lastLevelAuxConfigInfoFSMID = fsmID
	lastLevelAuxConfigInfoSnapshotOK = true
	lastLevelAuxConfigInfoSnapshotMu.Unlock()
}

func latestLevelAuxConfigInfoSnapshot() (int32, LevelAuxConfigInfo, bool) {
	lastLevelAuxConfigInfoSnapshotMu.RLock()
	defer lastLevelAuxConfigInfoSnapshotMu.RUnlock()
	return lastLevelAuxConfigInfoFSMID, lastLevelAuxConfigInfoSnapshot, lastLevelAuxConfigInfoSnapshotOK
}
