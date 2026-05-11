package tcp

import (
	"testing"
	"unsafe"
)

func TestStGradeItemInfoSizeReference(t *testing.T) {
	if got := stGradeItemInfoWireSize(); got != 35 {
		t.Fatalf("stGradeItemInfoWireSize() = %d, want 35", got)
	}

	if got := stGradeItemInfoQtLinux64Pack4Size(); got != 36 {
		t.Fatalf("stGradeItemInfoQtLinux64Pack4Size() = %d, want 36", got)
	}
}

func TestStStatisticsSize(t *testing.T) {
	if got := stStatisticsWireSize(); got != stStatisticsExpectedSize {
		t.Fatalf("stStatisticsWireSize() = %d, want %d (%s)",
			got, stStatisticsExpectedSize, stStatisticsSizeSummary())
	}
	if got := int(unsafe.Sizeof(StStatistics{})); got != stStatisticsExpectedSize {
		t.Fatalf("unsafe.Sizeof(StStatistics{}) = %d, want %d (%s)",
			got, stStatisticsExpectedSize, stStatisticsSizeSummary())
	}
}
