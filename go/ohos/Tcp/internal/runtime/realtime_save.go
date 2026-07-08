package tcp

import (
	"strings"
	"sync"
	"time"

	"gotest/ohos/database"
)

const (
	cTCPRealtimeSaveInterval       = 3 * time.Second
	cTCPRealtimeSaveErrorLogPeriod = 30 * time.Second
	// 数据清零标记的有效窗口:FSM 收到清零后计数器立即归零,下一个3秒落库周期内必然观测到递减
	cTCPRealtimeSaveClearWindow = 90 * time.Second
)

type realtimeSaveAggregate struct {
	GradeCount       [256]uint64
	WeightGradeCount [256]uint64
	BoxGradeCount    [256]int64
	ExitCount        [MAX_EXIT_NUM]uint64
	ExitWeightCount  [MAX_EXIT_NUM]uint64
	TotalExitCount   uint64
	SystemCount      [cTCPServerMaxSubsysNum]uint64
	SystemWeight     [cTCPServerMaxSubsysNum]uint64
	TotalCount       uint64
	TotalWeight      float64
	TotalCupNum      uint64
	SortSpeed        float64
	SubsysNum        int
	HasStats         bool
}

type realtimeSaveProcessSnapshot struct {
	TotalExitCount uint64
	TotalCount     uint64
	TotalCupNum    uint64
	TotalWeight    float64
	At             time.Time
	HasPrev        bool
}

var (
	realtimeSaveMu             sync.Mutex
	realtimeSaveLatestGlobal   StGlobal
	realtimeSaveHasGlobal      bool
	realtimeSaveLastAt         time.Time
	realtimeSaveInFlight       bool
	realtimeSaveLastErr        string
	realtimeSaveLastErrAt      time.Time
	realtimeSaveLastSkip       string
	realtimeSaveLastSkipAt     time.Time
	realtimeSaveProcessHistory realtimeSaveProcessSnapshot
	realtimeSavePausedAfterEnd bool
	realtimeSaveEndBaseline    database.RealtimeFruitSaveInput
	realtimeSaveHasEndBaseline bool
	realtimeSaveClearRequestedAt time.Time
)

func cacheRealtimeSaveGlobalConfig(stg StGlobal) {
	realtimeSaveMu.Lock()
	realtimeSaveLatestGlobal = stg
	realtimeSaveHasGlobal = true
	realtimeSaveMu.Unlock()
}

// markRealtimeSaveClearRequested 记录前端刚下发了数据清零指令。
// 对齐 48 数据清零(HC_SERVICE_CMD_CLEAR)语义:窗口期内的计数器递减按"批内清零"处理,
// 不结束批次——批次和原开始时间保留,数据缩回重新累计;结批只属于结束加工/设备重启。
func markRealtimeSaveClearRequested() {
	realtimeSaveMu.Lock()
	realtimeSaveClearRequestedAt = time.Now()
	// 清零后产量/效率的增量基线作废,防止拿清零前的累计算出异常增量
	realtimeSaveProcessHistory = realtimeSaveProcessSnapshot{}
	realtimeSaveMu.Unlock()

	// 对齐48服务端 CLEAR:立即清空当前批次库内数据,不等下一次统计落库(设备停机时也生效)
	go func() {
		if customerID, err := database.ClearCurrentFruitBatchData(); err != nil {
			setCTCPServerLastMessage("clearData: 清空当前批次数据失败: %v", err)
		} else if customerID > 0 {
			setCTCPServerLastMessage("clearData: 当前批次(CustomerID=%d)库内数据已清零,批次保留", customerID)
		}
	}()
}

func realtimeSaveClearRecentlyRequested() bool {
	realtimeSaveMu.Lock()
	at := realtimeSaveClearRequestedAt
	realtimeSaveMu.Unlock()
	return !at.IsZero() && time.Since(at) <= cTCPRealtimeSaveClearWindow
}

func resetRealtimeSaveState() {
	realtimeSaveMu.Lock()
	realtimeSaveLatestGlobal = StGlobal{}
	realtimeSaveHasGlobal = false
	realtimeSaveLastAt = time.Time{}
	realtimeSaveInFlight = false
	realtimeSaveLastErr = ""
	realtimeSaveLastErrAt = time.Time{}
	realtimeSaveLastSkip = ""
	realtimeSaveLastSkipAt = time.Time{}
	realtimeSaveProcessHistory = realtimeSaveProcessSnapshot{}
	realtimeSavePausedAfterEnd = false
	realtimeSaveEndBaseline = database.RealtimeFruitSaveInput{}
	realtimeSaveHasEndBaseline = false
	realtimeSaveClearRequestedAt = time.Time{}
	realtimeSaveMu.Unlock()
}

func maybeSaveRealtimeStatistics(now time.Time) {
	realtimeSaveMu.Lock()
	if realtimeSaveInFlight {
		realtimeSaveMu.Unlock()
		return
	}
	if !realtimeSaveHasGlobal {
		realtimeSaveMu.Unlock()
		logRealtimeSaveSkip("等待 StGlobal/FSM_CMD_CONFIG，暂不保存")
		return
	}
	pausedAfterEnd := realtimeSavePausedAfterEnd
	if !realtimeSaveLastAt.IsZero() && now.Sub(realtimeSaveLastAt) < cTCPRealtimeSaveInterval {
		realtimeSaveMu.Unlock()
		return
	}
	realtimeSaveLastAt = now
	realtimeSaveInFlight = true
	realtimeSaveMu.Unlock()

	input, ok, reason := buildRealtimeSaveInput(now)
	if !ok {
		finishRealtimeSaveSkip(reason)
		return
	}
	if pausedAfterEnd {
		deltaInput, hasDelta := realtimeSaveDeltaAfterEnd(input)
		if !hasDelta {
			finishRealtimeSaveSkip("结束加工后等待下位机统计清零或新批次增量，暂不保存")
			return
		}
		input = deltaInput
		resumeRealtimeSaveAfterEnd("StStatistics increased after end process; saving delta as new batch")
	}

	// 对齐48:清零指令窗口期内的计数回落按批内清零落库,不拆批次
	input.ClearWithinBatch = realtimeSaveClearRecentlyRequested()

	go func() {
		_, err := database.SaveRealtimeFruitInfo(input)
		finishRealtimeSave(err)
	}()
}

func finishRealtimeSaveSkip(reason string) {
	realtimeSaveMu.Lock()
	realtimeSaveInFlight = false
	realtimeSaveMu.Unlock()
	logRealtimeSaveSkip(reason)
}

func finishRealtimeSave(err error) {
	realtimeSaveMu.Lock()
	realtimeSaveInFlight = false
	if err != nil {
		errText := err.Error()
		shouldLog := errText != realtimeSaveLastErr || realtimeSaveLastErrAt.IsZero() || time.Since(realtimeSaveLastErrAt) >= cTCPRealtimeSaveErrorLogPeriod
		if shouldLog {
			realtimeSaveLastErr = errText
			realtimeSaveLastErrAt = time.Now()
		}
		realtimeSaveMu.Unlock()
		if shouldLog {
			setCTCPServerLastMessage("CTCP realtime save failed: %v", err)
		}
		return
	}
	realtimeSaveLastErr = ""
	realtimeSaveLastErrAt = time.Time{}
	realtimeSaveMu.Unlock()
}

func markRealtimeSaveProcessEnded() {
	baseline, hasBaseline, _ := buildRealtimeSaveInput(cTCPNow())
	realtimeSaveMu.Lock()
	realtimeSaveProcessHistory = realtimeSaveProcessSnapshot{}
	realtimeSaveLastAt = time.Time{}
	realtimeSavePausedAfterEnd = true
	realtimeSaveEndBaseline = baseline
	realtimeSaveHasEndBaseline = hasBaseline
	realtimeSaveMu.Unlock()
	resetHomeStatsHistoryAfterEndProcess()
	resetStStatisticsCacheAfterEndProcess()
}

func maybeResumeRealtimeSaveAfterStatisticsReset() {
	realtimeSaveMu.Lock()
	paused := realtimeSavePausedAfterEnd
	realtimeSaveMu.Unlock()
	if !paused {
		return
	}

	statsList := latestRealtimeSaveStatisticsSnapshots()
	if len(statsList) == 0 {
		return
	}
	for _, stats := range statsList {
		if homeStatsTotalCount(stats) > 0 {
			return
		}
	}

	realtimeSaveMu.Lock()
	if realtimeSavePausedAfterEnd {
		realtimeSavePausedAfterEnd = false
		realtimeSaveLastAt = time.Time{}
		realtimeSaveLastSkip = ""
		realtimeSaveLastSkipAt = time.Time{}
		realtimeSaveEndBaseline = database.RealtimeFruitSaveInput{}
		realtimeSaveHasEndBaseline = false
	}
	realtimeSaveMu.Unlock()
	setCTCPServerLastMessage("CTCP realtime save resumed: StStatistics total count reset to 0")
}

func resumeRealtimeSaveAfterEnd(reason string) {
	realtimeSaveMu.Lock()
	if realtimeSavePausedAfterEnd {
		realtimeSavePausedAfterEnd = false
		realtimeSaveLastAt = time.Time{}
		realtimeSaveLastSkip = ""
		realtimeSaveLastSkipAt = time.Time{}
		realtimeSaveEndBaseline = database.RealtimeFruitSaveInput{}
		realtimeSaveHasEndBaseline = false
	}
	realtimeSaveMu.Unlock()
	setCTCPServerLastMessage("CTCP realtime save resumed: %s", reason)
}

func logRealtimeSaveSkip(reason string) {
	if strings.TrimSpace(reason) == "" {
		reason = "unknown"
	}
	realtimeSaveMu.Lock()
	shouldLog := reason != realtimeSaveLastSkip || realtimeSaveLastSkipAt.IsZero() || time.Since(realtimeSaveLastSkipAt) >= cTCPRealtimeSaveErrorLogPeriod
	if shouldLog {
		realtimeSaveLastSkip = reason
		realtimeSaveLastSkipAt = time.Now()
	}
	realtimeSaveMu.Unlock()
	if shouldLog {
		setCTCPServerLastMessage("CTCP realtime save skipped: %s, database=%s", reason, database.RealtimeSaveDatabaseForLog())
	}
}
