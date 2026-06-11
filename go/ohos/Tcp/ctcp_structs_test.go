package tcp

import (
	"testing"
	"unsafe"
)

func TestStGlobalSize(t *testing.T) {
	goSz := int(unsafe.Sizeof(StGlobal{}))
	want := cTCP48StGlobalExpectedSize
	if goSz == want {
		return
	}
	gw := int(unsafe.Sizeof(StGlobalWeightBaseInfo{}))
	t.Fatalf("sizeof(StGlobal)=%d，线宽/预期=%d，相差 %d 字节。请先打开 interface.h 里 StGlobalWeightBaseInfo，"+
		"用 offsetof 或从首成员累加对齐，核对 WeightTh 之后是否还有未写入 Go 的成员；"+
		"当前 sizeof(StGlobalWeightBaseInfo)=%d。禁止用匿名 _ [n]byte 补齐。",
		goSz, want, want-goSz, gw)
}d

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
