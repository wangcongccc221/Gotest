package tcp

import (
	"math"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"gotest/ohos/database"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
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

	go func() {
		customerID, err := database.SaveRealtimeFruitInfo(input)
		finishRealtimeSave(customerID, err)
	}()
}

func finishRealtimeSaveSkip(reason string) {
	realtimeSaveMu.Lock()
	realtimeSaveInFlight = false
	realtimeSaveMu.Unlock()
	logRealtimeSaveSkip(reason)
}

func finishRealtimeSave(customerID int, err error) {
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
		setCTCPServerLastMessage("CTCP realtime save ok: CustomerID=%d, database=%s", customerID, database.RealtimeSaveDatabaseForLog())
	}
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

func buildRealtimeSaveInput(now time.Time) (database.RealtimeFruitSaveInput, bool, string) {
	statsList := latestRealtimeSaveStatisticsSnapshots()
	if len(statsList) == 0 {
		return database.RealtimeFruitSaveInput{}, false, "暂无 StStatistics 快照"
	}

	realtimeSaveMu.Lock()
	stg := realtimeSaveLatestGlobal
	hasGlobal := realtimeSaveHasGlobal
	realtimeSaveMu.Unlock()
	if !hasGlobal {
		return database.RealtimeFruitSaveInput{}, false, "暂无 StGlobal 配置"
	}

	aggregate := aggregateRealtimeSaveStats(statsList, stg.Sys)
	if !aggregate.HasStats || aggregate.TotalCount == 0 {
		return database.RealtimeFruitSaveInput{}, false, "统计总数为 0"
	}

	fruitName := realtimeSaveFixedText(stg.Grade.StrFruitName[:])
	input := database.RealtimeFruitSaveInput{
		SavedAt:              now,
		FruitName:            fruitName,
		ProgramName:          fruitName,
		BatchNumber:          aggregate.TotalCount,
		BatchWeight:          aggregate.TotalWeight,
		QualityGradeSum:      realtimeSaveStoredGradeCount(stg.Grade.NQualityGradeNum),
		WeightOrSizeGradeSum: realtimeSaveStoredGradeCount(stg.Grade.NSizeGradeNum),
		ExportSum:            realtimeSaveExportSum(stg.Sys),
		ColorGradeName:       realtimeSaveJoinedNames(stg.Grade.StrColorGradeName[:], cTCPServerMaxColorGradeNum),
		ShapeGradeName:       realtimeSaveJoinedNames(stg.Grade.StrShapeGradeName[:], cTCPServerMinorGradeNum),
		FlawGradeName:        realtimeSaveJoinedNames(stg.Grade.StFlawareaGradeName[:], cTCPServerMinorGradeNum),
		HardGradeName:        realtimeSaveJoinedNames(stg.Grade.StRigidityGradeName[:], cTCPServerMinorGradeNum),
		DensityGradeName:     realtimeSaveJoinedNames(stg.Grade.StDensityGradeName[:], cTCPServerMinorGradeNum),
		SugarDegreeGradeName: realtimeSaveJoinedNames(stg.Grade.StSugarGradeName[:], cTCPServerMinorGradeNum),
	}
	input.Grades = buildRealtimeSaveGrades(stg.Grade, aggregate)
	input.Exports = buildRealtimeSaveExports(stg.Sys, aggregate)
	input.Systems = buildRealtimeSaveSystems(aggregate)
	input.Process = buildRealtimeSaveProcess(now, aggregate)
	return input, true, ""
}

func latestRealtimeSaveStatisticsSnapshots() []StStatistics {
	cTCPStStatisticsSpeedMu.Lock()
	defer cTCPStStatisticsSpeedMu.Unlock()

	snapshots := make([]StStatistics, 0, len(cTCPStStatisticsSpeedBySys))
	for _, entry := range cTCPStStatisticsSpeedBySys {
		if entry == nil || !entry.HasLatest {
			continue
		}
		snapshots = append(snapshots, entry.Latest)
	}
	return snapshots
}

func aggregateRealtimeSaveStats(statsList []StStatistics, sys StSysConfig) realtimeSaveAggregate {
	var aggregate realtimeSaveAggregate
	subsysNum := normalizeHomeStatsSubsysNum(int(sys.NSubsysNum))
	if subsysNum == 0 {
		subsysNum = realtimeSaveMaxSubsysID(statsList)
	}
	aggregate.SubsysNum = subsysNum

	speedSum := 0.0
	speedValidCount := 0
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
			aggregate.TotalCupNum += uint64(maxInt64(0, int64(stats.NTotalCupNum))) * uint64(realtimeSaveChannelCount(sys, subsysID))
		}

		systemIndex := subsysID - 1
		if systemIndex >= 0 && systemIndex < cTCPServerMaxSubsysNum {
			aggregate.SystemCount[systemIndex] += totalCount
			aggregate.SystemWeight[systemIndex] += realtimeSaveFloatToUint64(totalWeight)
		}

		for i := 0; i < len(aggregate.GradeCount); i++ {
			aggregate.GradeCount[i] += stats.NGradeCount[i]
			aggregate.WeightGradeCount[i] += stats.NWeightGradeCount[i]
			aggregate.BoxGradeCount[i] += int64(stats.NBoxGradeCount[i])
		}
		for i := 0; i < len(aggregate.ExitCount); i++ {
			aggregate.ExitCount[i] += stats.NExitCount[i]
			aggregate.ExitWeightCount[i] += stats.NExitWeightCount[i]
		}

		if stats.NIntervalSumperminute > 0 {
			speedSum += float64(stats.NIntervalSumperminute)
			speedValidCount++
		}
	}

	if speedValidCount > 0 {
		aggregate.SortSpeed = math.Floor(speedSum / float64(speedValidCount))
	}
	return aggregate
}

func realtimeSaveMaxSubsysID(statsList []StStatistics) int {
	maxSubsys := 0
	for _, stats := range statsList {
		if int(stats.NSubsysId) > maxSubsys {
			maxSubsys = int(stats.NSubsysId)
		}
	}
	return normalizeHomeStatsSubsysNum(maxSubsys)
}

func realtimeSaveChannelCount(sys StSysConfig, subsysID int) int {
	index := subsysID - 1
	if index < 0 || index >= len(sys.NChannelInfo) {
		return 0
	}
	return normalizeHomeStatsChannelCount(int(sys.NChannelInfo[index]))
}

func buildRealtimeSaveGrades(grade StGradeInfo, aggregate realtimeSaveAggregate) []database.RealtimeGradeSaveInput {
	qualityCount := realtimeSaveStoredGradeCount(grade.NQualityGradeNum)
	sizeCount := realtimeSaveStoredGradeCount(grade.NSizeGradeNum)
	qualityLoop := realtimeSaveLoopGradeCount(qualityCount)
	sizeLoop := realtimeSaveLoopGradeCount(sizeCount)
	selectWeightOrSize := realtimeSaveSelectWeightOrSize(grade.NClassifyType)

	items := make([]database.RealtimeGradeSaveInput, 0, qualityLoop*sizeLoop)
	for qualityIndex := 0; qualityIndex < qualityLoop; qualityIndex++ {
		for sizeIndex := 0; sizeIndex < sizeLoop; sizeIndex++ {
			gradeID := qualityIndex*cTCPServerMaxSizeGradeNum + sizeIndex
			if gradeID < 0 || gradeID >= len(grade.Grades) {
				continue
			}
			gradeItem := grade.Grades[gradeID]
			qualityName := ""
			if qualityCount > 0 {
				qualityName = realtimeSaveFixedName(grade.StrQualityGradeName[:], qualityIndex)
			}
			item := database.RealtimeGradeSaveInput{
				GradeID:            gradeID,
				BoxNumber:          float64(aggregate.BoxGradeCount[gradeID]),
				FruitNumber:        aggregate.GradeCount[gradeID],
				FruitWeight:        float64(aggregate.WeightGradeCount[gradeID]),
				QualityName:        qualityName,
				WeightOrSizeName:   realtimeSaveFixedName(grade.StrSizeGradeName[:], sizeIndex),
				WeightOrSizeLimit:  float64(gradeItem.NMinSize),
				SelectWeightOrSize: selectWeightOrSize,
				TraitWeightOrSize:  "",
			}
			if qualityCount > 0 {
				item.TraitColor = strconv.Itoa(int(gradeItem.NColorGrade))
				item.TraitShape = strconv.Itoa(int(gradeItem.SbShapeSize))
				item.TraitFlaw = strconv.Itoa(int(gradeItem.SbFlawArea))
				item.TraitHard = strconv.Itoa(int(gradeItem.SbRigidity))
				item.TraitDensity = strconv.Itoa(int(gradeItem.SbDensity))
				item.TraitSugarDegree = strconv.Itoa(int(gradeItem.SbSugar))
			}
			items = append(items, item)
		}
	}
	return items
}

func buildRealtimeSaveExports(sys StSysConfig, aggregate realtimeSaveAggregate) []database.RealtimeExportSaveInput {
	exportSum := realtimeSaveExportSum(sys)
	_, exitInfos, ok := latestStExitInfosSnapshot()
	if !ok {
		exitInfos = defaultStExitInfos()
	}

	items := make([]database.RealtimeExportSaveInput, 0, exportSum)
	for i := 0; i < exportSum; i++ {
		exitName := realtimeSaveFixedName(exitInfos.ExitName[:], i)
		if exitName == "" {
			exitName = strconv.Itoa(i+1) + " "
		}
		items = append(items, database.RealtimeExportSaveInput{
			ExportID:    i,
			FruitNumber: aggregate.ExitCount[i],
			FruitWeight: float64(aggregate.ExitWeightCount[i]),
			ExitName:    exitName,
		})
	}
	return items
}

func buildRealtimeSaveSystems(aggregate realtimeSaveAggregate) []database.RealtimeSysFruitSaveInput {
	subsysNum := aggregate.SubsysNum
	if subsysNum <= 0 {
		subsysNum = cTCPServerMaxSubsysNum
	}
	if subsysNum > cTCPServerMaxSubsysNum {
		subsysNum = cTCPServerMaxSubsysNum
	}

	items := make([]database.RealtimeSysFruitSaveInput, 0, subsysNum)
	for i := 0; i < subsysNum; i++ {
		items = append(items, database.RealtimeSysFruitSaveInput{
			SystemID:    i,
			BatchNumber: realtimeSaveUint64ToInt(aggregate.SystemCount[i]),
			BatchWeight: realtimeSaveUint64ToInt(aggregate.SystemWeight[i]),
		})
	}
	return items
}

func buildRealtimeSaveProcess(now time.Time, aggregate realtimeSaveAggregate) *database.RealtimeFruitProcessSaveInput {
	avgWeight := 0.0
	if aggregate.TotalCount > 0 {
		avgWeight = roundHomeStats(aggregate.TotalWeight/float64(aggregate.TotalCount), 2)
	}

	process := &database.RealtimeFruitProcessSaveInput{
		RunningDate:  now.Format("2006-01-02 15:04"),
		SpeedPercent: roundHomeStats((aggregate.SortSpeed*100.0)/homeStatsDefaultMaxSpeed, 2),
		AvgWeight:    avgWeight,
	}

	realtimeSaveMu.Lock()
	prev := realtimeSaveProcessHistory
	if prev.HasPrev && now.After(prev.At) {
		elapsed := now.Sub(prev.At)
		deltaCount := int64(aggregate.TotalCount) - int64(prev.TotalCount)
		deltaCup := int64(aggregate.TotalCupNum) - int64(prev.TotalCupNum)
		deltaWeight := aggregate.TotalWeight - prev.TotalWeight

		if deltaCup > 0 && deltaCount >= 0 {
			process.SeparationEfficiency = roundHomeStats(clampHomeStats((float64(deltaCount)*100.0)/float64(deltaCup), 0, 100), 1)
		}
		if deltaWeight > 0 && elapsed > 0 {
			process.RealWeightCount = (deltaWeight / homeStatsWeightScale) * (float64(time.Hour) / float64(elapsed))
			process.RealWeightCountPer = clampHomeStats(math.Round((process.RealWeightCount*100.0)/homeStatsDefaultMaxRealWeightCount), 0, 100)
		}
	}
	realtimeSaveProcessHistory = realtimeSaveProcessSnapshot{
		TotalCount:  aggregate.TotalCount,
		TotalCupNum: aggregate.TotalCupNum,
		TotalWeight: aggregate.TotalWeight,
		At:          now,
		HasPrev:     true,
	}
	realtimeSaveMu.Unlock()
	return process
}

func realtimeSaveStoredGradeCount(value uint8) int {
	count := int(value)
	if count < 0 {
		return 0
	}
	if count > cTCPServerMaxSizeGradeNum {
		return cTCPServerMaxSizeGradeNum
	}
	return count
}

func realtimeSaveLoopGradeCount(value int) int {
	if value <= 0 {
		return 1
	}
	if value > cTCPServerMaxSizeGradeNum {
		return cTCPServerMaxSizeGradeNum
	}
	return value
}

func realtimeSaveExportSum(sys StSysConfig) int {
	count := int(sys.NExitNum)
	if count <= 0 {
		return MAX_EXIT_NUM
	}
	if count > MAX_EXIT_NUM {
		return MAX_EXIT_NUM
	}
	return count
}

func realtimeSaveSelectWeightOrSize(classifyType uint8) string {
	if ((classifyType>>2)&1) == 1 || ((classifyType>>3)&1) == 1 || ((classifyType>>4)&1) == 1 {
		return "0"
	}
	return "1"
}

func realtimeSaveJoinedNames(data []uint8, maxItems int) string {
	names := make([]string, 0, maxItems)
	for i := 0; i < maxItems; i++ {
		name := realtimeSaveFixedName(data, i)
		if name != "" {
			names = append(names, name)
		}
	}
	return strings.Join(names, "，")
}

func realtimeSaveFixedName(data []uint8, index int) string {
	if index < 0 {
		return ""
	}
	start := index * cTCPServerMaxTextLength
	end := start + cTCPServerMaxTextLength
	if start >= len(data) {
		return ""
	}
	if end > len(data) {
		end = len(data)
	}
	return realtimeSaveFixedText(data[start:end])
}

func realtimeSaveFixedText(data []uint8) string {
	raw := realtimeSaveCStringBytes(data)
	if len(raw) == 0 {
		return ""
	}
	if utf8.Valid(raw) {
		return strings.TrimSpace(string(raw))
	}
	decoded, _, err := transform.String(simplifiedchinese.GBK.NewDecoder(), string(raw))
	if err == nil {
		return strings.TrimSpace(decoded)
	}
	return cleanTCPServerCString(raw)
}

func realtimeSaveCStringBytes(data []uint8) []byte {
	end := len(data)
	for i, value := range data {
		if value == 0 {
			end = i
			break
		}
	}
	raw := make([]byte, end)
	copy(raw, data[:end])
	return raw
}

func realtimeSaveUint64ToInt(value uint64) int {
	maxInt := uint64(^uint(0) >> 1)
	if value > maxInt {
		return int(maxInt)
	}
	return int(value)
}

func realtimeSaveFloatToUint64(value float64) uint64 {
	if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 {
		return 0
	}
	maxUint := ^uint64(0)
	if value >= float64(maxUint) {
		return maxUint
	}
	return uint64(value)
}
