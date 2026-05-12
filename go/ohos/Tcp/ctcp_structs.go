package tcp

import (
	"encoding/binary"
	"fmt"
	"unsafe"
)

// Structures below mirror the CTCP payload structs in 48/RSS/Base/interface.h.
// RawSize/Offset fields are local parser metadata, not part of the wire data.

// 48/RSS（interface.h 注释区 ConstPreDefine）硬编码线宽，与下位机 StGlobal 对齐用。
const (
	cTCP48StGradeInfoWireSize  = 11600
	cTCP48StGlobalExpectedSize = 28712
	cTCP48MaxChannelNum        = 12
	cTCP48MaxIPMNum            = 12
	cTCP48MaxExitNum           = 48
	cTCP48MaxLabelNum          = 4
	cTCP48ChannelNumPerIPM     = 2
	cTCP48MaxColorCamera       = 3
	cTCP48MaxNIRCamera         = 6
	cTCP48AnalogDensitySlots   = 32 // MAX_FRUIT_TYPE_MAJOR_CLASS_NUM
	cTCP48MaxCameraDelayInts   = cTCPServerMaxCameraNum * 2
)

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
	Global     StGlobal
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

type StGlobal struct {
	Sys   StSysConfig      // 系统配置   已完成  504
	grade StGradeInfo      // 等级信息  已完成  11600
	gexit StGlobalExitInfo //全局出口信息  已完成 484

	//#ifndef
	gweight StGlobalWeightBaseInfo // 全局重量信息

	analogdensity StAnalogDensity // 水果设置界面  已完成 128
	exit          [12]StExitInfo  // 出口信息  304   304*12= 3648
	paras         [12]StParas     //IPM参数  11136

	weights [12]StWeightBaseInfo //192

	motor    [48]StMotorInfo //20*48=960 电机信息
	cFSMInfo [12]uint8
	cIPMInfo [12]uint8

	nSubsysId int32
	nVersion  int32 //版本号

	// 先注释 后面在使用 查找一下问题
	// nNetState   uint8
	// nFsmRestart uint8
	// nFsmModule  uint8
}

type StSysConfig struct {
	exitstate              [384]uint8
	nChannelInfo           [4]uint8
	nImageUV               [4]uint8
	nDataRegistration      [4]uint8
	nImageSugar            [4]uint8
	nImageUltrasonic       [4]uint8
	nCameraDelay           [18]int32
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
	percent          [48]StPercentInfo      //48个等级，每个等级3种颜色
	grades           [256]StGradeItemInfo   // 36字节
	ExitEnabled      [2]int32
	ColorIntervals   [2]int32
	nExitSwitchNum   [48]int32
	nTagInfo         [6]uint8
	nFruitType       int32
	strFruitName     [50]uint8
	unFlawAreaFactor [12]uint32
	unBruiseFactor   [12]uint32
	unRotFactor      [12]uint32
	fDensityFactor   [6]float32
	fSugarFactor     [6]float32
	fAcidityFactor   [6]float32
	fHollowFactor    [6]float32
	fSkinFactor      [6]float32
	fBrownFactor     [6]float32
	fTangxinFactor   [6]float32
	fRigidityFactor  [6]float32
	fWaterFactor     [6]float32
	fShapeFactor     [6]float32

	strSizeGradeName    [192]uint8
	strQualityGradeName [192]uint8
	stDensityGradeName  [72]uint8
	strColorGradeName   [192]uint8
	strShapeGradeName   [72]uint8
	stFlawareaGradeName [72]uint8
	stBruiseGradeName   [72]uint8
	stRotGradeName      [72]uint8
	stSugarGradeName    [72]uint8
	stAcidityGradeName  [72]uint8
	stHollowGradeName   [72]uint8
	stSkinGradeName     [72]uint8
	stBrownGradeName    [72]uint8
	stTangxinGradeName  [72]uint8
	stRigidityGradeName [72]uint8
	stWaterGradeName    [72]uint8

	ColorType        uint8
	nLabelType       uint8
	nLabelbyExit     [48]uint8
	nSwitchLabel     [48]uint8
	nSizeGradeNum    uint8
	nQualityGradeNum uint8
	nClassifyType    uint8

	nCheckNum int16
	//ifdef
	ForceChannel int16
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
type StGradeItemInfo struct {
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

}

type StGlobalExitInfo struct { //484字节
	nPulse      uint8
	versionFlag uint8
	nLabelPulse int16
	nDriverPin  [48]int16
	Delay_time  [48]float32
	Hold_time   [48]float32
}

type StGlobalWeightBaseInfo struct {
	fFilterParam           float32 // 4
	AD_Filter_ALG          uint8   // 1
	nEffectCupThreshold    int16
	nMinGradeThreshold     int16
	nCupDeviationThreshold int16
	nCupBreakageThreshold  int16
	nBaseCupNum            int16
	nTotalCupNums          [4]int16
	RefWeight              int16
	WeightTh               uint8
}

type StAnalogDensity struct { //水果设置界面  //128
	uAnalogDensity [32]float32
}

type StExitInfo struct {
	labelexit [4]StLabelItemInfo
	exits     [48]StExitItemInfo
}

type StLabelItemInfo struct { // 贴标信息
	nDis       int16
	nDriverPin int16
}

type StExitItemInfo struct { // 出口信息
	nDis       int16
	nOffset    int16
	nDriverPin int16
}

type StParas struct { //IPM参数
	cameraParas   [3]StCameraParas
	irCameraParas [6]StIRCameraParas
	nCupNum       int32
}

type StWeightBaseInfo struct { //IPM参数中的重量信息 16字节
	fGADParam          [2]float32
	fTemperatureParams float32
	waveinterval       [2]uint16
}
type StCameraParas struct {
	MeanValue           StWhiteBalanceMean
	cup                 [2]StFruitCup
	nROIOffsetY         [2]int32
	nTriggerDelay       int32
	nShutter            int32
	nDetectionThreshold [2]int32
	nDetectWhiteTh      [2]int32
	fGammaCorrection    float32
	fPixelRatio         [2]float32
	fFruitCupRangeTh    [2]float32
	nXYEdgeBreakTh      [2]uint8
	cCameraNum          uint8
}

type StWhiteBalanceMean struct { //隶属于StCameraParas
	MeanR int32
	MeanG int32
	MeanB int32
}

type StFruitCup struct { //隶属于StCameraParas
	nLeft    [2]int32
	nTop     int32
	nBottom  int32
	nOffsetX int32
	nOffsetY int32
}

type StIRCameraParas struct { //红外参数
	cup                   [2]StFruitCup
	nROIOffsetY           [2]int32
	nTriggerDelay         int32
	nShutter              int32
	nIRDetectionThreshold [2]int32
	fGammaCorrection      float32
	fPixelRatio           [2]float32
	fFruitCupRangeTh      [2]float32
	nXYEdgeBreakTh        [2]uint8
	cCameraNum            uint8
}

type StMotorInfo struct {
	bExitId                  uint8
	bMotorSwitch             uint8
	nMotorEnableSwitchNum    int32
	nMotorEnableSwitchWeight int32
	fDelay_time              float32
	fHold_time               float32
}

// 拼接4 字节成为8
func (s StGradeItemInfo) Exit() uint64 {
	return uint64(s.ExitLow) | (uint64(s.ExitHigh) << 32)
}

const (
	cTCPServerStatisticsMaxExitNum = cTCPServerStSysConfigExit48
	cTCPServerMaxNoticeLength      = 30
	cTCPServerMaxQualityGradeNum   = 16

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
