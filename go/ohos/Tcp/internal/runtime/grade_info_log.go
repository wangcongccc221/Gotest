package tcp

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"gotest/ohos/Tcp/events"
	"gotest/ohos/Tcp/state"
)

const (
	stGradeInfoPeriodicLogInterval = 10 * time.Second
	stMotorInfoPeriodicLogInterval = 20 * time.Second
)

var (
	stGradeInfoPeriodicLogMu  sync.Mutex
	stGradeInfoPeriodicCancel context.CancelFunc

	stMotorInfoPeriodicLogMu  sync.Mutex
	stMotorInfoPeriodicCancel context.CancelFunc
)

// setLastStGradeInfoSnapshot 在收到/发送等级数据时更新，供周期日志读取。
func setLastStGradeInfoSnapshot(grade StGradeInfo) {
	state.SetLatestGradeInfoSnapshot(grade)
}

func latestStGradeInfoSnapshot() (StGradeInfo, bool) {
	return state.LatestGradeInfoSnapshot()
}

// setLastStExitInfosSnapshot 在收到出口本地配置时更新，供日志/排查读取。
func setLastStExitInfosSnapshot(fsmID int32, exitInfos StExitInfos) {
	state.SetLatestExitInfosSnapshot(fsmID, exitInfos)
}

func latestStExitInfosSnapshot() (int32, StExitInfos, bool) {
	return state.LatestExitInfosSnapshot()
}

// setLastStMotorInfoSnapshot 在收到有效电机下发数据时更新，供周期日志读取。
func setLastStMotorInfoSnapshot(fsmID int32, motor StMotorInfo) {
	state.SetLatestMotorInfoSnapshot(fsmID, motor)
}

func latestStMotorInfoSnapshot() (int32, StMotorInfo, bool) {
	return state.LatestMotorInfoSnapshot()
}

// DumpStGradeInfo 格式化 StGradeInfo 全部字段。
// 后续若要改输出格式（只打部分字段、uint8 转字符串等），改这个函数即可。
func DumpStGradeInfo(grade StGradeInfo) string {
	text, err := FormatDataFullJSON(grade)
	if err != nil {
		return fmt.Sprintf("StGradeInfo JSON 序列化失败: %v", err)
	}
	return text
}

// LogStGradeInfo 输出一次 StGradeInfo 全量字段到 CTCP 日志（HiLog 搜「StGradeInfo 全量」）。
func LogStGradeInfo(grade StGradeInfo) {
	appendCTCPLogChunks("StGradeInfo 全量", DumpStGradeInfo(grade))
}

func logQualityParameterFields(source string, destID int32, grade StGradeInfo) {
	colorType := int(grade.ColorType)
	colorTypeBase := colorType & 0x07
	colorPercentMode := (colorType & 0x80) != 0
	events.Info(
		"[QUALITY_PARAMS] %s dest=0x%04X sizeGrade=%d qualityGrade=%d classify=%d colorType=%d(base=%d,percent=%t) colorIntervals=%s colorNames=%s",
		source,
		uint32(destID),
		grade.NSizeGradeNum,
		grade.NQualityGradeNum,
		grade.NClassifyType,
		colorType,
		colorTypeBase,
		colorPercentMode,
		formatQualityInt32Slice(grade.ColorIntervals[:]),
		formatQualityFixedTextSlots(grade.StrColorGradeName[:], cTCPServerMaxTextLength, cTCPServerMaxColorGradeNum),
	)
	events.Info(
		"[QUALITY_PARAMS] %s dest=0x%04X shapeNames=%s shapeFactors=%s",
		source,
		uint32(destID),
		formatQualityFixedTextSlots(grade.StrShapeGradeName[:], cTCPServerMaxTextLength, cTCPServerMinorGradeNum),
		formatQualityFloat32Slice(grade.FShapeFactor[:]),
	)
}

func formatQualityFixedTextSlots(bytes []uint8, slotSize int, maxSlots int) string {
	if slotSize <= 0 || maxSlots <= 0 {
		return "[]"
	}
	var b strings.Builder
	b.WriteByte('[')
	written := 0
	for index := 0; index < maxSlots; index++ {
		start := index * slotSize
		if start >= len(bytes) {
			break
		}
		end := start + slotSize
		if end > len(bytes) {
			end = len(bytes)
		}
		text := qualityFixedTextSlot(bytes[start:end])
		if text == "" {
			continue
		}
		if written > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%d:%q", index+1, text)
		written++
	}
	b.WriteByte(']')
	return b.String()
}

func qualityFixedTextSlot(bytes []uint8) string {
	end := 0
	for end < len(bytes) && bytes[end] != 0 {
		end++
	}
	return strings.TrimSpace(string(bytes[:end]))
}

func formatQualityInt32Slice(values []int32) string {
	var b strings.Builder
	b.WriteByte('[')
	for index, value := range values {
		if index > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%d", value)
	}
	b.WriteByte(']')
	return b.String()
}

func formatQualityFloat32Slice(values []float32) string {
	var b strings.Builder
	b.WriteByte('[')
	for index, value := range values {
		if index > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%.6g", value)
	}
	b.WriteByte(']')
	return b.String()
}

// DumpStMotorInfo 格式化 StMotorInfo 全部字段。
func DumpStMotorInfo(motor StMotorInfo) string {
	text, err := FormatDataFullJSON(motor)
	if err != nil {
		return fmt.Sprintf("StMotorInfo JSON 序列化失败: %v", err)
	}
	return text
}

// LogStMotorInfo 输出一次 StMotorInfo 全量字段到 CTCP 日志（HiLog 搜「StMotorInfo Go结构体」）。
func LogStMotorInfo(motor StMotorInfo) {
	appendCTCPLogChunks("StMotorInfo Go结构体", fmt.Sprintf("%+v", motor))
	appendCTCPLogChunks("StMotorInfo 全量JSON", DumpStMotorInfo(motor))
}

// LogLatestStMotorInfo 输出最近一次缓存的 StMotorInfo；尚无数据时只打提示。
func LogLatestStMotorInfo() {
	_, motor, ok := latestStMotorInfoSnapshot()
	if !ok {
		setCTCPServerLastMessage("StMotorInfo 周期输出: 尚无缓存数据（等待 WebSocket saveMotorInfo）")
		return
	}
	LogStMotorInfo(motor)
}

// LogLatestStGradeInfo 输出最近一次缓存的 StGradeInfo；尚无数据时只打提示。
func LogLatestStGradeInfo() {
	grade, ok := latestStGradeInfoSnapshot()
	if !ok {
		setCTCPServerLastMessage("StGradeInfo 周期输出: 尚无缓存数据（等待 FSM_CMD_CONFIG 或 WebSocket save）")
		return
	}
	LogStGradeInfo(grade)
}

// StartStGradeInfoPeriodicLog 每 10 秒输出一次最近一次 StGradeInfo 全量字段。
func StartStGradeInfoPeriodicLog() {
	StartStGradeInfoPeriodicLogWithInterval(stGradeInfoPeriodicLogInterval)
}

// StartStGradeInfoPeriodicLogWithInterval 按指定间隔周期输出 StGradeInfo。
func StartStGradeInfoPeriodicLogWithInterval(interval time.Duration) {
	if interval <= 0 {
		interval = stGradeInfoPeriodicLogInterval
	}

	stGradeInfoPeriodicLogMu.Lock()
	defer stGradeInfoPeriodicLogMu.Unlock()
	if stGradeInfoPeriodicCancel != nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	stGradeInfoPeriodicCancel = cancel

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		setCTCPServerLastMessage("StGradeInfo 周期输出已启动: 每 %s 打印一次全量字段", interval)
		for {
			select {
			case <-ctx.Done():
				setCTCPServerLastMessage("StGradeInfo 周期输出已停止")
				return
			case <-ticker.C:
				LogLatestStGradeInfo()
			}
		}
	}()
}

// StartStMotorInfoPeriodicLog 每 20 秒输出一次最近一次 StMotorInfo 全量字段。
func StartStMotorInfoPeriodicLog() {
	StartStMotorInfoPeriodicLogWithInterval(stMotorInfoPeriodicLogInterval)
}

// StartStMotorInfoPeriodicLogWithInterval 按指定间隔周期输出 StMotorInfo。
func StartStMotorInfoPeriodicLogWithInterval(interval time.Duration) {
	if interval <= 0 {
		interval = stMotorInfoPeriodicLogInterval
	}

	stMotorInfoPeriodicLogMu.Lock()
	defer stMotorInfoPeriodicLogMu.Unlock()
	if stMotorInfoPeriodicCancel != nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	stMotorInfoPeriodicCancel = cancel

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		setCTCPServerLastMessage("StMotorInfo 周期输出已启动: 每 %s 打印一次全量字段", interval)
		for {
			select {
			case <-ctx.Done():
				setCTCPServerLastMessage("StMotorInfo 周期输出已停止")
				return
			case <-ticker.C:
				LogLatestStMotorInfo()
			}
		}
	}()
}

// StopStGradeInfoPeriodicLog 停止周期输出。
func StopStGradeInfoPeriodicLog() {
	stGradeInfoPeriodicLogMu.Lock()
	cancel := stGradeInfoPeriodicCancel
	stGradeInfoPeriodicCancel = nil
	stGradeInfoPeriodicLogMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// StopStMotorInfoPeriodicLog 停止周期输出。
func StopStMotorInfoPeriodicLog() {
	stMotorInfoPeriodicLogMu.Lock()
	cancel := stMotorInfoPeriodicCancel
	stMotorInfoPeriodicCancel = nil
	stMotorInfoPeriodicLogMu.Unlock()
	if cancel != nil {
		cancel()
	}
}
