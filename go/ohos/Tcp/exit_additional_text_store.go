package tcp

import (
	"fmt"
	"sync"

	"gotest/ohos/database"
)

const cTCPExitAdditionalTextWireSize = 100

type ExitAdditionalTextInfo struct {
	AdditionalTexts [cTCP48MaxExitNum]string `json:"additionalTexts"`
}

var (
	lastExitAdditionalTextInfoSnapshot   ExitAdditionalTextInfo
	lastExitAdditionalTextInfoFSMID      int32
	lastExitAdditionalTextInfoSnapshotOK bool
	lastExitAdditionalTextInfoSnapshotMu sync.RWMutex
)

func LoadExitAdditionalTextInfoFromLocalConfig() {
	info, err := ReadExitAdditionalTextInfoFromLocalConfig()
	if err != nil {
		setCTCPServerLastMessage("出口附加信息配置读取失败: %v", err)
		return
	}

	setLastExitAdditionalTextInfoSnapshot(0, info)
	setCTCPServerLastMessage("出口附加信息配置已加载")
}

func ReadExitAdditionalTextInfoFromLocalConfig() (ExitAdditionalTextInfo, error) {
	values := make(map[string]string, cTCP48MaxExitNum)
	for i := 0; i < cTCP48MaxExitNum; i++ {
		key := exitAdditionalTextConfigName(i)
		text, err := database.GetConfigValue(key)
		if err != nil {
			return ExitAdditionalTextInfo{}, fmt.Errorf("read %s: %w", key, err)
		}
		values[key] = text
	}

	info := parseExitAdditionalTextInfoConfigValues(values)
	return info, nil
}

func SaveExitAdditionalTextInfoToLocalConfig(fsmID int32, info ExitAdditionalTextInfo) error {
	values := exitAdditionalTextInfoConfigValues(info)
	for i := 0; i < cTCP48MaxExitNum; i++ {
		key := exitAdditionalTextConfigName(i)
		if err := database.SaveConfigValue(key, values[key]); err != nil {
			setCTCPServerLastMessage("出口附加信息保存失败: key=%s, err=%v", key, err)
			return err
		}
	}

	setCTCPServerLastMessage("出口附加信息配置已保存: fsmId=0x%04X", uint32(fsmID))
	return nil
}

func defaultExitAdditionalTextInfo() ExitAdditionalTextInfo {
	return ExitAdditionalTextInfo{}
}

func exitAdditionalTextInfoConfigValues(info ExitAdditionalTextInfo) map[string]string {
	values := make(map[string]string, cTCP48MaxExitNum)
	for i, text := range info.AdditionalTexts {
		values[exitAdditionalTextConfigName(i)] = text
	}
	return values
}

func parseExitAdditionalTextInfoConfigValues(values map[string]string) ExitAdditionalTextInfo {
	info := defaultExitAdditionalTextInfo()
	for i := 0; i < cTCP48MaxExitNum; i++ {
		info.AdditionalTexts[i] = values[exitAdditionalTextConfigName(i)]
	}
	return info
}

func exitAdditionalTextConfigName(exitIndex int) string {
	return fmt.Sprintf("出口%d附加信息", exitIndex+1)
}

func applyExitAdditionalTextInfoToBroadcastData(data *StExitAdditionalTextData, info ExitAdditionalTextInfo) {
	if data == nil {
		return
	}
	for i := range data.Additionaldata {
		data.Additionaldata[i] = 0
	}
	for i, text := range info.AdditionalTexts {
		start := i * cTCPExitAdditionalTextWireSize
		end := start + cTCPExitAdditionalTextWireSize
		copy(data.Additionaldata[start:end], []byte(text))
	}
}

func setLastExitAdditionalTextInfoSnapshot(fsmID int32, info ExitAdditionalTextInfo) {
	lastExitAdditionalTextInfoSnapshotMu.Lock()
	lastExitAdditionalTextInfoSnapshot = info
	lastExitAdditionalTextInfoFSMID = fsmID
	lastExitAdditionalTextInfoSnapshotOK = true
	lastExitAdditionalTextInfoSnapshotMu.Unlock()
}

func latestExitAdditionalTextInfoSnapshot() (int32, ExitAdditionalTextInfo, bool) {
	lastExitAdditionalTextInfoSnapshotMu.RLock()
	defer lastExitAdditionalTextInfoSnapshotMu.RUnlock()
	return lastExitAdditionalTextInfoFSMID, lastExitAdditionalTextInfoSnapshot, lastExitAdditionalTextInfoSnapshotOK
}
