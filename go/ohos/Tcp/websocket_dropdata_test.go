package tcp

import "testing"

func intPtr(v int) *int {
	return &v
}

func TestApplyGradeDropActionAddUsesMatrixIndexAndOrsExitBits(t *testing.T) {
	var grade StGradeInfo
	grade.Grades[2*cTCPServerMaxSizeGradeNum+3].ExitLow = 1

	err := applyGradeDropAction(&grade, "add", 33, []webSocketDropGrade{{
		Row: intPtr(2),
		Col: intPtr(3),
	}})
	if err != nil {
		t.Fatalf("applyGradeDropAction returned error: %v", err)
	}

	item := grade.Grades[2*cTCPServerMaxSizeGradeNum+3]
	if item.ExitLow != 1 || item.ExitHigh != 1 {
		t.Fatalf("exit bits = low:%08x high:%08x, want low:00000001 high:00000001", item.ExitLow, item.ExitHigh)
	}
	if grade.ExitEnabled[1]&1 == 0 {
		t.Fatalf("ExitEnabled[1] did not enable exit 33: %08x", uint32(grade.ExitEnabled[1]))
	}
}

func TestApplyGradeDropActionRemoveClearsOnlyRequestedExitBit(t *testing.T) {
	var grade StGradeInfo
	grade.Grades[1*cTCPServerMaxSizeGradeNum+2].ExitLow = 0b101

	err := applyGradeDropAction(&grade, "remove", 3, []webSocketDropGrade{{
		Row: intPtr(1),
		Col: intPtr(2),
	}})
	if err != nil {
		t.Fatalf("applyGradeDropAction returned error: %v", err)
	}

	item := grade.Grades[1*cTCPServerMaxSizeGradeNum+2]
	if item.ExitLow != 0b001 || item.ExitHigh != 0 {
		t.Fatalf("exit bits = low:%08x high:%08x, want low:00000001 high:00000000", item.ExitLow, item.ExitHigh)
	}
}

func TestMergeGradeExitStatePreservesCachedAndIncomingExitBits(t *testing.T) {
	var incoming StGradeInfo
	var cached StGradeInfo

	incoming.Grades[0].ExitLow = 0b001
	cached.Grades[0].ExitLow = 0b010
	incoming.Grades[1].ExitLow = 0b100
	cached.Grades[1].ExitHigh = 0b001
	incoming.ExitEnabled[0] = 0b001
	cached.ExitEnabled[0] = 0b010
	cached.ExitEnabled[1] = 0b001

	mergeGradeExitState(&incoming, cached)

	if incoming.Grades[0].ExitLow != 0b011 || incoming.Grades[0].ExitHigh != 0 {
		t.Fatalf("grade0 exit bits = low:%08x high:%08x, want low:00000003 high:00000000", incoming.Grades[0].ExitLow, incoming.Grades[0].ExitHigh)
	}
	if incoming.Grades[1].ExitLow != 0b100 || incoming.Grades[1].ExitHigh != 0b001 {
		t.Fatalf("grade1 exit bits = low:%08x high:%08x, want low:00000004 high:00000001", incoming.Grades[1].ExitLow, incoming.Grades[1].ExitHigh)
	}
	if incoming.ExitEnabled[0] != 0b011 || incoming.ExitEnabled[1] != 0b001 {
		t.Fatalf("ExitEnabled = [%08x,%08x], want [00000003,00000001]", uint32(incoming.ExitEnabled[0]), uint32(incoming.ExitEnabled[1]))
	}
}

func TestShouldPreserveCachedGradeExitsRequiresSameGradeShape(t *testing.T) {
	cached := StGradeInfo{
		NClassifyType:    1,
		NSizeGradeNum:    10,
		NQualityGradeNum: 2,
	}
	incoming := cached

	if !shouldPreserveCachedGradeExits(incoming, cached, nil) {
		t.Fatalf("same grade shape should preserve cached exit bits")
	}

	incoming.NSizeGradeNum = 2
	if shouldPreserveCachedGradeExits(incoming, cached, nil) {
		t.Fatalf("changed size grade count should not preserve cached exit bits")
	}

	incoming = cached
	incoming.NQualityGradeNum = 1
	if shouldPreserveCachedGradeExits(incoming, cached, nil) {
		t.Fatalf("changed quality grade count should not preserve cached exit bits")
	}

	incoming = cached
	incoming.NClassifyType = 0
	if shouldPreserveCachedGradeExits(incoming, cached, nil) {
		t.Fatalf("changed classify type should not preserve cached exit bits")
	}
}

func TestShouldPreserveCachedGradeExitsHonorsExplicitFalse(t *testing.T) {
	cached := StGradeInfo{
		NClassifyType:    1,
		NSizeGradeNum:    10,
		NQualityGradeNum: 2,
	}
	incoming := cached
	preserve := false

	if shouldPreserveCachedGradeExits(incoming, cached, &preserve) {
		t.Fatalf("explicit preserveCachedGradeExits=false should not preserve cached exit bits")
	}
}

func TestShouldPreserveCachedGradeExitsHonorsExplicitTrue(t *testing.T) {
	cached := StGradeInfo{
		NClassifyType:    1,
		NSizeGradeNum:    10,
		NQualityGradeNum: 2,
	}
	incoming := cached
	incoming.NSizeGradeNum = 3
	preserve := true

	if !shouldPreserveCachedGradeExits(incoming, cached, &preserve) {
		t.Fatalf("explicit preserveCachedGradeExits=true should preserve cached exit bits")
	}
}

func TestClearGradeExitMappingsClearsAllGradeExitValues(t *testing.T) {
	var grade StGradeInfo
	grade.NClassifyType = 1
	grade.NQualityGradeNum = 2
	grade.NSizeGradeNum = 3
	grade.ExitEnabled[0] = 0b011
	grade.ExitEnabled[1] = 0b001

	grade.Grades[0].ExitLow = 0b001
	grade.Grades[2].ExitHigh = 0b001
	grade.Grades[cTCPServerMaxSizeGradeNum+2].ExitLow = 0b100
	grade.Grades[5].ExitLow = 0b100

	clearGradeExitMappings(&grade)

	if grade.Grades[0].ExitLow != 0 || grade.Grades[0].ExitHigh != 0 {
		t.Fatalf("grade0 exit bits = low:%08x high:%08x, want zero", grade.Grades[0].ExitLow, grade.Grades[0].ExitHigh)
	}
	if grade.Grades[2].ExitLow != 0 || grade.Grades[2].ExitHigh != 0 {
		t.Fatalf("grade2 exit bits = low:%08x high:%08x, want zero", grade.Grades[2].ExitLow, grade.Grades[2].ExitHigh)
	}
	if grade.Grades[cTCPServerMaxSizeGradeNum+2].ExitLow != 0 || grade.Grades[cTCPServerMaxSizeGradeNum+2].ExitHigh != 0 {
		t.Fatalf("quality1 size2 exit bits = low:%08x high:%08x, want zero", grade.Grades[cTCPServerMaxSizeGradeNum+2].ExitLow, grade.Grades[cTCPServerMaxSizeGradeNum+2].ExitHigh)
	}
	if grade.Grades[5].ExitLow != 0 || grade.Grades[5].ExitHigh != 0 {
		t.Fatalf("inactive grade5 exit bits = low:%08x high:%08x, want zero", grade.Grades[5].ExitLow, grade.Grades[5].ExitHigh)
	}
	if grade.ExitEnabled[0] != 0b011 || grade.ExitEnabled[1] != 0b001 {
		t.Fatalf("ExitEnabled = [%08x,%08x], want unchanged [00000003,00000001]", uint32(grade.ExitEnabled[0]), uint32(grade.ExitEnabled[1]))
	}
}

func TestClearGradeExitMappingsWithSizeOnlyClassifyIgnoresQualityRows(t *testing.T) {
	var grade StGradeInfo
	grade.NClassifyType = 0
	grade.NQualityGradeNum = 4
	grade.NSizeGradeNum = 2

	grade.Grades[0].ExitLow = 0b001
	grade.Grades[1].ExitLow = 0b010
	grade.Grades[cTCPServerMaxSizeGradeNum].ExitLow = 0b100

	clearGradeExitMappings(&grade)

	if grade.Grades[0].ExitLow != 0 || grade.Grades[1].ExitLow != 0 {
		t.Fatalf("size-only first row exit bits = [%08x,%08x], want zero", grade.Grades[0].ExitLow, grade.Grades[1].ExitLow)
	}
	if grade.Grades[cTCPServerMaxSizeGradeNum].ExitLow != 0 {
		t.Fatalf("quality row exit bits = %08x, want zero", grade.Grades[cTCPServerMaxSizeGradeNum].ExitLow)
	}
}

func TestClearDataCommandIDMatches48Enum(t *testing.T) {
	if cTCPHCClearData != 0x0001 {
		t.Fatalf("cTCPHCClearData = 0x%04X, want 0x0001", uint32(cTCPHCClearData))
	}
}
