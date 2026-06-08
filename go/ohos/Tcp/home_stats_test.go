package tcp

import (
	"path/filepath"
	"testing"
	"time"

	"gotest/ohos/database"
)

func resetHomeStatsTestState() {
	resetRealtimeSaveState()
	resetStStatisticsCacheAfterEndProcess()

	homeStatsMu.Lock()
	homeStatsConfig = homeStatsConfigState{}
	homeStatsHistory = homeStatsHistoryState{}
	homeStatsMu.Unlock()
}

func TestMarkRealtimeSaveProcessEndedResetsHomeStatsHistory(t *testing.T) {
	resetHomeStatsTestState()
	t.Cleanup(resetHomeStatsTestState)

	startAt := time.Date(2026, 6, 6, 9, 0, 0, 0, cTCPChinaLocation)

	homeStatsMu.Lock()
	homeStatsHistory = homeStatsHistoryState{
		PrevTotalCount:           50000,
		PrevTotalCupNum:          52000,
		PrevTotalWeight:          987654,
		PrevAt:                   startAt.Add(homeStatsHistoryInterval),
		HasPrev:                  true,
		CupFillEfficiency:        96.4,
		RealtimeOutputTonPerHour: 12.8,
		RealtimeOutputPercent:    19,
		ProcessInfoPoints: []homeStatsProcessInfoPoint{
			{
				RunningDate:          "2026-06-06 09:00",
				RealWeightCount:      12.8,
				RealWeightCountPer:   19,
				SeparationEfficiency: 96.4,
				SpeedPercent:         80,
				AvgWeight:            210.5,
			},
		},
		StartAt:                    startAt,
		LastProcessInfoRunningDate: "2026-06-06 09:00",
	}
	homeStatsMu.Unlock()

	markRealtimeSaveProcessEnded()

	homeStatsMu.Lock()
	history := homeStatsHistory
	homeStatsMu.Unlock()

	if history.HasPrev {
		t.Fatal("HasPrev = true, want false after end process")
	}
	if !history.PrevAt.IsZero() {
		t.Fatalf("PrevAt = %v, want zero after end process", history.PrevAt)
	}
	if history.PrevTotalCount != 0 || history.PrevTotalCupNum != 0 || history.PrevTotalWeight != 0 {
		t.Fatalf(
			"previous totals = (%d, %d, %.0f), want all zero after end process",
			history.PrevTotalCount,
			history.PrevTotalCupNum,
			history.PrevTotalWeight,
		)
	}
	if history.CupFillEfficiency != 0 || history.RealtimeOutputTonPerHour != 0 || history.RealtimeOutputPercent != 0 {
		t.Fatalf(
			"efficiency/output = (%.1f, %.1f, %.1f), want all zero after end process",
			history.CupFillEfficiency,
			history.RealtimeOutputTonPerHour,
			history.RealtimeOutputPercent,
		)
	}
	if !history.StartAt.IsZero() {
		t.Fatalf("StartAt = %v, want zero after end process", history.StartAt)
	}
	if history.LastProcessInfoRunningDate != "" {
		t.Fatalf("LastProcessInfoRunningDate = %q, want empty after end process", history.LastProcessInfoRunningDate)
	}
	if len(history.ProcessInfoPoints) != 0 {
		t.Fatalf("ProcessInfoPoints length = %d, want 0 after end process", len(history.ProcessInfoPoints))
	}
}

func TestHomeStatsTotalCountDoesNotFallbackToExitOrGradeCounts(t *testing.T) {
	stats := StStatistics{}
	stats.NExitCount[0] = 123
	stats.NGradeCount[0] = 456

	if got := homeStatsTotalCount(stats); got != 0 {
		t.Fatalf("homeStatsTotalCount() = %d, want 0 when nTotalCount is 0", got)
	}

	stats.NChannelTotalCount = 789
	if got := homeStatsTotalCount(stats); got != 789 {
		t.Fatalf("homeStatsTotalCount() = %d, want nTotalCount 789", got)
	}
}

func TestUpdateHomeStatsHistorySkipsBaselineWhenCupDenominatorMissing(t *testing.T) {
	resetHomeStatsTestState()
	t.Cleanup(resetHomeStatsTestState)

	now := time.Date(2026, 6, 6, 9, 0, 0, 0, cTCPChinaLocation)

	homeStatsMu.Lock()
	updateHomeStatsHistoryLocked(now, homeStatsAggregate{
		TotalCount:  100,
		TotalCupNum: 0,
		TotalWeight: 5000,
		HasStats:    true,
	}, 0, 0)
	history := homeStatsHistory
	homeStatsMu.Unlock()

	if history.HasPrev {
		t.Fatal("HasPrev = true, want false when cup denominator is missing")
	}
	if history.PrevTotalCount != 0 || history.PrevTotalCupNum != 0 || history.PrevTotalWeight != 0 {
		t.Fatalf(
			"previous totals = (%d, %d, %.0f), want all zero when cup denominator is missing",
			history.PrevTotalCount,
			history.PrevTotalCupNum,
			history.PrevTotalWeight,
		)
	}
	if history.CupFillEfficiency != 0 || history.RealtimeOutputTonPerHour != 0 || history.RealtimeOutputPercent != 0 {
		t.Fatalf(
			"efficiency/output = (%.1f, %.1f, %.1f), want all zero when cup denominator is missing",
			history.CupFillEfficiency,
			history.RealtimeOutputTonPerHour,
			history.RealtimeOutputPercent,
		)
	}
}

func TestUpdateHomeStatsHistoryUsesExitCountForEfficiency(t *testing.T) {
	resetHomeStatsTestState()
	t.Cleanup(resetHomeStatsTestState)

	start := time.Date(2026, 6, 6, 9, 0, 0, 0, cTCPChinaLocation)
	now := start.Add(homeStatsHistoryInterval)

	homeStatsMu.Lock()
	homeStatsHistory = homeStatsHistoryState{
		PrevTotalCount:  1000,
		PrevTotalCupNum: 2000,
		PrevExitCount:   800,
		PrevTotalWeight: 1000000,
		PrevAt:          start,
		HasPrev:         true,
	}
	updateHomeStatsHistoryLocked(now, homeStatsAggregate{
		TotalCount:     2500,
		TotalCupNum:    3386,
		TotalExitCount: 2000,
		TotalWeight:    2000000,
		HasStats:       true,
	}, 0, 0)
	history := homeStatsHistory
	homeStatsMu.Unlock()

	if history.CupFillEfficiency != 86.6 {
		t.Fatalf("CupFillEfficiency = %.1f, want 86.6 from deltaExit/deltaCup", history.CupFillEfficiency)
	}
}

func TestBuildLatestHomeStatsPayloadDoesNotStartOnIdleStatsAfterEndProcess(t *testing.T) {
	resetHomeStatsTestState()
	t.Cleanup(resetHomeStatsTestState)

	endAt := time.Date(2026, 6, 6, 9, 0, 0, 0, cTCPChinaLocation)
	nextAt := endAt.Add(2 * time.Minute)

	homeStatsMu.Lock()
	homeStatsConfig.ChannelInfo[0] = 1
	homeStatsMu.Unlock()

	cTCPStStatisticsSpeedMu.Lock()
	cTCPStStatisticsSpeedBySys = map[int32]*cTCPStStatisticsSpeedState{
		1: {
			Latest:    StStatistics{NSubsysId: 1},
			LatestAt:  endAt,
			HasLatest: true,
		},
	}
	cTCPStStatisticsSpeedMu.Unlock()

	idlePayload, ok := buildLatestHomeStatsPayload(endAt)
	if !ok {
		t.Fatal("idle payload ok = false, want true with a statistics snapshot")
	}
	if idlePayload.StartTime != 0 {
		t.Fatalf("idle StartTime = %d, want 0 before the next batch starts", idlePayload.StartTime)
	}

	cTCPStStatisticsSpeedMu.Lock()
	cTCPStStatisticsSpeedBySys[1] = &cTCPStStatisticsSpeedState{
		Latest: StStatistics{
			NSubsysId:             1,
			NChannelTotalCount:    1,
			NTotalCupNum:          1,
			NIntervalSumperminute: 300,
		},
		LatestAt:  nextAt,
		HasLatest: true,
	}
	cTCPStStatisticsSpeedMu.Unlock()

	nextPayload, ok := buildLatestHomeStatsPayload(nextAt)
	if !ok {
		t.Fatal("next payload ok = false, want true when the next batch starts")
	}
	if nextPayload.StartTime != nextAt.UnixMilli() {
		t.Fatalf("next StartTime = %d, want %d", nextPayload.StartTime, nextAt.UnixMilli())
	}
}

func TestResetHomeStatsEfficiencyWindowClearsBaselineButKeepsBatchContext(t *testing.T) {
	resetHomeStatsTestState()
	t.Cleanup(resetHomeStatsTestState)

	startAt := time.Date(2026, 6, 6, 9, 0, 0, 0, cTCPChinaLocation)

	homeStatsMu.Lock()
	homeStatsHistory = homeStatsHistoryState{
		PrevTotalCount:           315234,
		PrevTotalCupNum:          13467305,
		PrevExitCount:            300000,
		PrevGradeCount:           315234,
		PrevTotalWeight:          123456789,
		PrevAt:                   startAt,
		HasPrev:                  true,
		CupFillEfficiency:        100,
		RealtimeOutputTonPerHour: 12.3,
		RealtimeOutputPercent:    19,
		ProcessInfoPoints: []homeStatsProcessInfoPoint{
			{RunningDate: "2026-06-06 09:00", SeparationEfficiency: 100},
		},
		StartAt:                    startAt,
		LastProcessInfoRunningDate: "2026-06-06 09:00",
	}
	homeStatsMu.Unlock()

	resetHomeStatsEfficiencyWindow("")

	homeStatsMu.Lock()
	history := homeStatsHistory
	homeStatsMu.Unlock()

	if history.HasPrev {
		t.Fatal("HasPrev = true, want false after opening a fresh home stats window")
	}
	if !history.PrevAt.IsZero() || history.PrevTotalCount != 0 || history.PrevTotalCupNum != 0 || history.PrevTotalWeight != 0 {
		t.Fatalf(
			"baseline = (%v, %d, %d, %.0f), want cleared",
			history.PrevAt,
			history.PrevTotalCount,
			history.PrevTotalCupNum,
			history.PrevTotalWeight,
		)
	}
	if history.PrevExitCount != 0 || history.PrevGradeCount != 0 {
		t.Fatalf("diagnostic baseline = (%d, %d), want cleared", history.PrevExitCount, history.PrevGradeCount)
	}
	if history.CupFillEfficiency != 0 || history.RealtimeOutputTonPerHour != 0 || history.RealtimeOutputPercent != 0 {
		t.Fatalf(
			"efficiency/output = (%.1f, %.1f, %.1f), want cleared",
			history.CupFillEfficiency,
			history.RealtimeOutputTonPerHour,
			history.RealtimeOutputPercent,
		)
	}
	if history.StartAt.IsZero() || len(history.ProcessInfoPoints) != 1 || history.LastProcessInfoRunningDate == "" {
		t.Fatal("batch context was cleared; want only the efficiency baseline reset")
	}
}

func TestHomeStatsHistoryIntervalMatches48StaticTimer(t *testing.T) {
	if homeStatsHistoryInterval != 20*time.Second {
		t.Fatalf("homeStatsHistoryInterval = %s, want 20s to match 48 CalculateStatics timer", homeStatsHistoryInterval)
	}
}

func TestRealtimeSaveIntervalMatches48SaveThread(t *testing.T) {
	if cTCPRealtimeSaveInterval != 10*time.Second {
		t.Fatalf("cTCPRealtimeSaveInterval = %s, want 10s to match 48 SaveDataThread delay", cTCPRealtimeSaveInterval)
	}
}

func TestBuildRealtimeSaveProcessUses48TwentySecondWindow(t *testing.T) {
	resetHomeStatsTestState()
	t.Cleanup(resetHomeStatsTestState)

	start := time.Date(2026, 6, 6, 9, 0, 0, 0, cTCPChinaLocation)
	first := realtimeSaveAggregate{
		TotalCount:     100,
		TotalCupNum:    200,
		TotalExitCount: 100,
		TotalWeight:    1000000,
		SortSpeed:      340,
		HasStats:       true,
	}
	if process := buildRealtimeSaveProcess(start, first); process != nil {
		t.Fatalf("first process = %#v, want nil baseline sample", process)
	}

	after10s := realtimeSaveAggregate{
		TotalCount:     110,
		TotalCupNum:    220,
		TotalExitCount: 110,
		TotalWeight:    1100000,
		SortSpeed:      340,
		HasStats:       true,
	}
	if process := buildRealtimeSaveProcess(start.Add(10*time.Second), after10s); process != nil {
		t.Fatalf("10s process = %#v, want nil before 20s window", process)
	}

	after20s := realtimeSaveAggregate{
		TotalCount:     120,
		TotalCupNum:    240,
		TotalExitCount: 120,
		TotalWeight:    1200000,
		SortSpeed:      340,
		HasStats:       true,
	}
	process := buildRealtimeSaveProcess(start.Add(20*time.Second), after20s)
	if process == nil {
		t.Fatal("20s process = nil, want process point")
	}
	if process.SeparationEfficiency != 50 {
		t.Fatalf("SeparationEfficiency = %.1f, want 50.0", process.SeparationEfficiency)
	}
	if process.RealWeightCount != 36 {
		t.Fatalf("RealWeightCount = %.1f, want 36.0", process.RealWeightCount)
	}
}

func TestBuildRealtimeSaveProcessUsesExitCountForSeparationEfficiency(t *testing.T) {
	resetHomeStatsTestState()
	t.Cleanup(resetHomeStatsTestState)

	start := time.Date(2026, 6, 6, 9, 0, 0, 0, cTCPChinaLocation)
	first := realtimeSaveAggregate{
		TotalCount:     1000,
		TotalCupNum:    2000,
		TotalExitCount: 800,
		TotalWeight:    1000000,
		SortSpeed:      340,
		HasStats:       true,
	}
	if process := buildRealtimeSaveProcess(start, first); process != nil {
		t.Fatalf("first process = %#v, want nil baseline sample", process)
	}

	second := realtimeSaveAggregate{
		TotalCount:     2500,
		TotalCupNum:    3386,
		TotalExitCount: 2000,
		TotalWeight:    2000000,
		SortSpeed:      340,
		HasStats:       true,
	}
	process := buildRealtimeSaveProcess(start.Add(20*time.Second), second)
	if process == nil {
		t.Fatal("20s process = nil, want process point")
	}
	if process.SeparationEfficiency != 86.6 {
		t.Fatalf("SeparationEfficiency = %.1f, want 86.6 from deltaExit/deltaCup", process.SeparationEfficiency)
	}
}

func TestBuildRealtimeSaveProcessUsesConfiguredOutputAndSpeedPercent(t *testing.T) {
	resetHomeStatsTestState()
	t.Cleanup(resetHomeStatsTestState)
	t.Cleanup(func() {
		_ = database.InitORMWithPath("")
	})

	dbPath := filepath.Join(t.TempDir(), "config.db")
	if err := database.InitORMWithPath(dbPath); err != nil {
		t.Fatalf("InitORMWithPath() error = %v", err)
	}
	if err := database.SaveConfigValue("MaxRealWeightCount", "120"); err != nil {
		t.Fatalf("SaveConfigValue(MaxRealWeightCount) error = %v", err)
	}
	if err := database.SaveConfigValue("MaxSpeed", "1000"); err != nil {
		t.Fatalf("SaveConfigValue(MaxSpeed) error = %v", err)
	}

	start := time.Date(2026, 6, 6, 9, 0, 0, 0, cTCPChinaLocation)
	first := realtimeSaveAggregate{
		TotalCount:     100,
		TotalCupNum:    200,
		TotalExitCount: 100,
		TotalWeight:    1000000,
		SortSpeed:      500,
		HasStats:       true,
	}
	if process := buildRealtimeSaveProcess(start, first); process != nil {
		t.Fatalf("first process = %#v, want nil baseline sample", process)
	}

	second := realtimeSaveAggregate{
		TotalCount:     120,
		TotalCupNum:    240,
		TotalExitCount: 120,
		TotalWeight:    1200000,
		SortSpeed:      500,
		HasStats:       true,
	}
	process := buildRealtimeSaveProcess(start.Add(20*time.Second), second)
	if process == nil {
		t.Fatal("20s process = nil, want process point")
	}
	if process.RealWeightCount != 36 {
		t.Fatalf("RealWeightCount = %.1f, want 36.0", process.RealWeightCount)
	}
	if process.RealWeightCountPer != 30 {
		t.Fatalf("RealWeightCountPer = %.1f, want 30.0 from configured MaxRealWeightCount", process.RealWeightCountPer)
	}
	if process.SpeedPercent != 50 {
		t.Fatalf("SpeedPercent = %.1f, want 50.0 from configured MaxSpeed", process.SpeedPercent)
	}
}
