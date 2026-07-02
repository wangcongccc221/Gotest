package tcp

import (
	"encoding/json"

	"gotest/ohos/Tcp/state"
)

// LastCTCPStGlobalFullJSON 返回最近一次 save 写入的 StGlobal 全量 JSON；未调用过 save 或失败清空后为 ""。
func LastCTCPStGlobalFullJSON() string {
	return state.LastStGlobalFullJSON()
}

func setCTCPLastStGlobalFullJSON(jsonText string) {
	state.SetLastStGlobalFullJSON(jsonText)
}

func saveCTCPStGlobalFullJSON(global StGlobal) string {
	full, err := FormatStGlobalFullJSON(global)
	if err != nil {
		setCTCPLastStGlobalFullJSON("")
	} else {
		setCTCPLastStGlobalFullJSON(full)
	}
	if err != nil {
		return ""
	}
	return full
}

type stGlobalFrontendJSON struct {
	StGlobal
	GradeNameTexts webSocketGradeNameTexts `json:"GradeNameTexts,omitempty"`
}

func FormatStGlobalFullJSON(global StGlobal) (string, error) {
	return FormatDataFullJSON(stGlobalFrontendJSON{
		StGlobal:       stGlobalFromGBKWireForFrontend(global),
		GradeNameTexts: gradeNameTextsFromGBKWire(global.Grade),
	})
}

// 解析传入的data转换成字符串  前提是 结构体中的字段开头必须是大写的 才可以用这个 不然需要导出
func FormatDataFullJSON[T any](data T) (string, error) {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
