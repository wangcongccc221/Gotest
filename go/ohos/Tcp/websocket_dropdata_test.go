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
