package tcp

import (
	"encoding/binary"
	"fmt"
	"unsafe"
)

// Structures below mirror the CTCP payload structs in 48/RSS/Base/interface.h.
// RawSize/Offset fields are local parser metadata, not part of the wire data.

type StSysConfig struct {
	MaxExitNum             int
	NChannelInfo           [cTCPServerMaxSubsysNum]uint8
	NImageUV               [cTCPServerMaxSubsysNum]uint8
	NDataRegistration      [cTCPServerMaxSubsysNum]uint8
	NImageSugar            [cTCPServerMaxSubsysNum]uint8
	NImageUltrasonic       [cTCPServerMaxSubsysNum]uint8
	NCameraDelay           [cTCPServerMaxCameraNum * 2]int32
	Width                  int32
	Height                 int32
	PacketSize             int32
	NSystemInfo            uint16
	NSubsysNum             uint8
	NExitNum               uint8
	NClassificationInfo    uint8
	MultiFreq              uint8
	NCameraType            uint8
	CIRClassifyType        uint8
	UVClassifyType         uint8
	WeightClassifyType     uint8
	InternalClassifyType   uint8
	UltrasonicClassifyType uint8
	IfWIFIEnable           uint8
	CheckExit              uint8
	CheckNum               uint8
	NIQSEnable             uint8
	RawSize                int
}

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

type StGradeInfo struct {
	Offset           int
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

type StGlobalExitInfo struct { //全局出口
	nPulse      uint8
	versionFlag uint8
	nLabelPulse uint16
	nDriverPin  [cTCPServerStSysConfigExit48]uint8
	Delay_time  [cTCPServerStSysConfigExit48]float32
	Hold_time   [cTCPServerStSysConfigExit48]float32
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
