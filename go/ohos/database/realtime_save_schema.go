package database

import "math"

func realtimeSaveFruitInfoValues(input RealtimeFruitSaveInput, startTime string, programName string) map[string]any {
	return map[string]any{
		"CustomerName":         input.CustomerName,
		"FarmName":             input.FarmName,
		"FruitName":            input.FruitName,
		"StartTime":            startTime,
		"EndTime":              "",
		"StartedState":         "1",
		"CompletedState":       "0",
		"BatchWeight":          input.BatchWeight,
		"BatchNumber":          input.BatchNumber,
		"QualityGradeSum":      input.QualityGradeSum,
		"WeightOrSizeGradeSum": input.WeightOrSizeGradeSum,
		"ExportSum":            input.ExportSum,
		"ColorGradeName":       input.ColorGradeName,
		"ShapeGradeName":       input.ShapeGradeName,
		"FlawGradeName":        input.FlawGradeName,
		"HardGradeName":        input.HardGradeName,
		"DensityGradeName":     input.DensityGradeName,
		"SugarDegreeGradeName": input.SugarDegreeGradeName,
		"ProgramName":          programName,
	}
}

func realtimeSaveFruitInfoColumns() []string {
	return []string{
		"CustomerName",
		"FarmName",
		"FruitName",
		"StartTime",
		"EndTime",
		"StartedState",
		"CompletedState",
		"BatchWeight",
		"BatchNumber",
		"QualityGradeSum",
		"WeightOrSizeGradeSum",
		"ExportSum",
		"ColorGradeName",
		"ShapeGradeName",
		"FlawGradeName",
		"HardGradeName",
		"DensityGradeName",
		"SugarDegreeGradeName",
		"ProgramName",
	}
}

func realtimeSaveGradeInfoColumns() []string {
	return []string{
		"CustomerID",
		"GradeID",
		"BoxNumber",
		"FruitNumber",
		"FruitWeight",
		"QualityName",
		"WeightOrSizeName",
		"WeightOrSizeLimit",
		"SelectWeightOrSize",
		"TraitWeightOrSize",
		"TraitColor",
		"TraitShape",
		"TraitFlaw",
		"TraitHard",
		"TraitDensity",
		"TraitSugarDegree",
		"FPrice",
	}
}

func realtimeSaveExportInfoColumns() []string {
	return []string{
		"CustomerID",
		"ExportID",
		"FruitNumber",
		"FruitWeight",
		"ExitName",
	}
}

func realtimeSaveFruitProcessInfoColumns() []string {
	return []string{
		"RealWeightCount",
		"RealWeightCountPer",
		"SeparationEfficiency",
		"SpeedPercent",
		"AvgWeight",
		"RunningDate",
	}
}

func realtimeFiniteFloat(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return value
}
