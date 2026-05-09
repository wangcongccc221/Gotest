package tcp

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
type StGlobalExitInfo struct { //全局出口
	nPulse      uint8
	versionFlag uint8
	nLabelPulse uint16
	nDriverPin  [cTCPServerStSysConfigExit48]uint8
	Delay_time  [cTCPServerStSysConfigExit48]float32
	Hold_time   [cTCPServerStSysConfigExit48]float32
}

const (
	cTCPServerStatisticsMaxExitNum = cTCPServerStSysConfigExit64
	cTCPServerMaxNoticeLength      = 30
)

// StStatistics mirrors RSS/Base/interface.h on 64-bit Linux with MAX64 enabled
// and WEIGHTANDSIZE disabled. C++ ulong is unsigned long under LP64, so it is
// represented as uint64 here.
type StStatistics struct {
	NGradeCount           [cTCPServerMaxQualityGradeNum * cTCPServerMaxSizeGradeNum]uint64
	NWeightGradeCount     [cTCPServerMaxQualityGradeNum * cTCPServerMaxSizeGradeNum]float64
	NExitCount            [cTCPServerStatisticsMaxExitNum]uint64
	NExitWeightCount      [cTCPServerStatisticsMaxExitNum]float64
	NChannelTotalCount    [cTCPServerStSysConfigMaxChan]uint64
	NChannelWeightCount   [cTCPServerStSysConfigMaxChan]float64
	NSubsysId             int32
	NBoxGradeCount        [cTCPServerMaxQualityGradeNum * cTCPServerMaxSizeGradeNum]int32
	NBoxGradeWeight       [cTCPServerMaxQualityGradeNum * cTCPServerMaxSizeGradeNum]float64
	NTotalCupNum          int32
	NInterval             int32
	NIntervalSumperminute int32
	NCupState             uint16
	NPulseInterval        uint16
	NUnpushFruitCount     uint16
	NNetState             uint16
	NWeightSetting        uint16
	NSCMState             int32
	NIQSNetState          uint16 //quint8
	NLockState            uint8
	ExitBoxNum            [cTCPServerStatisticsMaxExitNum]uint16
	ExitWeight            [cTCPServerStatisticsMaxExitNum]float64
	Notice                [cTCPServerMaxNoticeLength]uint8
}
