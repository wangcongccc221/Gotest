package tcp

import (
	"math"
	"time"

	"gotest/ohos/database"
)

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
