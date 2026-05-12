package tcp

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"unsafe"
)

// Structures below mirror the CTCP payload structs in 48/RSS/Base/interface.h.
// RawSize/Offset fields are local parser metadata, not part of the wire data.

const stGradeItemInfoQtLinux64Pack4ReferenceSize = 36

func stGradeItemInfoGoSize() uintptr {
	return unsafe.Sizeof(StGradeItemInfo{})
}

func stGradeItemInfoWireSize() int {
	return len(StGradeItemInfoWire{})
}

func stGradeItemInfoQtLinux64Pack4Size() int {
	return stGradeItemInfoQtLinux64Pack4ReferenceSize
}

func stGradeItemInfoSizeSummary() string {
	return fmt.Sprintf(
		"StGradeItemInfo size(go=%d, wire=%d, qt_linux64_pack4=%d)",
		stGradeItemInfoGoSize(),
		stGradeItemInfoWireSize(),
		stGradeItemInfoQtLinux64Pack4Size(),
	)
}

// StGradeItemInfoLayoutReport 返回当前编译目标下 StGradeItemInfo 的真实 sizeof/alignof 及各字段 offset（调试用）。
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

type CTCPConfigSnapshot struct {
	ServerName string
	Port       int
	RemoteAddr string
	SrcID      int32
	DstID      int32
	CmdID      int32
	ReceivedAt int64
	RawPayload []byte
	SysConfig  StSysConfig
	GradeInfo  StGradeInfo
}

/*
go           QT  linux 64
uint64   // ulong    8字节
float64  // double   8字节
float32  // float    4字节
int32    // int      4
uint16   // ushort / quint16    2
uint8    // quint8       1

*/

type StGlobal struct { //全局出口
	sys   StSysConfig
	grade StGradeInfo
}

type StSysConfig struct {
	exitstate              [48 * 2 * 4]uint8
	nChannelInfo           [4]uint8
	nImageUV               [4]uint8
	nDataRegistration      [4]uint8
	nImageSugar            [4]uint8
	nImageUltrasonic       [4]uint8
	nCameraDelay           [4 * 2]int32
	width                  int32
	height                 int32
	packetSize             int32
	nSystemInfo            uint16
	nSubsysNum             uint8
	nExitNum               uint8
	nClassificationInfo    uint8
	multiFreq              uint8
	nCameraType            uint8
	CIRClassifyType        uint8
	UVClassifyType         uint8
	WeightClassifyTpye     uint8
	InternalClassifyType   uint8
	UltrasonicClassifyType uint8
	IfWIFIEnable           uint8
	CheckExit              uint8
	CheckNum               uint8

	//#if defined
	nIQSEnable uint8
}

type StGradeInfo struct {
	Intervals        [3]StColorIntervalItem      // 3个颜色
	percent          [16 * 3]StPercentInfo       //16个等级，每个等级3种颜色
	grades           [16 * 3]StGradeItemInfoWire // C 侧 #pragma pack(4)，每项固定 36 字节
	ExitEnabled      [2]int32
	ColorIntervals   [2]int32
	nExitSwitchNum   [48]int32
	nTagInfo         [6]uint8
	nFruitType       int32
	strFruitName     [50]uint8
	unFlawAreaFactor [12]uint32
	unBruiseFactor   [12]uint32
	unRotFactor      [12]uint32
	fDensityFactor   [12]float32
	fSugarFactor     [12]float32
}

type StColorIntervalItem struct { //等级设置信息,发送给每一个FSM (HC_ID, FSM, HC_CMD_GRADE_INFO, stGradeInfo)
	nMinU uint8
	nMaxU uint8
	nMinV uint8
	nMaxV uint8
}

type StPercentInfo struct {
	nMax uint8
	nMin uint8
}

// 4字节对齐
// #pack(4) --- IGNORE ---
type StGradeItemInfo struct {
	// 对应 C 端 ulong exit 的小端 8 字节：先低 32 位、再高 32 位（与线上一致）。
	ExitLow  uint32 // offset 0
	ExitHigh uint32 // offset 4

	NMinSize  float32 // offset 8
	NMaxSize  float32 // offset 12
	NFruitNum int32   // offset 16

	NColorGrade    int8 // offset 20
	SbShapeSize    int8 // offset 21
	SbDensity      int8 // offset 22
	SbFlawArea     int8 // offset 23
	SbBruise       int8 // offset 24
	SbRot          int8 // offset 25
	SbSugar        int8 // offset 26
	SbAcidity      int8 // offset 27
	SbHollow       int8 // offset 28
	SbSkin         int8 // offset 29
	SbBrown        int8 // offset 30
	SbTangxin      int8 // offset 31
	SbRigidity     int8 // offset 32
	SbWater        int8 // offset 33
	SbLabelbyGrade int8 // offset 34

	// TailPad C 端 #pragma pack(4) 第 36 字节（编译器尾填充）；与 StGradeItemInfoWire 最后一字节对应。
	TailPad byte // offset 35
}

// 拼接4 字节成为8
func (s StGradeItemInfo) Exit() uint64 {
	return uint64(s.ExitLow) | (uint64(s.ExitHigh) << 32)
}

const (
	cTCPServerStatisticsMaxExitNum = cTCPServerStSysConfigExit48
	cTCPServerMaxNoticeLength      = 30
	cTCPServerMaxQualityGradeNum   = 16

	// 与当前 StStatistics 定义一致（unsafe.Sizeof）；binary.Size 可能因尾部 padding 少几字节。
	stStatisticsExpectedSize = 7152
)

// StStatistics 对应 struct StStatistics。

/*
go           QT  linux 64
uint64   // ulong    8字节
float64  // double   8字节
int32    // int      4
uint16   // ushort / quint16    2
uint8    // quint8       1

*/

// ---------------------------------------------------------------------------------------------------
type StStatistics struct {
	NGradeCount         [256]uint64
	NWeightGradeCount   [256]uint64
	NExitCount          [48]uint64
	NExitWeightCount    [48]uint64
	NChannelTotalCount  uint64
	NChannelWeightCount uint64
	NSubsysId           int32
	NBoxGradeCount      [256]int32
	// Pad1                  [4]byte
	NBoxGradeWeight [256]int32
	//------------
	NTotalCupNum          int32
	NInterval             int32
	NIntervalSumperminute int32
	NCupState             uint16
	NPulseInterval        uint16
	NUnpushFruitCount     uint16

	//#if defined
	NNetState      uint8 //1字节
	NWeightSetting uint8

	NSCMState    uint8
	NIQSNetState uint8
	NLockState   uint8

	ExitBoxNum [48]uint32
}

func stStatisticsWireSize() int {
	return binary.Size(StStatistics{})
}

func stStatisticsSizeSummary() string {
	return fmt.Sprintf(
		"StStatistics size(go=%d, wire=%d, expected=%d)",
		unsafe.Sizeof(StStatistics{}),
		stStatisticsWireSize(),
		stStatisticsExpectedSize,
	)
}

//---------------------------------------------------------------------------------------------------
