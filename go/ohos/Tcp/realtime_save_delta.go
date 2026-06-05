package tcp

import "gotest/ohos/database"

func realtimeSaveDeltaAfterEnd(current database.RealtimeFruitSaveInput) (database.RealtimeFruitSaveInput, bool) {
	realtimeSaveMu.Lock()
	baseline := realtimeSaveEndBaseline
	hasBaseline := realtimeSaveHasEndBaseline
	realtimeSaveMu.Unlock()

	if !hasBaseline {
		return current, current.BatchNumber > 0 || current.BatchWeight > 0
	}

	if current.BatchNumber < baseline.BatchNumber || current.BatchWeight < baseline.BatchWeight {
		return current, current.BatchNumber > 0 || current.BatchWeight > 0
	}
	if current.BatchNumber <= baseline.BatchNumber && current.BatchWeight <= baseline.BatchWeight {
		return database.RealtimeFruitSaveInput{}, false
	}

	delta := current
	delta.BatchNumber = subtractUint64Floor(current.BatchNumber, baseline.BatchNumber)
	delta.BatchWeight = subtractFloat64Floor(current.BatchWeight, baseline.BatchWeight)
	delta.Grades = realtimeSaveDeltaGrades(current.Grades, baseline.Grades)
	delta.Exports = realtimeSaveDeltaExports(current.Exports, baseline.Exports)
	delta.Systems = realtimeSaveDeltaSystems(current.Systems, baseline.Systems)
	if delta.BatchNumber <= 0 && delta.BatchWeight <= 0 {
		return database.RealtimeFruitSaveInput{}, false
	}
	return delta, true
}

func realtimeSaveDeltaGrades(current []database.RealtimeGradeSaveInput, baseline []database.RealtimeGradeSaveInput) []database.RealtimeGradeSaveInput {
	baselineByID := make(map[int]database.RealtimeGradeSaveInput, len(baseline))
	for _, item := range baseline {
		baselineByID[item.GradeID] = item
	}

	result := make([]database.RealtimeGradeSaveInput, 0, len(current))
	for _, item := range current {
		base := baselineByID[item.GradeID]
		item.BoxNumber = subtractFloat64Floor(item.BoxNumber, base.BoxNumber)
		item.FruitNumber = subtractUint64Floor(item.FruitNumber, base.FruitNumber)
		item.FruitWeight = subtractFloat64Floor(item.FruitWeight, base.FruitWeight)
		result = append(result, item)
	}
	return result
}

func realtimeSaveDeltaExports(current []database.RealtimeExportSaveInput, baseline []database.RealtimeExportSaveInput) []database.RealtimeExportSaveInput {
	baselineByID := make(map[int]database.RealtimeExportSaveInput, len(baseline))
	for _, item := range baseline {
		baselineByID[item.ExportID] = item
	}

	result := make([]database.RealtimeExportSaveInput, 0, len(current))
	for _, item := range current {
		base := baselineByID[item.ExportID]
		item.FruitNumber = subtractUint64Floor(item.FruitNumber, base.FruitNumber)
		item.FruitWeight = subtractFloat64Floor(item.FruitWeight, base.FruitWeight)
		result = append(result, item)
	}
	return result
}

func realtimeSaveDeltaSystems(current []database.RealtimeSysFruitSaveInput, baseline []database.RealtimeSysFruitSaveInput) []database.RealtimeSysFruitSaveInput {
	baselineByID := make(map[int]database.RealtimeSysFruitSaveInput, len(baseline))
	for _, item := range baseline {
		baselineByID[item.SystemID] = item
	}

	result := make([]database.RealtimeSysFruitSaveInput, 0, len(current))
	for _, item := range current {
		base := baselineByID[item.SystemID]
		item.BatchNumber = subtractIntFloor(item.BatchNumber, base.BatchNumber)
		item.BatchWeight = subtractIntFloor(item.BatchWeight, base.BatchWeight)
		result = append(result, item)
	}
	return result
}

func subtractUint64Floor(current uint64, baseline uint64) uint64 {
	if current <= baseline {
		return 0
	}
	return current - baseline
}

func subtractFloat64Floor(current float64, baseline float64) float64 {
	if current <= baseline {
		return 0
	}
	return current - baseline
}

func subtractIntFloor(current int, baseline int) int {
	if current <= baseline {
		return 0
	}
	return current - baseline
}
