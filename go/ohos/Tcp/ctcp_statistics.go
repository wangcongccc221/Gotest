package tcp

import (
	"fmt"
	"math"
	"sync"
	"time"
)

const (
	cTCPStStatisticsSpeedPublishInterval = time.Second
	cTCPStStatisticsSpeedStaleInterval   = 2500 * time.Millisecond
	cTCPHomeBatchStatsLogInterval        = 5 * time.Second
)

type cTCPStStatisticsSpeedState struct {
	Latest    StStatistics
	LatestAt  time.Time
	HasLatest bool
}

var (
	cTCPStStatisticsSpeedMu     sync.Mutex
	cTCPStStatisticsSpeedBySys  = make(map[int32]*cTCPStStatisticsSpeedState)
	cTCPStStatisticsSpeedStop   chan struct{}
	cTCPHomeBatchStatsLastLogAt time.Time
)

func StartStStatisticsSpeedPublisher() {
	StartStStatisticsSpeedPublisherWithInterval(cTCPStStatisticsSpeedPublishInterval)
}

func StartStStatisticsSpeedPublisherWithInterval(interval time.Duration) {
	if interval <= 0 {
		interval = cTCPStStatisticsSpeedPublishInterval
	}

	cTCPStStatisticsSpeedMu.Lock()
	if cTCPStStatisticsSpeedStop != nil {
		cTCPStStatisticsSpeedMu.Unlock()
		return
	}
	stop := make(chan struct{})
	cTCPStStatisticsSpeedStop = stop
	cTCPStStatisticsSpeedMu.Unlock()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		setCTCPServerLastMessage("CTCP StStatistics 分选速度后端计算已启动: 每 %s 推送一次", interval)
		for {
			select {
			case <-stop:
				setCTCPServerLastMessage("CTCP StStatistics 分选速度后端计算已停止")
				return
			case now := <-ticker.C:
				publishLatestStStatisticsSpeed(now)
			}
		}
	}()
}

func StopStStatisticsSpeedPublisher() {
	cTCPStStatisticsSpeedMu.Lock()
	stop := cTCPStStatisticsSpeedStop
	cTCPStStatisticsSpeedStop = nil
	cTCPStStatisticsSpeedBySys = make(map[int32]*cTCPStStatisticsSpeedState)
	cTCPHomeBatchStatsLastLogAt = time.Time{}
	cTCPStStatisticsSpeedMu.Unlock()
	if stop != nil {
		close(stop)
	}
}

func resetStStatisticsCacheAfterEndProcess() {
	cTCPStStatisticsSpeedMu.Lock()
	cTCPStStatisticsSpeedBySys = make(map[int32]*cTCPStStatisticsSpeedState)
	cTCPStStatisticsSpeedMu.Unlock()
}

func sanitizeStStatisticsSpeed(speed int32) int32 {
	if speed <= 0 {
		return 0
	}
	return speed
}

func calculateStStatisticsSpeedFromPulse(state StStatistics) (int32, bool) {
	if state.NInterval <= 0 {
		return 0, true
	}
	if state.NPulseInterval > 2000 {
		return 0, true
	}
	if state.NPulseInterval > 0 {
		speed := math.Round(60000.0 / float64(state.NPulseInterval))
		if speed <= 0 {
			return 0, true
		}
		if speed > 2147483647 {
			return 2147483647, true
		}
		return int32(speed), true
	}
	return 0, false
}

func resolveStStatisticsDisplaySpeed(state StStatistics) int32 {
	if speed, ok := calculateStStatisticsSpeedFromPulse(state); ok {
		return speed
	}
	return sanitizeStStatisticsSpeed(state.NIntervalSumperminute)
}

func cacheStStatisticsForSpeed(state StStatistics, receivedAt time.Time) {
	subsysID := state.NSubsysId
	if subsysID <= 0 {
		subsysID = 1
	}

	cTCPStStatisticsSpeedMu.Lock()
	entry := cTCPStStatisticsSpeedBySys[subsysID]
	if entry == nil {
		entry = &cTCPStStatisticsSpeedState{}
		cTCPStStatisticsSpeedBySys[subsysID] = entry
	}
	state.NIntervalSumperminute = resolveStStatisticsDisplaySpeed(state)
	entry.Latest = state
	entry.LatestAt = receivedAt
	entry.HasLatest = true
	cTCPStStatisticsSpeedMu.Unlock()
	maybeResumeRealtimeSaveAfterStatisticsReset()
	maybeSaveRealtimeStatistics(receivedAt)
}

func latestStStatisticsSpeedSnapshots(now time.Time) []StStatistics {
	cTCPStStatisticsSpeedMu.Lock()
	defer cTCPStStatisticsSpeedMu.Unlock()

	snapshots := make([]StStatistics, 0, len(cTCPStStatisticsSpeedBySys))
	for _, entry := range cTCPStStatisticsSpeedBySys {
		if entry == nil || !entry.HasLatest {
			continue
		}
		latest := entry.Latest
		if now.Sub(entry.LatestAt) > cTCPStStatisticsSpeedStaleInterval {
			latest.NIntervalSumperminute = 0
		}
		snapshots = append(snapshots, latest)
	}
	return snapshots
}

func shouldLogHomeBatchStats(now time.Time) bool {
	if now.IsZero() {
		return false
	}
	return cTCPHomeBatchStatsLastLogAt.IsZero() ||
		now.Sub(cTCPHomeBatchStatsLastLogAt) >= cTCPHomeBatchStatsLogInterval
}

func formatHomeBatchStatsLine(stats StStatistics) string {
	subsysID := stats.NSubsysId
	if subsysID <= 0 {
		subsysID = 1
	}
	exitWeightTotal := sumHomeStatsUint64(stats.NExitWeightCount[:])
	gradeWeightTotal := sumHomeStatsUint64(stats.NWeightGradeCount[:])
	return fmt.Sprintf(
		"subsys=%d, 本批个数(NChannelTotalCount)=%d, 本批重量=%.0f, NChannelWeightCount=%d, sum(NExitWeightCount)=%d, sum(NWeightGradeCount)=%d",
		subsysID,
		homeStatsTotalCount(stats),
		homeStatsTotalWeight(stats),
		stats.NChannelWeightCount,
		exitWeightTotal,
		gradeWeightTotal,
	)
}

func logLatestHomeBatchStats(now time.Time, snapshots []StStatistics) {
	cTCPStStatisticsSpeedMu.Lock()
	if !shouldLogHomeBatchStats(now) {
		cTCPStStatisticsSpeedMu.Unlock()
		return
	}
	cTCPHomeBatchStatsLastLogAt = now
	cTCPStStatisticsSpeedMu.Unlock()

	if len(snapshots) == 0 {
		setCTCPServerLastMessage(
			"CTCP WAM本批统计 5秒打印: 尚无 StStatistics 快照（本批个数字段=NChannelTotalCount, 本批重量字段=NChannelWeightCount，回退=sum(NExitWeightCount)/sum(NWeightGradeCount)）",
		)
		return
	}
	for _, stats := range snapshots {
		setCTCPServerLastMessage("CTCP WAM本批统计 5秒打印: %s", formatHomeBatchStatsLine(stats))
	}
}

func publishLatestStStatisticsSpeed(now time.Time) {
	snapshots := latestStStatisticsSpeedSnapshots(now)
	logLatestHomeBatchStats(now, snapshots)
	for _, state := range snapshots {
		stateJSON, jsonErr := FormatDataFullJSON(state)
		if stateJSON == "" || jsonErr != nil {
			setCTCPServerLastMessage("CTCP StStatistics 后端分选速度 JSON 生成失败: %v", jsonErr)
			continue
		}
		if err := PublishWebSocketJSON(webSocketTopicStatistics, stateJSON); err != nil {
			setCTCPServerLastMessage("CTCP StStatistics 后端分选速度 WebSocket 推送失败: %v", err)
		}
	}
	publishLatestHomeStats(now)
}

func normalizeStStatisticsSubsysID(srcID int32, payloadSubsysID int32) int32 {
	if idx := getSubsysIndex(srcID); idx >= 0 && idx < cTCPServerMaxSubsysNum {
		return int32(idx + 1)
	}
	if payloadSubsysID >= 1 && payloadSubsysID <= cTCPServerMaxSubsysNum {
		return payloadSubsysID
	}
	if idx := getSubsysIndex(payloadSubsysID); idx >= 0 && idx < cTCPServerMaxSubsysNum {
		return int32(idx + 1)
	}
	return 1
}
