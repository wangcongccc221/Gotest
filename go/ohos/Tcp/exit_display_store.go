package tcp

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"gotest/ohos/database"
)

const (
	cTCPExitDisplayEnabledConfigName = "ExitDisplay"
	cTCPExitDisplayTypeConfigName    = "出口显示名称类型"
	cTCPExitDisplayNameWireSize      = 20
)

type ExitDisplayInfo struct {
	DisplayType  int64                    `json:"displayType"`
	DisplayNames [cTCP48MaxExitNum]string `json:"displayNames"`
}

var (
	lastExitDisplayInfoSnapshot   ExitDisplayInfo
	lastExitDisplayInfoFSMID      int32
	lastExitDisplayInfoSnapshotOK bool
	lastExitDisplayInfoSnapshotMu sync.RWMutex
	exitDisplayBroadcastEnabled   bool
	exitDisplayBroadcastEnabledMu sync.RWMutex
)

func LoadExitDisplayInfoFromLocalConfig() {
	info, err := ReadExitDisplayInfoFromLocalConfig()
	if err != nil {
		setCTCPServerLastMessage("出口显示名称配置读取失败: %v", err)
		return
	}

	setLastExitDisplayInfoSnapshot(0, info)
	setCTCPServerLastMessage("出口显示名称配置已加载: displayType=%d", info.DisplayType)
}

func LoadExitDisplayBroadcastEnabledFromLocalConfig() {
	enabled, err := ReadExitDisplayBroadcastEnabledFromLocalConfig()
	if err != nil {
		setCTCPServerLastMessage("出口屏显示开关读取失败: %v", err)
		return
	}
	setExitDisplayBroadcastEnabled(enabled)
	setCTCPServerLastMessage("出口屏显示开关已加载: enabled=%t", enabled)
}

func ReadExitDisplayBroadcastEnabledFromLocalConfig() (bool, error) {
	text, err := database.GetConfigValue(cTCPExitDisplayEnabledConfigName)
	if err != nil {
		return false, fmt.Errorf("read %s: %w", cTCPExitDisplayEnabledConfigName, err)
	}
	return parseExitDisplayEnabledConfigValue(text), nil
}

func SaveExitDisplayBroadcastEnabledToLocalConfig(enabled bool) error {
	value := "false"
	if enabled {
		value = "true"
	}
	if err := database.SaveConfigValue(cTCPExitDisplayEnabledConfigName, value); err != nil {
		setCTCPServerLastMessage("出口屏显示开关保存失败: enabled=%t, err=%v", enabled, err)
		return err
	}
	setExitDisplayBroadcastEnabled(enabled)
	setCTCPServerLastMessage("出口屏显示开关已保存: enabled=%t", enabled)
	return nil
}

func parseExitDisplayEnabledConfigValue(text string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(text)), "true")
}

func ReadExitDisplayInfoFromLocalConfig() (ExitDisplayInfo, error) {
	values := make(map[string]string, cTCP48MaxExitNum+1)

	text, err := database.GetConfigValue(cTCPExitDisplayTypeConfigName)
	if err != nil {
		return ExitDisplayInfo{}, fmt.Errorf("read %s: %w", cTCPExitDisplayTypeConfigName, err)
	}
	values[cTCPExitDisplayTypeConfigName] = text

	for i := 0; i < cTCP48MaxExitNum; i++ {
		key := exitDisplayNameConfigName(i)
		text, err := database.GetConfigValue(key)
		if err != nil {
			return ExitDisplayInfo{}, fmt.Errorf("read %s: %w", key, err)
		}
		values[key] = text
	}

	info, err := parseExitDisplayInfoConfigValues(values)
	if err != nil {
		return ExitDisplayInfo{}, err
	}
	return info, nil
}

func SaveExitDisplayInfoToLocalConfig(fsmID int32, info ExitDisplayInfo) error {
	values := exitDisplayInfoConfigValues(info)
	if err := database.SaveConfigValue(cTCPExitDisplayTypeConfigName, values[cTCPExitDisplayTypeConfigName]); err != nil {
		setCTCPServerLastMessage("出口显示名称类型保存失败: %v", err)
		return err
	}

	for i := 0; i < cTCP48MaxExitNum; i++ {
		key := exitDisplayNameConfigName(i)
		if err := database.SaveConfigValue(key, values[key]); err != nil {
			setCTCPServerLastMessage("出口显示名称保存失败: key=%s, err=%v", key, err)
			return err
		}
	}

	setCTCPServerLastMessage("出口显示名称配置已保存: fsmId=0x%04X, displayType=%d", uint32(fsmID), info.DisplayType)
	return nil
}

func defaultExitDisplayInfo() ExitDisplayInfo {
	return ExitDisplayInfo{}
}

func exitDisplayInfoConfigValues(info ExitDisplayInfo) map[string]string {
	values := make(map[string]string, cTCP48MaxExitNum+1)
	values[cTCPExitDisplayTypeConfigName] = strconv.FormatInt(info.DisplayType, 10)
	for i, name := range info.DisplayNames {
		values[exitDisplayNameConfigName(i)] = name
	}
	return values
}

func parseExitDisplayInfoConfigValues(values map[string]string) (ExitDisplayInfo, error) {
	info := defaultExitDisplayInfo()
	displayType, err := parseExitDisplayTypeConfigValue(values[cTCPExitDisplayTypeConfigName])
	if err != nil {
		return ExitDisplayInfo{}, err
	}
	info.DisplayType = displayType

	for i := 0; i < cTCP48MaxExitNum; i++ {
		info.DisplayNames[i] = values[exitDisplayNameConfigName(i)]
	}
	return info, nil
}

func parseExitDisplayTypeConfigValue(text string) (int64, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, nil
	}
	value, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", cTCPExitDisplayTypeConfigName, err)
	}
	return value, nil
}

func exitDisplayNameConfigName(exitIndex int) string {
	return fmt.Sprintf("出口%d显示名称", exitIndex+1)
}

func exitDisplayNameEnabled(displayType int64, exitIndex int) bool {
	if exitIndex < 0 || exitIndex >= cTCP48MaxExitNum {
		return false
	}
	return displayType&(int64(1)<<uint(exitIndex)) != 0
}

func setExitDisplayNameEnabled(displayType int64, exitIndex int, enabled bool) int64 {
	if exitIndex < 0 || exitIndex >= cTCP48MaxExitNum {
		return displayType
	}
	mask := int64(1) << uint(exitIndex)
	if enabled {
		return displayType | mask
	}
	return displayType &^ mask
}

func applyExitDisplayInfoToBroadcastSysConfig(config *StBroadcastSysConfig, info ExitDisplayInfo) {
	if config == nil {
		return
	}
	config.ExitDisplayType = info.DisplayType
	for i := range config.StrDisplayName {
		config.StrDisplayName[i] = 0
	}
	for i, name := range info.DisplayNames {
		start := i * cTCPExitDisplayNameWireSize
		end := start + cTCPExitDisplayNameWireSize
		writeGBKFixedTextSlot(config.StrDisplayName[start:end], name)
	}
}

func setLastExitDisplayInfoSnapshot(fsmID int32, info ExitDisplayInfo) {
	lastExitDisplayInfoSnapshotMu.Lock()
	lastExitDisplayInfoSnapshot = info
	lastExitDisplayInfoFSMID = fsmID
	lastExitDisplayInfoSnapshotOK = true
	lastExitDisplayInfoSnapshotMu.Unlock()
}

func latestExitDisplayInfoSnapshot() (int32, ExitDisplayInfo, bool) {
	lastExitDisplayInfoSnapshotMu.RLock()
	defer lastExitDisplayInfoSnapshotMu.RUnlock()
	return lastExitDisplayInfoFSMID, lastExitDisplayInfoSnapshot, lastExitDisplayInfoSnapshotOK
}

func setExitDisplayBroadcastEnabled(enabled bool) {
	exitDisplayBroadcastEnabledMu.Lock()
	exitDisplayBroadcastEnabled = enabled
	exitDisplayBroadcastEnabledMu.Unlock()
}

func isExitDisplayBroadcastEnabled() bool {
	exitDisplayBroadcastEnabledMu.RLock()
	defer exitDisplayBroadcastEnabledMu.RUnlock()
	return exitDisplayBroadcastEnabled
}
