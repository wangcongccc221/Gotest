package tcp

import (
	"bytes"
	"encoding/binary"
	"fmt"
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
	Intervals        [3]StColorIntervalItem  // 3个颜色
	percent          [16 * 3]StPercentInfo   //16个等级，每个等级3种颜色
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

// Exit 将 ExitLow/ExitHigh 拼成与 C `ulong exit` 等价的 uint64（小端：低 32 位在前）。
func (s StGradeItemInfo) Exit() uint64 {
	return uint64(s.ExitLow) | (uint64(s.ExitHigh) << 32)
}

// StGradeItemInfoWire 对应 C 端 #pragma pack(4) 的 StGradeItemInfo 在信道上的 36 字节布局（sizeof 恒为 36）。
type StGradeItemInfoWire [stGradeItemInfoQtLinux64Pack4ReferenceSize]byte

func init() {
	if unsafe.Sizeof(StGradeItemInfo{}) != unsafe.Sizeof(StGradeItemInfoWire{}) {
		panic(fmt.Sprintf("StGradeItemInfo sizeof %d must equal StGradeItemInfoWire %d (与 C 36 字节线布局一致)",
			unsafe.Sizeof(StGradeItemInfo{}), unsafe.Sizeof(StGradeItemInfoWire{})))
	}
}

// StGradeItemInfoWireFromSlice 从缓冲区拷贝 36 字节包体。
func StGradeItemInfoWireFromSlice(data []byte) (StGradeItemInfoWire, error) {
	var w StGradeItemInfoWire
	if len(data) < len(w) {
		return w, fmt.Errorf("StGradeItemInfoWire: need %d bytes, got %d", len(w), len(data))
	}
	copy(w[:], data[:len(w)])
	return w, nil
}

// UnpackStGradeItemInfo 将 36 字节 pack(4) 包体解成 StGradeItemInfo（直接 binary.Read 到本结构体）。
func UnpackStGradeItemInfo(w StGradeItemInfoWire) StGradeItemInfo {
	var g StGradeItemInfo
	_ = binary.Read(bytes.NewReader(w[:]), binary.LittleEndian, &g)
	return g
}

// PackStGradeItemInfo 将 StGradeItemInfo 打成 36 字节包体（直接 binary.Write）。
func PackStGradeItemInfo(g StGradeItemInfo) StGradeItemInfoWire {
	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.LittleEndian, &g)
	var w StGradeItemInfoWire
	copy(w[:], buf.Bytes())
	return w
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
