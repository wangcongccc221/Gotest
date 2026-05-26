package tcp

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"gotest/ohos/database"
)

const cTCPExitInfosConfigName = "ExitInfos"

type stExitInfos48ConfigValue struct {
	ExitInfos       string `json:"ExitInfos"`
	ExitControlMode string `json:"ExitControlMode"`
	ExitBoxType     []int  `json:"ExitBoxType"`
}

func LoadStExitInfosFromLocalConfig() {
	text, err := database.GetConfigValue(cTCPExitInfosConfigName)
	if err != nil {
		setCTCPServerLastMessage("StExitInfos 本地配置读取失败: %v", err)
		return
	}
	if strings.TrimSpace(text) == "" {
		setCTCPServerLastMessage("StExitInfos 本地配置为空: key=%s", cTCPExitInfosConfigName)
		return
	}

	exitInfos, err := parseStExitInfosConfigValue(text)
	if err != nil {
		setCTCPServerLastMessage("StExitInfos 本地配置解析失败: %v", err)
		return
	}

	setLastStExitInfosSnapshot(0, exitInfos)
	setCTCPServerLastMessage("StExitInfos 本地配置已加载: key=%s", cTCPExitInfosConfigName)
}

func SaveStExitInfosToLocalConfig(fsmID int32, exitInfos StExitInfos) error {
	payload, err := formatStExitInfosConfigValue(exitInfos)
	if err != nil {
		setCTCPServerLastMessage("StExitInfos 本地配置序列化失败: %v", err)
		return err
	}
	if err := database.SaveConfigValue(cTCPExitInfosConfigName, payload); err != nil {
		setCTCPServerLastMessage("StExitInfos 本地配置保存失败: %v", err)
		return err
	}
	setCTCPServerLastMessage("StExitInfos 本地配置已保存: key=%s, fsmId=0x%04X", cTCPExitInfosConfigName, uint32(fsmID))
	return nil
}

func parseStExitInfosConfigValue(text string) (StExitInfos, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return defaultStExitInfos(), nil
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(text), &raw); err != nil {
		return StExitInfos{}, err
	}

	if _, ok := raw["exitInfos"]; ok {
		var data webSocketExitInfosData
		if err := json.Unmarshal([]byte(text), &data); err != nil {
			return StExitInfos{}, err
		}
		return data.ExitInfos, nil
	}

	if _, ok := raw["ExitInfos"]; ok {
		return parseStExitInfos48ConfigValue(text)
	}
	if _, ok := raw["ExitControlMode"]; ok {
		return parseStExitInfos48ConfigValue(text)
	}
	if _, ok := raw["ExitBoxType"]; ok {
		return parseStExitInfos48ConfigValue(text)
	}

	var exitInfos StExitInfos
	if err := json.Unmarshal([]byte(text), &exitInfos); err != nil {
		return StExitInfos{}, err
	}
	return exitInfos, nil
}

func parseStExitInfos48ConfigValue(text string) (StExitInfos, error) {
	var value stExitInfos48ConfigValue
	if err := json.Unmarshal([]byte(text), &value); err != nil {
		return StExitInfos{}, err
	}

	exitInfos := defaultStExitInfos()
	if strings.TrimSpace(value.ExitInfos) != "" {
		bytes, err := base64.StdEncoding.DecodeString(value.ExitInfos)
		if err != nil {
			return StExitInfos{}, fmt.Errorf("decode ExitInfos: %w", err)
		}
		copy(exitInfos.ExitName[:], bytes)
	}
	if strings.TrimSpace(value.ExitControlMode) != "" {
		bytes, err := base64.StdEncoding.DecodeString(value.ExitControlMode)
		if err != nil {
			return StExitInfos{}, fmt.Errorf("decode ExitControlMode: %w", err)
		}
		copy(exitInfos.ExitControlMode[:], bytes)
	}
	for i, item := range value.ExitBoxType {
		if i >= len(exitInfos.ExitBoxType) {
			break
		}
		if item < 0 {
			item = 0
		}
		if item > 255 {
			item = 255
		}
		exitInfos.ExitBoxType[i] = uint8(item)
	}
	return exitInfos, nil
}

func formatStExitInfosConfigValue(exitInfos StExitInfos) (string, error) {
	exitBoxType := make([]int, len(exitInfos.ExitBoxType))
	for i, item := range exitInfos.ExitBoxType {
		exitBoxType[i] = int(item)
	}
	value := stExitInfos48ConfigValue{
		ExitInfos:       base64.StdEncoding.EncodeToString(exitInfos.ExitName[:]),
		ExitControlMode: base64.StdEncoding.EncodeToString(exitInfos.ExitControlMode[:]),
		ExitBoxType:     exitBoxType,
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func defaultStExitInfos() StExitInfos {
	var exitInfos StExitInfos
	for i := range exitInfos.ExitControlMode {
		exitInfos.ExitControlMode[i] = 2
	}
	return exitInfos
}

func mergeStExitInfosBoxTypeUpdate(base StExitInfos, incoming StExitInfos) StExitInfos {
	next := base
	copy(next.ExitBoxType[:], incoming.ExitBoxType[:])
	return next
}
