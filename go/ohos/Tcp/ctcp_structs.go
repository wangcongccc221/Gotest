package tcp

import (
	"encoding/binary"
	"fmt"
	"unsafe"
)

// Structures below mirror the CTCP payload structs in 48/RSS/Base/interface.h.
// RawSize/Offset fields are local parser metadata, not part of the wire data.

type StGradeItemInfo struct {
	Exit           uint64
	NMinSize       float32
	NMaxSize       float32
	NFruitNum      int32
	NColorGrade    int8
	SBShapeSize    int8
	SBDensity      int8
	SBFlawArea     int8
	SBBruise       int8
	SBRot          int8
	SBSugar        int8
	SBAcidity      int8
	SBHollow       int8
	SBSkin         int8
	SBBrown        int8
	SBTangxin      int8
	SBRigidity     int8
	SBWater        int8
	SBLabelByGrade int8
}

const stGradeItemInfoQtLinux64Pack4ReferenceSize = 36

func stGradeItemInfoGoSize() uintptr {
	return unsafe.Sizeof(StGradeItemInfo{})
}

func stGradeItemInfoWireSize() int {
	return binary.Size(StGradeItemInfo{})
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
int32    // int      4
uint16   // ushort / quint16    2
uint8    // quint8       1

*/

type StGlobal struct { //全局出口
	sys StSysConfig
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
	Intervals        [3]StColorIntervalItem // 3个颜色
	RawSize          int
	MaxExitNum       int
	GradeItemSize    int
	ExitEnabled      [2]int32
	ColorIntervals   [2]int32
	NFruitType       int32
	FruitName        string
	ColorType        uint8
	NLabelType       uint8
	NSizeGradeNum    uint8
	NQualityGradeNum uint8
	NClassifyType    uint8
	NCheckNum        int16
	ForceChannel     int16
	FirstGrade       StGradeItemInfo
}

type StColorIntervalItem struct { //等级设置信息,发送给每一个FSM (HC_ID, FSM, HC_CMD_GRADE_INFO, stGradeInfo)
	nMinU uint8
	nMaxU uint8
	nMinV uint8
	nMaxV uint8
}

const (
	cTCPServerStatisticsMaxExitNum = cTCPServerStSysConfigExit48
	cTCPServerMaxNoticeLength      = 30
	cTCPServerMaxQualityGradeNum   = 16

	stStatisticsExpectedSize = 9096
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
