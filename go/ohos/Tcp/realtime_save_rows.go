package tcp

import (
	"math"
	"strconv"
	"time"

	"gotest/ohos/database"
)

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
	if aggregate.TotalCount > 0 && aggregate.TotalCupNum == 0 {
		return nil
	}

	realtimeSaveMu.Lock()
	prev := realtimeSaveProcessHistory
	if !prev.HasPrev {
		realtimeSaveProcessHistory = realtimeSaveProcessSnapshot{
			TotalExitCount: aggregate.TotalExitCount,
			TotalCount:     aggregate.TotalCount,
			TotalCupNum:    aggregate.TotalCupNum,
			TotalWeight:    aggregate.TotalWeight,
			At:             now,
			HasPrev:        true,
		}
		realtimeSaveMu.Unlock()
		return nil
	}
	if !now.After(prev.At) || now.Sub(prev.At) < homeStatsHistoryInterval {
		realtimeSaveMu.Unlock()
		return nil
	}

	elapsed := now.Sub(prev.At)
	deltaExit := int64(aggregate.TotalExitCount) - int64(prev.TotalExitCount)
	deltaCount := int64(aggregate.TotalCount) - int64(prev.TotalCount)
	deltaCup := int64(aggregate.TotalCupNum) - int64(prev.TotalCupNum)
	deltaWeight := aggregate.TotalWeight - prev.TotalWeight
	realtimeSaveProcessHistory = realtimeSaveProcessSnapshot{
		TotalExitCount: aggregate.TotalExitCount,
		TotalCount:     aggregate.TotalCount,
		TotalCupNum:    aggregate.TotalCupNum,
		TotalWeight:    aggregate.TotalWeight,
		At:             now,
		HasPrev:        true,
	}
	realtimeSaveMu.Unlock()

	avgWeight := 0.0
	if aggregate.TotalCount > 0 {
		avgWeight = roundHomeStats(aggregate.TotalWeight/float64(aggregate.TotalCount), 2)
	}

	process := &database.RealtimeFruitProcessSaveInput{
		RunningDate:  now.Format("2006-01-02 15:04"),
		SpeedPercent: roundHomeStats((aggregate.SortSpeed*100.0)/homeStatsDefaultMaxSpeed, 2),
		AvgWeight:    avgWeight,
	}
	if deltaCount >= 0 {
		process.SeparationEfficiency = roundHomeStats(calculateHomeStatsEfficiencyPercent(deltaExit, deltaCup), 1)
	}
	if deltaWeight > 0 && elapsed > 0 {
		process.RealWeightCount = (deltaWeight / homeStatsWeightScale) * (float64(time.Hour) / float64(elapsed))
		process.RealWeightCountPer = clampHomeStats(math.Round((process.RealWeightCount*100.0)/homeStatsDefaultMaxRealWeightCount), 0, 100)
	}
	return process
}
