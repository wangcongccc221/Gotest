package tcp

import (
	"encoding/json"
	"math"
	"strconv"
	"sync"
	"time"

	"gotest/ohos/database"
)

const (
	homeStatsDefaultProgramName        = "未选择"
	homeStatsDefaultMaxSpeed           = 680.0
	homeStatsDefaultMaxRealWeightCount = 66.0
	homeStatsWeightScale               = 1000000.0
	homeStatsHistoryInterval           = 15 * time.Second
	homeStatsMaxProcessInfoPoints      = 90
)

type homeStatsProcessInfoPoint struct {
	RunningDate          string  `json:"RunningDate"`
	RealWeightCount      float64 `json:"RealWeightCount"`
	RealWeightCountPer   float64 `json:"RealWeightCountPer"`
	SeparationEfficiency float64 `json:"SeparationEfficiency"`
	SpeedPercent         float64 `json:"SpeedPercent"`
	AvgWeight            float64 `json:"AvgWeight"`
}

type homeStatsPayload struct {
	SubsysId                 int32                       `json:"subsysId"`
	SortSpeedPerMinute       float64                     `json:"sortSpeedPerMinute"`
	SortSpeedPercent         float64                     `json:"sortSpeedPercent"`
	BatchWeightTon           float64                     `json:"batchWeightTon"`
	BatchCount               uint64                      `json:"batchCount"`
	AverageWeightG           float64                     `json:"averageWeightG"`
	CupFillEfficiency        float64                     `json:"cupFillEfficiency"`
	RealtimeOutputTonPerHour float64                     `json:"realtimeOutputTonPerHour"`
	RealtimeOutputPercent    float64                     `json:"realtimeOutputPercent"`
	IntervalSpeedMs          float64                     `json:"intervalSpeedMs"`
	StartTime                int64                       `json:"startTime"`
	RunningTimeSeconds       int64                       `json:"runningTimeSeconds"`
	RunningTimeText          string                      `json:"runningTimeText"`
	ProgramName              string                      `json:"programName"`
	IsProcessing             bool                        `json:"isProcessing"`
	ProcessInfoPoints        []homeStatsProcessInfoPoint `json:"processInfoPoints"`
	TopProductionTonPerHour  float64                     `json:"topProductionTonPerHour"`
	TopSumWeightTon          float64                     `json:"topSumWeightTon"`
	TopSumCounts             uint64                      `json:"topSumCounts"`
	TopAverageWeightG        float64                     `json:"topAverageWeightG"`
}

type homeStatsConfigState struct {
	SubsysNum   int
	ChannelInfo [cTCPServerMaxSubsysNum]int
	ProgramName string
}

type homeStatsHistoryState struct {
	PrevTotalCount             uint64
	PrevTotalCupNum            uint64
	PrevTotalWeight            float64
	PrevAt                     time.Time
	HasPrev                    bool
	CupFillEfficiency          float64
	RealtimeOutputTonPerHour   float64
	RealtimeOutputPercent      float64
	ProcessInfoPoints          []homeStatsProcessInfoPoint
	StartAt                    time.Time
	LastProcessInfoRunningDate string
}

type homeStatsAggregate struct {
	TotalCount         uint64
	TotalCupNum        uint64
	TotalWeight        float64
	SortSpeedPerMinute float64
	IntervalSpeedMs    float64
	HasStats           bool
}

var (
	homeStatsMu      sync.Mutex
	homeStatsConfig  homeStatsConfigState
	homeStatsHistory homeStatsHistoryState
)

func cacheHomeStatsGlobalConfig(stg StGlobal) {
	homeStatsMu.Lock()
	defer homeStatsMu.Unlock()

	homeStatsConfig.SubsysNum = normalizeHomeStatsSubsysNum(int(stg.Sys.NSubsysNum))
	for i := 0; i < cTCPServerMaxSubsysNum; i++ {
		homeStatsConfig.ChannelInfo[i] = normalizeHomeStatsChannelCount(int(stg.Sys.NChannelInfo[i]))
	}

	fruitName := cleanTCPServerCString(stg.Grade.StrFruitName[:])
	if fruitName != "" {
		homeStatsConfig.ProgramName = fruitName
	}
}

func publishLatestHomeStats(now time.Time) {
	payload, ok := buildLatestHomeStatsPayload(now)
	if !ok {
		return
	}
	data, err := json.Marshal(payload)
	if err != nil {
		setCTCPServerLastMessage("homeStats JSON marshal failed: %v", err)
		return
	}
	if err := PublishWebSocketJSON(webSocketTopicHomeStats, string(data)); err != nil {
		setCTCPServerLastMessage("homeStats WebSocket publish failed: %v", err)
	}
}

func buildLatestHomeStatsPayload(now time.Time) (homeStatsPayload, bool) {
	statsList := latestStStatisticsSpeedSnapshots(now)
	if len(statsList) == 0 {
		return homeStatsPayload{}, false
	}

	homeStatsMu.Lock()
	defer homeStatsMu.Unlock()

	subsysNum := resolveHomeStatsSubsysNumLocked(statsList)
	aggregate := aggregateHomeStatsLocked(statsList, subsysNum)
	if aggregate.HasStats && homeStatsHistory.StartAt.IsZero() {
		homeStatsHistory.StartAt = now
	}

	sortSpeedPerMinute := aggregate.SortSpeedPerMinute
	sortSpeedPercent := calculateHomeStatsSortSpeedPercent(sortSpeedPerMinute)
	averageWeightG := 0.0
	if aggregate.TotalCount > 0 {
		averageWeightG = roundHomeStats(aggregate.TotalWeight/float64(aggregate.TotalCount), 2)
	}

	updateHomeStatsHistoryLocked(now, aggregate, sortSpeedPercent, averageWeightG)

	startTime := int64(0)
	runningTimeSeconds := int64(0)
	if !homeStatsHistory.StartAt.IsZero() {
		startTime = homeStatsHistory.StartAt.UnixMilli()
		runningTimeSeconds = int64(now.Sub(homeStatsHistory.StartAt).Seconds())
		if runningTimeSeconds < 0 {
			runningTimeSeconds = 0
		}
	}

	programName := homeStatsConfig.ProgramName
	if programName == "" {
		programName = homeStatsDefaultProgramName
	}

	payload := homeStatsPayload{
		SubsysId:                 0,
		SortSpeedPerMinute:       sortSpeedPerMinute,
		SortSpeedPercent:         sortSpeedPercent,
		BatchWeightTon:           aggregate.TotalWeight / homeStatsWeightScale,
		BatchCount:               aggregate.TotalCount,
		AverageWeightG:           averageWeightG,
		CupFillEfficiency:        homeStatsHistory.CupFillEfficiency,
		RealtimeOutputTonPerHour: homeStatsHistory.RealtimeOutputTonPerHour,
		RealtimeOutputPercent:    homeStatsHistory.RealtimeOutputPercent,
		IntervalSpeedMs:          aggregate.IntervalSpeedMs,
		StartTime:                startTime,
		RunningTimeSeconds:       runningTimeSeconds,
		RunningTimeText:          formatHomeStatsDuration(runningTimeSeconds),
		ProgramName:              programName,
		IsProcessing:             aggregate.TotalCount > 0 || sortSpeedPerMinute > 0,
		ProcessInfoPoints:        append([]homeStatsProcessInfoPoint(nil), homeStatsHistory.ProcessInfoPoints...),
		TopProductionTonPerHour:  homeStatsHistory.RealtimeOutputTonPerHour,
		TopSumWeightTon:          aggregate.TotalWeight / homeStatsWeightScale,
		TopSumCounts:             aggregate.TotalCount,
		TopAverageWeightG:        averageWeightG,
	}
	return payload, true
}

func updateHomeStatsHistoryLocked(now time.Time, aggregate homeStatsAggregate, sortSpeedPercent float64, averageWeightG float64) {
	if !aggregate.HasStats {
		return
	}
	if !homeStatsHistory.HasPrev {
		homeStatsHistory.PrevTotalCount = aggregate.TotalCount
		homeStatsHistory.PrevTotalCupNum = aggregate.TotalCupNum
		homeStatsHistory.PrevTotalWeight = aggregate.TotalWeight
		homeStatsHistory.PrevAt = now
		homeStatsHistory.HasPrev = true
		return
	}
	if now.Sub(homeStatsHistory.PrevAt) < homeStatsHistoryInterval {
		return
	}

	deltaCount := int64(aggregate.TotalCount) - int64(homeStatsHistory.PrevTotalCount)
	deltaCup := int64(aggregate.TotalCupNum) - int64(homeStatsHistory.PrevTotalCupNum)
	deltaWeight := aggregate.TotalWeight - homeStatsHistory.PrevTotalWeight

	efficiency := 0.0
	if deltaCup > 0 && deltaCount >= 0 {
		efficiency = clampHomeStats((float64(deltaCount)*100.0)/float64(deltaCup), 0, 100)
	}

	realtimeOutput := 0.0
	if deltaWeight > 0 {
		realtimeOutput = (deltaWeight / homeStatsWeightScale) * homeStatsRealtimeOutputMultiplier()
	}
	realtimeOutputPercent := calculateHomeStatsRealtimeOutputPercent(realtimeOutput)

	homeStatsHistory.CupFillEfficiency = roundHomeStats(efficiency, 1)
	homeStatsHistory.RealtimeOutputTonPerHour = realtimeOutput
	homeStatsHistory.RealtimeOutputPercent = realtimeOutputPercent
	homeStatsHistory.PrevTotalCount = aggregate.TotalCount
	homeStatsHistory.PrevTotalCupNum = aggregate.TotalCupNum
	homeStatsHistory.PrevTotalWeight = aggregate.TotalWeight
	homeStatsHistory.PrevAt = now

	upsertHomeStatsProcessInfoPointLocked(now, realtimeOutput, realtimeOutputPercent, efficiency, sortSpeedPercent, averageWeightG)
}

func upsertHomeStatsProcessInfoPointLocked(now time.Time, realtimeOutput float64, realtimeOutputPercent float64, efficiency float64, sortSpeedPercent float64, averageWeightG float64) {
	runningDate := now.Format("2006-01-02 15:04")
	if runningDate == homeStatsHistory.LastProcessInfoRunningDate {
		return
	}
	homeStatsHistory.LastProcessInfoRunningDate = runningDate
	homeStatsHistory.ProcessInfoPoints = append(homeStatsHistory.ProcessInfoPoints, homeStatsProcessInfoPoint{
		RunningDate:          runningDate,
		RealWeightCount:      realtimeOutput,
		RealWeightCountPer:   realtimeOutputPercent,
		SeparationEfficiency: roundHomeStats(efficiency, 1),
		SpeedPercent:         roundHomeStats(sortSpeedPercent, 2),
		AvgWeight:            averageWeightG,
	})
	for len(homeStatsHistory.ProcessInfoPoints) > homeStatsMaxProcessInfoPoints {
		homeStatsHistory.ProcessInfoPoints = homeStatsHistory.ProcessInfoPoints[1:]
	}
}

func resolveHomeStatsSubsysNumLocked(statsList []StStatistics) int {
	if homeStatsConfig.SubsysNum > 0 {
		return homeStatsConfig.SubsysNum
	}
	maxSubsys := 0
	for _, stats := range statsList {
		subsysID := int(stats.NSubsysId)
		if subsysID > maxSubsys {
			maxSubsys = subsysID
		}
	}
	return normalizeHomeStatsSubsysNum(maxSubsys)
}

func aggregateHomeStatsLocked(statsList []StStatistics, subsysNum int) homeStatsAggregate {
	var aggregate homeStatsAggregate
	speedSum := 0.0
	speedValidCount := 0
	intervalSum := 0.0
	intervalValidCount := 0

	for _, stats := range statsList {
		subsysID := int(stats.NSubsysId)
		if subsysID <= 0 {
			subsysID = 1
		}
		if subsysNum > 0 && subsysID > subsysNum {
			continue
		}
		aggregate.HasStats = true
		totalCount := homeStatsTotalCount(stats)
		totalWeight := homeStatsTotalWeight(stats)
		aggregate.TotalCount += totalCount
		aggregate.TotalWeight += totalWeight
		if totalCount > 0 {
			aggregate.TotalCupNum += uint64(maxInt64(0, int64(stats.NTotalCupNum))) * uint64(homeStatsChannelCountLocked(subsysID))
		}

		speed := float64(stats.NIntervalSumperminute)
		if speed > 0 {
			speedSum += speed
			speedValidCount++
		}
		if stats.NInterval > 0 && stats.NPulseInterval > 0 {
			intervalSum += float64(stats.NPulseInterval)
			intervalValidCount++
		}
	}

	if speedValidCount > 0 {
		aggregate.SortSpeedPerMinute = math.Floor(speedSum / float64(speedValidCount))
	}
	if intervalValidCount > 0 {
		aggregate.IntervalSpeedMs = math.Floor(intervalSum / float64(intervalValidCount))
	}
	return aggregate
}

func homeStatsTotalCount(stats StStatistics) uint64 {
	if stats.NChannelTotalCount > 0 {
		return stats.NChannelTotalCount
	}
	exitSum := sumHomeStatsUint64(stats.NExitCount[:])
	if exitSum > 0 {
		return exitSum
	}
	return sumHomeStatsUint64(stats.NGradeCount[:])
}

func homeStatsTotalWeight(stats StStatistics) float64 {
	if stats.NChannelWeightCount > 0 {
		return float64(stats.NChannelWeightCount)
	}
	exitSum := sumHomeStatsUint64(stats.NExitWeightCount[:])
	if exitSum > 0 {
		return float64(exitSum)
	}
	return float64(sumHomeStatsUint64(stats.NWeightGradeCount[:]))
}

func sumHomeStatsUint64(values []uint64) uint64 {
	var total uint64
	for _, value := range values {
		total += value
	}
	return total
}

func homeStatsChannelCountLocked(subsysID int) int {
	index := subsysID - 1
	if index < 0 || index >= len(homeStatsConfig.ChannelInfo) {
		return 0
	}
	return normalizeHomeStatsChannelCount(homeStatsConfig.ChannelInfo[index])
}

func normalizeHomeStatsSubsysNum(value int) int {
	if value <= 0 {
		return 0
	}
	if value > cTCPServerMaxSubsysNum {
		return cTCPServerMaxSubsysNum
	}
	return value
}

func normalizeHomeStatsChannelCount(value int) int {
	if value <= 0 {
		return 0
	}
	if value > cTCPServerStSysConfigMaxChan {
		return cTCPServerStSysConfigMaxChan
	}
	return value
}

func calculateHomeStatsSortSpeedPercent(sortSpeedPerMinute float64) float64 {
	maxSpeed := readHomeStatsFloatConfig("MaxSpeed", homeStatsDefaultMaxSpeed)
	if maxSpeed <= 0 {
		return 0
	}
	return roundHomeStats((sortSpeedPerMinute*100.0)/maxSpeed, 2)
}

func calculateHomeStatsRealtimeOutputPercent(realtimeOutputTonPerHour float64) float64 {
	maxOutput := readHomeStatsFloatConfig("MaxRealWeightCount", homeStatsDefaultMaxRealWeightCount)
	if maxOutput <= 0 {
		return 0
	}
	return clampHomeStats(math.Round((realtimeOutputTonPerHour*100.0)/maxOutput), 0, 100)
}

func homeStatsRealtimeOutputMultiplier() float64 {
	if homeStatsHistoryInterval <= 0 {
		return 0
	}
	return float64(time.Hour) / float64(homeStatsHistoryInterval)
}

func readHomeStatsFloatConfig(name string, fallback float64) float64 {
	text, err := database.GetConfigValue(name)
	if err != nil {
		return fallback
	}
	value, err := strconv.ParseFloat(text, 64)
	if err != nil || !isFiniteHomeStats(value) || value <= 0 {
		return fallback
	}
	return value
}

func roundHomeStats(value float64, decimals int) float64 {
	if !isFiniteHomeStats(value) {
		return 0
	}
	if decimals <= 0 {
		return math.Round(value)
	}
	scale := math.Pow10(decimals)
	return math.Round(value*scale) / scale
}

func clampHomeStats(value float64, min float64, max float64) float64 {
	if !isFiniteHomeStats(value) {
		return min
	}
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func formatHomeStatsDuration(seconds int64) string {
	if seconds < 0 {
		seconds = 0
	}
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60
	return twoDigitHomeStats(hours) + ":" + twoDigitHomeStats(minutes) + ":" + twoDigitHomeStats(secs)
}

func twoDigitHomeStats(value int64) string {
	if value < 10 {
		return "0" + strconv.FormatInt(value, 10)
	}
	return strconv.FormatInt(value, 10)
}

func maxInt64(a int64, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func isFiniteHomeStats(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
