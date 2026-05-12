package tcp

import (
	"bytes"
	"encoding/binary"
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
	if got := int(unsafe.Sizeof(StGradeItemInfo{})); got != 36 {
		t.Fatalf("sizeof(StGradeItemInfo{}) = %d, want 36", got)
	}
	g0 := StGradeItemInfo{ExitLow: 1, ExitHigh: 0, NMinSize: 2, NMaxSize: 3, NFruitNum: 4, NColorGrade: 5, SbLabelbyGrade: 6, TailPad: 0xAB}
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, &g0); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 36 {
		t.Fatalf("binary.Write len = %d, want 36", buf.Len())
	}
	var g1 StGradeItemInfo
	if err := binary.Read(bytes.NewReader(buf.Bytes()), binary.LittleEndian, &g1); err != nil {
		t.Fatal(err)
	}
	if g1 != g0 {
		t.Fatalf("binary roundtrip mismatch: %+v vs %+v", g1, g0)
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
