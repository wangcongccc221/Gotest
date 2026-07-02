package tcp

import (
	"fmt"
	"strings"

	"gotest/ohos/database"
)

func formatRealtimeSaveInputForLog(customerID int, input database.RealtimeFruitSaveInput) string {
	var b strings.Builder
	fmt.Fprintf(
		&b,
		"CustomerID=%d, SavedAt=%s, FruitName=%s, ProgramName=%s, BatchNumber=%d, BatchWeight=%.3f, QualityGradeSum=%d, WeightOrSizeGradeSum=%d, ExportSum=%d\n",
		customerID,
		input.SavedAt.Format("2006-01-02 15:04:05"),
		realtimeSaveLogText(input.FruitName),
		realtimeSaveLogText(input.ProgramName),
		input.BatchNumber,
		input.BatchWeight,
		input.QualityGradeSum,
		input.WeightOrSizeGradeSum,
		input.ExportSum,
	)
	fmt.Fprintf(
		&b,
		"GradeNames: color=%s, shape=%s, flaw=%s, hard=%s, density=%s, sugar=%s\n",
		realtimeSaveLogText(input.ColorGradeName),
		realtimeSaveLogText(input.ShapeGradeName),
		realtimeSaveLogText(input.FlawGradeName),
		realtimeSaveLogText(input.HardGradeName),
		realtimeSaveLogText(input.DensityGradeName),
		realtimeSaveLogText(input.SugarDegreeGradeName),
	)
	if input.Process != nil {
		fmt.Fprintf(
			&b,
			"Process: RunningDate=%s, RealWeightCount=%.3f, RealWeightCountPer=%.3f, SeparationEfficiency=%.3f, SpeedPercent=%.3f, AvgWeight=%.3f\n",
			realtimeSaveLogText(input.Process.RunningDate),
			input.Process.RealWeightCount,
			input.Process.RealWeightCountPer,
			input.Process.SeparationEfficiency,
			input.Process.SpeedPercent,
			input.Process.AvgWeight,
		)
	}

	formatRealtimeSaveGradesForLog(&b, input.Grades)
	formatRealtimeSaveExportsForLog(&b, input.Exports)
	formatRealtimeSaveSystemsForLog(&b, input.Systems)
	return b.String()
}

func formatRealtimeSaveGradesForLog(b *strings.Builder, grades []database.RealtimeGradeSaveInput) {
	nonZero := 0
	for _, grade := range grades {
		if realtimeSaveGradeLogIsZero(grade) {
			continue
		}
		nonZero++
	}
	fmt.Fprintf(b, "Grades: rows=%d, nonZero=%d, zeroRows=%d\n", len(grades), nonZero, len(grades)-nonZero)
	for _, grade := range grades {
		if realtimeSaveGradeLogIsZero(grade) {
			continue
		}
		fmt.Fprintf(
			b,
			"  grade[%d]: quality=%s, weightOrSize=%s, fruit=%d, weight=%.3f, box=%.3f, limit=%.3f, select=%s, traits{color=%s,shape=%s,flaw=%s,hard=%s,density=%s,sugar=%s}\n",
			grade.GradeID,
			realtimeSaveLogText(grade.QualityName),
			realtimeSaveLogText(grade.WeightOrSizeName),
			grade.FruitNumber,
			grade.FruitWeight,
			grade.BoxNumber,
			grade.WeightOrSizeLimit,
			realtimeSaveLogText(grade.SelectWeightOrSize),
			realtimeSaveLogText(grade.TraitColor),
			realtimeSaveLogText(grade.TraitShape),
			realtimeSaveLogText(grade.TraitFlaw),
			realtimeSaveLogText(grade.TraitHard),
			realtimeSaveLogText(grade.TraitDensity),
			realtimeSaveLogText(grade.TraitSugarDegree),
		)
	}
}

func formatRealtimeSaveExportsForLog(b *strings.Builder, exports []database.RealtimeExportSaveInput) {
	nonZero := 0
	for _, export := range exports {
		if export.FruitNumber == 0 && export.FruitWeight == 0 {
			continue
		}
		nonZero++
	}
	fmt.Fprintf(b, "Exports: rows=%d, nonZero=%d, zeroRows=%d\n", len(exports), nonZero, len(exports)-nonZero)
	for _, export := range exports {
		if export.FruitNumber == 0 && export.FruitWeight == 0 {
			continue
		}
		fmt.Fprintf(
			b,
			"  export[%d]: name=%s, fruit=%d, weight=%.3f\n",
			export.ExportID,
			realtimeSaveLogText(export.ExitName),
			export.FruitNumber,
			export.FruitWeight,
		)
	}
}

func formatRealtimeSaveSystemsForLog(b *strings.Builder, systems []database.RealtimeSysFruitSaveInput) {
	fmt.Fprintf(b, "Systems: rows=%d\n", len(systems))
	for _, system := range systems {
		fmt.Fprintf(
			b,
			"  system[%d]: batchNumber=%d, batchWeight=%d\n",
			system.SystemID,
			system.BatchNumber,
			system.BatchWeight,
		)
	}
}

func realtimeSaveGradeLogIsZero(grade database.RealtimeGradeSaveInput) bool {
	return grade.FruitNumber == 0 && grade.FruitWeight == 0 && grade.BoxNumber == 0
}

func realtimeSaveLogText(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return value
}
