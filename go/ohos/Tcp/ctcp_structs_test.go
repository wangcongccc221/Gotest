package tcp

import (
	"testing"
	"unsafe"
)

func TestStGradeItemInfoSizeReference(t *testing.T) {
	if got := stGradeItemInfoWireSize(); got != 36 {
		t.Fatalf("stGradeItemInfoWireSize() = %d, want 36", got)
	}

	if got := stGradeItemInfoQtLinux64Pack4Size(); got != 36 {
		t.Fatalf("stGradeItemInfoQtLinux64Pack4Size() = %d, want 36", got)
	}
	var w StGradeItemInfoWire
	if len(w) != 36 {
		t.Fatalf("len(StGradeItemInfoWire{}) = %d, want 36", len(w))
	}
	if got := int(unsafe.Sizeof(StGradeItemInfo{})); got != 36 {
		t.Fatalf("sizeof(StGradeItemInfo{}) = %d, want 36", got)
	}
	g0 := StGradeItemInfo{ExitLow: 1, ExitHigh: 0, NMinSize: 2, NMaxSize: 3, NFruitNum: 4, NColorGrade: 5, SbLabelbyGrade: 6, TailPad: 0xAB}
	w1 := PackStGradeItemInfo(g0)
	g1 := UnpackStGradeItemInfo(w1)
	if g1 != g0 {
		t.Fatalf("Pack/Unpack roundtrip mismatch: %+v vs %+v", g1, g0)
	}
}

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
