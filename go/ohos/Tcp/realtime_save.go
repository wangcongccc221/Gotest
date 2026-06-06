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
)

type realtimeSaveAggregate struct {
	GradeCount       [256]uint64
	WeightGradeCount [256]uint64
	BoxGradeCount    [256]int64
	ExitCount        [MAX_EXIT_NUM]uint64
	ExitWeightCount  [MAX_EXIT_NUM]uint64
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
	TotalCount  uint64
	TotalCupNum uint64
	TotalWeight float64
	At          time.Time
	HasPrev     bool
}

var (
	realtimeSaveMu             sync.Mutex
	realtimeSaveLatestGlobal   StGlobal
	realtimeSaveHasGlobal      bool
	realtimeSaveLastAt         time.Time
	realtimeSaveInFlight       bool
	realtimeSaveLastErr        string
	realtimeSaveLastErrAt      time.Time
	realtimeSaveLastOKAt       time.Time
	realtimeSaveLastSkip       string
	realtimeSaveLastSkipAt     time.Time
	realtimeSaveProcessHistory realtimeSaveProcessSnapshot
	realtimeSavePausedAfterEnd bool
	realtimeSaveEndBaseline    database.RealtimeFruitSaveInput
	realtimeSaveHasEndBaseline bool
)

func cacheRealtimeSaveGlobalConfig(stg StGlobal) {
	realtimeSaveMu.Lock()
	realtimeSaveLatestGlobal = stg
	realtimeSaveHasGlobal = true
	realtimeSaveMu.Unlock()
}

func resetRealtimeSaveState() {
	realtimeSaveMu.Lock()
	realtimeSaveLatestGlobal = StGlobal{}
	realtimeSaveHasGlobal = false
	realtimeSaveLastAt = time.Time{}
	realtimeSaveInFlight = false
	realtimeSaveLastErr = ""
	realtimeSaveLastErrAt = time.Time{}
	realtimeSaveLastOKAt = time.Time{}
	realtimeSaveLastSkip = ""
	realtimeSaveLastSkipAt = time.Time{}
	realtimeSaveProcessHistory = realtimeSaveProcessSnapshot{}
	realtimeSavePausedAfterEnd = false
	realtimeSaveEndBaseline = database.RealtimeFruitSaveInput{}
	realtimeSaveHasEndBaseline = false
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

	go func() {
		customerID, err := database.SaveRealtimeFruitInfo(input)
		finishRealtimeSave(customerID, input, err)
	}()
}

func finishRealtimeSaveSkip(reason string) {
	realtimeSaveMu.Lock()
	realtimeSaveInFlight = false
	realtimeSaveMu.Unlock()
	logRealtimeSaveSkip(reason)
}

func finishRealtimeSave(customerID int, input database.RealtimeFruitSaveInput, err error) {
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
	shouldLog := realtimeSaveLastOKAt.IsZero() || time.Since(realtimeSaveLastOKAt) >= cTCPRealtimeSaveErrorLogPeriod
	if shouldLog {
		realtimeSaveLastOKAt = time.Now()
	}
	realtimeSaveMu.Unlock()

	if shouldLog {
		snapshot, readErr := database.GetFruitCustomerInfoSnapshot(customerID)
		if readErr == nil {
			setCTCPServerLastMessage(
				"CTCP realtime save ok: CustomerID=%d, database=%s, db CustomerName=%s, FarmName=%s, FruitName=%s",
				customerID,
				database.RealtimeSaveDatabaseForLog(),
				snapshot.CustomerName,
				snapshot.FarmName,
				snapshot.FruitName,
			)
		} else {
			setCTCPServerLastMessage("CTCP realtime save ok: CustomerID=%d, database=%s, readback failed: %v", customerID, database.RealtimeSaveDatabaseForLog(), readErr)
		}
		appendCTCPLogChunks("CTCP realtime save rows", formatRealtimeSaveInputForLog(customerID, input))
	}
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
