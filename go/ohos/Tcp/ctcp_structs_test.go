package tcp

import (
	"testing"
	"unsafe"
)

func TestStStatisticsSize(t *testing.T) {
	goSz := int(unsafe.Sizeof(StStatistics{}))
	wireSz := stStatisticsWireSize()
	if goSz != stStatisticsExpectedSize {
		t.Fatalf("unsafe.Sizeof(StStatistics{}) = %d, want %d (%s)",
			goSz, stStatisticsExpectedSize, stStatisticsSizeSummary())
	}
	if wireSz > goSz {
		t.Fatalf("stStatisticsWireSize() = %d > sizeof %d (%s)", wireSz, goSz, stStatisticsSizeSummary())
	}
}
