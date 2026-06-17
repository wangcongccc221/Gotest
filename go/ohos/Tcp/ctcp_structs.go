package tcp

import (
	"encoding/binary"
	"fmt"
	"unsafe"
)

// Structures below mirror the CTCP payload structs in 48/RSS/Base/interface.h.
// RawSize/Offset fields are local parser metadata, not part of the wire data.

const (
	cTCP48StSysConfigWireSize  = 504
	cTCP48StGradeInfoWireSize  = 11596
	cTCP48StParasWireSize      = 928
	cTCP48StMotorInfoWireSize  = 20
	cTCP48StGlobalExitInfoSize = 484
	cTCP48StExitInfoWireSize   = 304
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
	Grade StGradeInfo      // 等级信息  L4线宽 11596
	GExit StGlobalExitInfo //全局出口信息  已完成  484

	//#ifndef
	GWeight StGlobalWeightBaseInfo // 全局重量信息

	AnalogDensity StAnalogDensity // 水果设置界面  已完成 128
	Exit          [12]StExitInfo  // 出口信息  304   304*12= 3648
	Paras         [12]StParas     //IPM参数  11136

	Weights [12]StWeightBaseInfo //192

	Motor    [48]StMotorInfo //20*48=960 电机信息
	CFSMInfo [12]uint8
	CIPMInfo [12]uint8

	NSubsysId int32
	NVersion  int32 //版本号

	// L4版本没有 nNetState，只有 FSM 重启/模块标志。
	NFsmRestart uint8
	NFsmModule  uint8
}

type StSysConfig struct {
	ExitState              [384]uint8
	NChannelInfo           [4]uint8
	NImageUV               [4]uint8
	NDataRegistration      [4]uint8
	NImageSugar            [4]uint8
	NImageUltrasonic       [4]uint8
	NCameraDelay           [18]int32
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
	WeightClassifyTpye     uint8
	InternalClassifyType   uint8
	UltrasonicClassifyType uint8
	IfWIFIEnable           uint8
	CheckExit              uint8
	CheckNum               uint8

	//#if defined
	NIQSEnable uint8
}

type StGradeInfo struct {
	Intervals        [3]StColorIntervalItem // 3个颜色
	Percent          [48]StPercentInfo      //48个等级，每个等级3种颜色
	Grades           [256]StGradeItemInfo   // 36字节
	ExitEnabled      [2]int32
	ColorIntervals   [2]int32
	NExitSwitchNum   [48]int32
	NTagInfo         [6]uint8
	NFruitType       int32
	StrFruitName     [50]uint8
	UnFlawAreaFactor [12]uint32
	UnBruiseFactor   [12]uint32
	UnRotFactor      [12]uint32
	FDensityFactor   [6]float32
	FSugarFactor     [6]float32
	FAcidityFactor   [6]float32
	FHollowFactor    [6]float32
	FSkinFactor      [6]float32
	FBrownFactor     [6]float32
	FTangxinFactor   [6]float32
	FRigidityFactor  [6]float32
	FWaterFactor     [6]float32
	FShapeFactor     [6]float32

	StrSizeGradeName    [192]uint8
	StrQualityGradeName [192]uint8
	StDensityGradeName  [72]uint8
	StrColorGradeName   [192]uint8
	StrShapeGradeName   [72]uint8
	StFlawareaGradeName [72]uint8
	StBruiseGradeName   [72]uint8
	StRotGradeName      [72]uint8
	StSugarGradeName    [72]uint8
	StAcidityGradeName  [72]uint8
	StHollowGradeName   [72]uint8
	StSkinGradeName     [72]uint8
	StBrownGradeName    [72]uint8
	StTangxinGradeName  [72]uint8
	StRigidityGradeName [72]uint8
	StWaterGradeName    [72]uint8

	ColorType        uint8
	NLabelType       uint8
	NLabelbyExit     [48]uint8
	NSwitchLabel     [48]uint8
	NSizeGradeNum    uint8
	NQualityGradeNum uint8
	NClassifyType    uint8

	NCheckNum int16
	// L4下位机回包里没有 ForceChannel；否则 StGlobal.gexit 会整体后移 4 字节。
}

type StColorIntervalItem struct { //等级设置信息,发送给每一个FSM (HC_ID, FSM, HC_CMD_GRADE_INFO, stGradeInfo)
	NMinU uint8
	NMaxU uint8
	NMinV uint8
	NMaxV uint8
}

type StPercentInfo struct {
	NMax uint8
	NMin uint8
}

type stRange struct {
	NMax float32
	NMix float32
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

type StExitFlowInfo struct {
	No   int32
	Time int32
}

type StGlobalExitInfo struct { //484字节
	NPulse      uint8
	VersionFlag uint8
	NLabelPulse int16
	NDriverPin  [48]int16
	DelayTime   [48]float32
	HoldTime    [48]float32
}

type StGlobalWeightBaseInfo struct {
	FFilterParam           float32 // 4
	AD_Filter_ALG          uint8   // 1
	NEffectCupThreshold    int16
	NMinGradeThreshold     int16
	NCupDeviationThreshold int16
	NCupBreakageThreshold  int16
	NBaseCupNum            int16
	NTotalCupNums          [4]int16
	RefWeight              int16
	WeightTh               uint16
}

type StWeightGlobal struct {
	NAccuracy uint8
	NVersion  int32
	NWAMId    int32
	CFSMInfo  [30]uint8
	GWeight   StGlobalWeightBaseInfo
	Weights   [12]StWeightBaseInfo
}

type StAnalogDensity struct { //水果设置界面  //128
	UAnalogDensity [32]float32
}

type StOldGlobalWeightBaseInfo struct {
	fFilterParam           float32
	nMinGradeThreshold     int16
	nCupDeviationThreshold int16
	nCupBreakageThreshold  int16
	nBaseCupNum            int16
	nTotalCupNums          [4]int16
	RefWeight              int16
	WeightTh               uint8
}

type StExitInfo struct {
	LabelExit [4]StLabelItemInfo
	Exits     [48]StExitItemInfo
}

type StLabelItemInfo struct { // 贴标信息
	NDis       int16
	NDriverPin int16
}

type StExitItemInfo struct { // 出口信息
	NDis       int16
	NOffset    int16
	NDriverPin int16
}

type StParas struct { //IPM参数
	CameraParas   [3]StCameraParas
	IRCameraParas [6]StIRCameraParas
	NCupNum       int32
}

type StWeightBaseInfo struct { //IPM参数中的重量信息 16字节
	FGADParam          [2]float32
	FTemperatureParams float32
	WaveInterval       [2]uint16
}

type StResetAD struct {
	Value int32
}

// 电磁阀测试
type StVolveTest struct {
	ExitId uint8
}

type StOldWeightBaseInfo struct { //IPM参数中的重量信息 16字节
	fGADParam          [2]float32
	fTemperatureParams float32
	waveinterval       [2]uint8
}

type StCameraParas struct {
	MeanValue           StWhiteBalanceMean
	Cup                 [2]StFruitCup
	NROIOffsetY         [2]int32
	NTriggerDelay       int32
	NShutter            int32
	NDetectionThreshold [2]int32
	NDetectWhiteTh      [2]int32
	FGammaCorrection    float32
	FPixelRatio         [2]float32
	FFruitCupRangeTh    [2]float32
	NXYEdgeBreakTh      [2]uint8
	CCameraNum          uint8
}

type StWhiteBalanceMean struct { //隶属于StCameraParas
	MeanR int32
	MeanG int32
	MeanB int32
}

type StFruitCup struct { //隶属于StCameraParas
	NLeft    [2]int32
	NTop     int32
	NBottom  int32
	NOffsetX int32
	NOffsetY int32
}

type StIRCameraParas struct { //红外参数
	Cup                   [2]StFruitCup
	NROIOffsetY           [2]int32
	NTriggerDelay         int32
	NShutter              int32
	NIRDetectionThreshold [2]int32
	FGammaCorrection      float32
	FPixelRatio           [2]float32
	FFruitCupRangeTh      [2]float32
	NXYEdgeBreakTh        [2]uint8
	CCameraNum            uint8
}

type StMotorInfo struct {
	BExitId                  uint8
	BMotorSwitch             uint8
	NMotorEnableSwitchNum    int32
	NMotorEnableSwitchWeight int32
	FDelayTime               float32
	FHoldTime                float32
}

// 拼接4 字节成为8
func (s StGradeItemInfo) Exit() uint64 {
	return uint64(s.ExitLow) | (uint64(s.ExitHigh) << 32)
}

const (
	cTCPServerMaxNoticeLength    = 30
	cTCPServerMaxQualityGradeNum = 16

	stStatisticsExpectedSize = 7152
)

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

// 水果子系统信息
type SysStStatistics struct {
	NTotalCount  [4]uint64
	NWeightCount [4]uint64
}

// 水果实时分级信息
type StFruitVisionParam struct {
	UnColorRate0   uint32
	UnColorRate1   uint32
	UnColorRate2   uint32
	UnArea         uint32
	UnFlawArea     uint32
	UnVolume       uint32
	UnFlawNum      uint32
	UnMaxR         float32
	UnMinR         float32
	UnSelectBasis  float32
	FDiameterRatio float32
	FMinDRatio     float32
}

// 水果实时分级信息，紫外线相机的IPM发送给FSM，FSM转发过来
type StFruitUVParam struct {
	UnBruiseArea uint32
	UnBruiseNum  uint32
	UnRotArea    uint32
	UnRotNumy    uint32
	UnRigidity   uint32
	UnWater      uint32
	UnTimeTag    uint32
}

// 水果实时分级信息，NIR(含糖量检测仪)发送给FSM,FSM转发过来
type StNIRParam struct {
	FSugar    float32
	FAcidity  float32
	FHollow   float32
	FSkin     float32
	FBrown    float32
	FTangxin  float32
	UnTimeTag uint32
}

// 水果实时分级信息，FSM发送过来 (FSM, HC_ID, FSM_CMD_GRADEINFO, stFruitGradeInfo)
type StFruitParam struct {
	VisionParam StFruitVisionParam
	UVParam     StFruitUVParam
	NIRParam    StNIRParam
	FWeight     float32
	FDensity    float32
	UnGrade     uint32

	UnWhichExit uint8
}

type StFruitGradeInfo struct {
	Param    [2]StFruitParam //IPM的id号
	NRouteId int32
}

// 重量统计信息,FSM发送过来 (WM, HC_ID, FSM_CMD_WEIGHTINFO, stWeightResult
type StWeightStat struct {
	FCupAverageWeight float32
	NAD0              uint16
	NAD1              uint16
	NStandardAD0      uint16
	NStandardAD1      uint16
}

type StTrackingData struct {
	NVehicleId     int32
	FFruitWeight   float32
	FVehicleWeight float32

	NADFruit   uint16
	NADVehicle uint16
}

type StWeightResult struct {
	Data            StTrackingData //追踪数据
	Paras           StWeightStat   //统计信息
	NChannelId      int32
	FVehicleWeight0 float32
	FVehicleWeight1 float32
	State           uint8
}

// 波形数据,FSM发送过来 (WM, HC_ID, FSM_CMD_WAVEINFO, stWaveInfo)
type StWaveInfo struct {
	NChannelId  int32
	Waveform0   [256]uint16
	Waveform1   [256]uint16
	FruitWeight float32
}

// 自定义结构体
type Rect struct {
	Top      int32
	Bottom   int32
	Left     int32
	Right    int32
	NOffsetX int32
	NOffsetY int32
}

type ColorRGB struct {
	UCR uint8
	UCG uint8
	UCB uint8
}

type QaulGradeInfo struct {
	QualName      [12]byte
	ColorIndex    int32
	ShapeIndex    int32
	DensityIndex  int32
	FlawIndex     int32
	BruiseIndex   int32
	RotIndex      int32
	SugarIndex    int32
	AcidityIndex  int32
	HollowIndex   int32
	SkinIndex     int32
	BrownIndex    int32
	TangxinIndex  int32
	RigidityIndex int32
	WaterIndex    int32
}

type QaulityGradeItem struct {
	GradeName    [12]uint8
	ColorGrade   int8
	SbShapeGrade int8
	SbDensity    int8
	SbFlaw       int8
	SbBruise     int8
	SbRot        int8
	SbSugar      int8
	SbAcidity    int8
	SbHollow     int8
	SbSkin       int8
	SbBrown      int8
	SbTangxin    int8
	SbRigidity   int8
	SbWater      int8
	FruitNum     int32
}

type QualityGradeInfo struct {
	Item     [16]QaulityGradeItem
	GradeCnt int32
}

type ExitState struct {
	Index       int32
	ColumnIndex int32
	ItemIndex   int32
}

type TempExitState struct {
	Index       int32
	ColumnIndex int32
	ItemIndex   int32
}

// / 颜色界面-》颜色列表背景颜色
type StColorList struct {
	Color1 [12]uint8
	Color2 [12]uint8
	Color3 [12]uint8
}

type StClientInfo struct {
	CustomerName [20]uint8
	FarmName     [20]uint8
	FruitName    [20]uint8
}

type StClientNewInfo struct {
	CustomerName string
	FarmName     string
	FruitName    string
}

type StOldClientInfo struct {
	CustomerName [50]uint8
	FarmName     [50]uint8
	FruitName    [50]uint8
}

// / 水果种类成员（对应IPM算法）
type StFruitTypeMember struct {
	IFruitTypeID int32
	BFruitName   [20]uint8
}

// / 水果种类（对应IPM算法）
type StFruitType struct {
	ICurrentFruitNumber int32
	Member              [32 * 8]StFruitTypeMember
}

// 基本统计信息,HC->平板
type StBroadcastStatistics struct {
	Statistics            StStatistics
	StrStartTime          [12]uint8
	FSeparationEfficiency float32
	FRealWeightCount      float32
	StrProgramName        [12]uint8
	StrLabelName          [4 * 12]uint8
}

// 平板系统信息,HC->平板
type StBroadcastSysConfig struct {
	SysConfig       StSysConfig
	NLanguage       int32
	ExitDisplayType int64
	StrDisplayName  [48 * 20]uint8
}

// 出口附加信息，HC->平板（在stBroadcastSysConfig数据包之后发送）
type StExitAdditionalTextData struct {
	Additionaldata [48 * 100]byte
}

// 出口等级信息,HC->触摸屏  1字节对齐
type StExitGradeItemInfo struct { //先拆分 后面拼接 成完整的出口ID和等级信息
	ExitIndexLow  uint16
	ExitIndexHigh uint16
	GradeName     [26]byte
}

type StExitGradeInfo struct {
	ExitGrade [48]StExitGradeItemInfo
}

type StFruitGradeInfos struct {
	FruitGradeInfos [12]StFruitGradeInfo
}

const (
	MAX_EXIT_NUM    = 48
	MAX_TEXT_LENGTH = 12
)

type StExitInfos struct {
	ExitName        [MAX_EXIT_NUM * MAX_TEXT_LENGTH]uint8
	ExitControlMode [MAX_EXIT_NUM]uint8
	ExitBoxType     [MAX_EXIT_NUM]uint8
}

// 内部品质----------------------------------------------------------------------------------------------------

// 语言包
type LanguageType int32

const (
	Chinese LanguageType = 0x0
	English LanguageType = 0x1
	Spanish LanguageType = 0x2
)

// -----------------------------------------------------
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
