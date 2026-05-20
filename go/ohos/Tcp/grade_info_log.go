package tcp

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const stGradeInfoPeriodicLogInterval = 10 * time.Second

var (
	lastStGradeInfoSnapshot   StGradeInfo
	lastStGradeInfoSnapshotOK bool
	lastStGradeInfoSnapshotMu sync.RWMutex

	stGradeInfoPeriodicLogMu sync.Mutex
	stGradeInfoPeriodicCancel context.CancelFunc
)

// setLastStGradeInfoSnapshot 在收到/发送等级数据时更新，供周期日志读取。
func setLastStGradeInfoSnapshot(grade StGradeInfo) {
	lastStGradeInfoSnapshotMu.Lock()
	lastStGradeInfoSnapshot = grade
	lastStGradeInfoSnapshotOK = true
	lastStGradeInfoSnapshotMu.Unlock()
}

func latestStGradeInfoSnapshot() (StGradeInfo, bool) {
	lastStGradeInfoSnapshotMu.RLock()
	defer lastStGradeInfoSnapshotMu.RUnlock()
	return lastStGradeInfoSnapshot, lastStGradeInfoSnapshotOK
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
