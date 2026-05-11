package tcp

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"unsafe"
)

// StGradeItemInfoLayoutReport 返回当前编译目标下（本 so 真实运行环境）StGradeItemInfo 的 sizeof / alignof / binary.Size
// 以及 reflect 给出的各字段 offset、size（用于与鸿蒙设备上 C 结构体对齐核对）。
func StGradeItemInfoLayoutReport() string {
	var z StGradeItemInfo
	var w StGradeItemInfoWire
	t := reflect.TypeOf(z)
	var b strings.Builder
	fmt.Fprintf(&b, "=== StGradeItemInfo（Exit 已拆为 ExitLow+ExitHigh）真实内存布局 ===\n")
	fmt.Fprintf(&b, "GOOS=%s GOARCH=%s（当前这份 libohos 的编译目标）\n", runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(&b, "真实 unsafe.Sizeof(StGradeItemInfo)=%d 字节\n", unsafe.Sizeof(z))
	fmt.Fprintf(&b, "真实 unsafe.Alignof(StGradeItemInfo)=%d\n", unsafe.Alignof(z))
	fmt.Fprintf(&b, "encoding/binary.Size(StGradeItemInfo)=%d\n", binary.Size(z))
	fmt.Fprintf(&b, "StGradeItemInfoWire=[%d]byte sizeof=%d（线长，应与上式一致）\n", len(w), unsafe.Sizeof(w))
	if unsafe.Sizeof(z) != unsafe.Sizeof(w) {
		fmt.Fprintf(&b, "警告: sizeof(结构体) != sizeof(Wire)，binary.Read/Write 与 C 36 字节对齐可能有问题\n")
	}
	fmt.Fprintf(&b, "stGradeItemInfoWireSize()=%d\n", stGradeItemInfoWireSize())
	fmt.Fprintf(&b, "字段 reflect offset/size:\n")
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fmt.Fprintf(&b, "  %s offset=%d size=%d\n", f.Name, f.Offset, f.Type.Size())
	}
	return b.String()
}
