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
	cTCPWeightStandardConfigName     = "重量等级标准"
	cTCPPackingWeight1ConfigName     = "装箱数1"
	cTCPPackingWeight2ConfigName     = "装箱数2"

	cTCPDefaultSelectedFruitTypes = "1.1-新鲜脐橙;3.1-安岳柠檬;12.1-苹果;"
	cTCPDefaultExitAlarmThreshold = 30
	cTCPDefaultPackingWeight1     = 20.0
	cTCPDefaultPackingWeight2     = 15.0
)

type LevelAuxConfigInfo struct {
	SelectedFruitTypes string   `json:"selectedFruitTypes"`
	GradeAccuracy      int      `json:"gradeAccuracy"`
	ExitAlarmThreshold int      `json:"exitAlarmThreshold"`
	PackingWeight1     float64  `json:"packingWeight1"`
	PackingWeight2     float64  `json:"packingWeight2"`
	WeightStandards    []int    `json:"weightStandards"`
	LabelerNames       []string `json:"labelerNames"`
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
		"等级辅助配置已加载: selectedFruitTypesLen=%d, gradeAccuracy=%d, exitAlarmThreshold=%d, packingWeight1=%.3f, packingWeight2=%.3f, weightStandards=%s, labelerNames=%s",
		len(info.SelectedFruitTypes),
		info.GradeAccuracy,
		info.ExitAlarmThreshold,
		info.PackingWeight1,
		info.PackingWeight2,
		formatIntListForLog(info.WeightStandards),
		formatStringListForLog(info.LabelerNames),
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

	packingWeight1Text, err := database.GetConfigValue(cTCPPackingWeight1ConfigName)
	if err != nil {
		return LevelAuxConfigInfo{}, fmt.Errorf("read %s: %w", cTCPPackingWeight1ConfigName, err)
	}
	packingWeight1, err := parseFloatConfigValue(cTCPPackingWeight1ConfigName, packingWeight1Text, info.PackingWeight1)
	if err != nil {
		return LevelAuxConfigInfo{}, err
	}
	info.PackingWeight1 = packingWeight1

	packingWeight2Text, err := database.GetConfigValue(cTCPPackingWeight2ConfigName)
	if err != nil {
		return LevelAuxConfigInfo{}, fmt.Errorf("read %s: %w", cTCPPackingWeight2ConfigName, err)
	}
	packingWeight2, err := parseFloatConfigValue(cTCPPackingWeight2ConfigName, packingWeight2Text, info.PackingWeight2)
	if err != nil {
		return LevelAuxConfigInfo{}, err
	}
	info.PackingWeight2 = packingWeight2

	weightStandardText, err := database.GetConfigValue(cTCPWeightStandardConfigName)
	if err != nil {
		return LevelAuxConfigInfo{}, fmt.Errorf("read %s: %w", cTCPWeightStandardConfigName, err)
	}
	if strings.TrimSpace(weightStandardText) != "" {
		info.WeightStandards = parseWeightStandardsConfig(weightStandardText)
	}

	for i := 0; i < 4; i++ {
		key := fmt.Sprintf("贴标机%d", i+1)
		labelerName, err := database.GetConfigValuePreferNonEmpty(key)
		if err != nil {
			return LevelAuxConfigInfo{}, fmt.Errorf("read %s: %w", key, err)
		}
		info.LabelerNames[i] = labelerName
	}

	info = normalizeLevelAuxConfigInfo(info)
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
	if err := database.SaveConfigValue(cTCPPackingWeight1ConfigName, strconv.FormatFloat(info.PackingWeight1, 'f', -1, 64)); err != nil {
		setCTCPServerLastMessage("装箱数1保存失败: %v", err)
		return err
	}
	if err := database.SaveConfigValue(cTCPPackingWeight2ConfigName, strconv.FormatFloat(info.PackingWeight2, 'f', -1, 64)); err != nil {
		setCTCPServerLastMessage("装箱数2保存失败: %v", err)
		return err
	}
	if err := database.SaveConfigValue(cTCPWeightStandardConfigName, formatWeightStandardsConfig(info.WeightStandards)); err != nil {
		setCTCPServerLastMessage("重量等级标准保存失败: %v", err)
		return err
	}
	for i := 0; i < 4; i++ {
		key := fmt.Sprintf("贴标机%d", i+1)
		labelerName := strings.TrimSpace(info.LabelerNames[i])
		if labelerName == "" {
			storedLabelerName, err := database.GetConfigValuePreferNonEmpty(key)
			if err != nil {
				setCTCPServerLastMessage("%s读取已有值失败: %v", key, err)
				return err
			}
			labelerName = strings.TrimSpace(storedLabelerName)
			info.LabelerNames[i] = labelerName
		}
		if err := database.SaveConfigValue(key, labelerName); err != nil {
			setCTCPServerLastMessage("%s保存失败: %v", key, err)
			return err
		}
	}

	setCTCPServerLastMessage(
		"等级辅助配置已保存: fsmId=0x%04X, selectedFruitTypesLen=%d, gradeAccuracy=%d, exitAlarmThreshold=%d, packingWeight1=%.3f, packingWeight2=%.3f, weightStandards=%s, labelerNames=%s",
		uint32(fsmID),
		len(info.SelectedFruitTypes),
		info.GradeAccuracy,
		info.ExitAlarmThreshold,
		info.PackingWeight1,
		info.PackingWeight2,
		formatIntListForLog(info.WeightStandards),
		formatStringListForLog(info.LabelerNames),
	)
	return nil
}

func defaultLevelAuxConfigInfo() LevelAuxConfigInfo {
	return LevelAuxConfigInfo{
		SelectedFruitTypes: cTCPDefaultSelectedFruitTypes,
		GradeAccuracy:      0,
		ExitAlarmThreshold: cTCPDefaultExitAlarmThreshold,
		PackingWeight1:     cTCPDefaultPackingWeight1,
		PackingWeight2:     cTCPDefaultPackingWeight2,
		WeightStandards:    defaultWeightStandards(),
		LabelerNames:       make([]string, 4),
	}
}

func normalizeLevelAuxConfigInfo(info LevelAuxConfigInfo) LevelAuxConfigInfo {
	info.SelectedFruitTypes = strings.TrimSpace(info.SelectedFruitTypes)
	if info.SelectedFruitTypes == "" {
		info.SelectedFruitTypes = cTCPDefaultSelectedFruitTypes
	}
	info.GradeAccuracy = clampInt(info.GradeAccuracy, 0, 6)
	info.ExitAlarmThreshold = clampInt(info.ExitAlarmThreshold, 0, 100)
	if info.PackingWeight1 <= 0 {
		info.PackingWeight1 = cTCPDefaultPackingWeight1
	}
	if info.PackingWeight2 <= 0 {
		info.PackingWeight2 = cTCPDefaultPackingWeight2
	}
	info.WeightStandards = normalizeWeightStandards(info.WeightStandards)
	info.LabelerNames = normalizeLabelerNames(info.LabelerNames)
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

func parseFloatConfigValue(key string, text string, defaultValue float64) (float64, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return defaultValue, nil
	}
	value, err := strconv.ParseFloat(text, 64)
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

func defaultWeightStandards() []int {
	return []int{23, 27, 32, 40, 48, 56, 64, 72, 80, 88, 100, 113, 125, 138, 140, 152, 165}
}

func normalizeWeightStandards(values []int) []int {
	defaults := defaultWeightStandards()
	normalized := make([]int, len(defaults))
	for i := range defaults {
		value := defaults[i]
		if i < len(values) && values[i] > 0 {
			value = values[i]
		}
		normalized[i] = value
	}
	return normalized
}

func parseWeightStandardsConfig(text string) []int {
	defaults := defaultWeightStandards()
	standards := make([]int, len(defaults))
	copy(standards, defaults)
	parts := strings.Split(text, ";")
	for i := 0; i < len(parts) && i < len(standards); i++ {
		part := strings.TrimSpace(parts[i])
		if part == "" {
			continue
		}
		value, err := strconv.Atoi(part)
		if err != nil || value <= 0 {
			continue
		}
		standards[i] = value
	}
	return standards
}

func formatWeightStandardsConfig(values []int) string {
	standards := normalizeWeightStandards(values)
	var builder strings.Builder
	for _, value := range standards {
		builder.WriteString(strconv.Itoa(value))
		builder.WriteString(";")
	}
	return builder.String()
}

func normalizeLabelerNames(values []string) []string {
	normalized := make([]string, 4)
	for i := 0; i < len(normalized) && i < len(values); i++ {
		normalized[i] = strings.TrimSpace(values[i])
	}
	return normalized
}

func formatIntListForLog(values []int) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, strconv.Itoa(value))
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func formatStringListForLog(values []string) string {
	return "[" + strings.Join(values, ",") + "]"
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
