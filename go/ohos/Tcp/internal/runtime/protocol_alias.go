package tcp

import "gotest/ohos/Tcp/protocol"

type StGlobal = protocol.StGlobal
type StSysConfig = protocol.StSysConfig
type StGradeInfo = protocol.StGradeInfo
type StColorIntervalItem = protocol.StColorIntervalItem
type StPercentInfo = protocol.StPercentInfo
type StGradeItemInfo = protocol.StGradeItemInfo
type StExitFlowInfo = protocol.StExitFlowInfo
type StGlobalExitInfo = protocol.StGlobalExitInfo
type StGlobalWeightBaseInfo = protocol.StGlobalWeightBaseInfo
type StWeightGlobal = protocol.StWeightGlobal
type StAnalogDensity = protocol.StAnalogDensity
type StOldGlobalWeightBaseInfo = protocol.StOldGlobalWeightBaseInfo
type StExitInfo = protocol.StExitInfo
type StLabelItemInfo = protocol.StLabelItemInfo
type StExitItemInfo = protocol.StExitItemInfo
type StParas = protocol.StParas
type StWeightBaseInfo = protocol.StWeightBaseInfo
type StResetAD = protocol.StResetAD
type StVolveTest = protocol.StVolveTest
type StOldWeightBaseInfo = protocol.StOldWeightBaseInfo
type StCameraParas = protocol.StCameraParas
type StWhiteBalanceMean = protocol.StWhiteBalanceMean
type StFruitCup = protocol.StFruitCup
type StIRCameraParas = protocol.StIRCameraParas
type StMotorInfo = protocol.StMotorInfo
type StStatistics = protocol.StStatistics
type SysStStatistics = protocol.SysStStatistics
type StFruitVisionParam = protocol.StFruitVisionParam
type StFruitUVParam = protocol.StFruitUVParam
type StNIRParam = protocol.StNIRParam
type StFruitParam = protocol.StFruitParam
type StFruitGradeInfo = protocol.StFruitGradeInfo
type StWeightStat = protocol.StWeightStat
type StTrackingData = protocol.StTrackingData
type StWeightResult = protocol.StWeightResult
type StWaveInfo = protocol.StWaveInfo
type Rect = protocol.Rect
type ColorRGB = protocol.ColorRGB
type QaulGradeInfo = protocol.QaulGradeInfo
type QaulityGradeItem = protocol.QaulityGradeItem
type QualityGradeInfo = protocol.QualityGradeInfo
type ExitState = protocol.ExitState
type TempExitState = protocol.TempExitState
type StColorList = protocol.StColorList
type StClientInfo = protocol.StClientInfo
type StClientNewInfo = protocol.StClientNewInfo
type StOldClientInfo = protocol.StOldClientInfo
type StFruitTypeMember = protocol.StFruitTypeMember
type StFruitType = protocol.StFruitType
type StBroadcastStatistics = protocol.StBroadcastStatistics
type StBroadcastSysConfig = protocol.StBroadcastSysConfig
type StExitAdditionalTextData = protocol.StExitAdditionalTextData
type StExitGradeItemInfo = protocol.StExitGradeItemInfo
type StExitGradeInfo = protocol.StExitGradeInfo
type StFruitGradeInfos = protocol.StFruitGradeInfos
type StExitInfos = protocol.StExitInfos
type LanguageType = protocol.LanguageType

const (
	cTCP48StSysConfigWireSize  = protocol.CTCP48StSysConfigWireSize
	cTCP48StGradeInfoWireSize  = protocol.CTCP48StGradeInfoWireSize
	cTCP48StParasWireSize      = protocol.CTCP48StParasWireSize
	cTCP48StMotorInfoWireSize  = protocol.CTCP48StMotorInfoWireSize
	cTCP48StGlobalExitInfoSize = protocol.CTCP48StGlobalExitInfoSize
	cTCP48StExitInfoWireSize   = protocol.CTCP48StExitInfoWireSize
	cTCP48StGlobalExpectedSize = protocol.CTCP48StGlobalExpectedSize
	cTCP48MaxChannelNum        = protocol.CTCP48MaxChannelNum
	cTCP48MaxIPMNum            = protocol.CTCP48MaxIPMNum
	cTCP48MaxExitNum           = protocol.CTCP48MaxExitNum
	cTCP48MaxLabelNum          = protocol.CTCP48MaxLabelNum
	cTCP48ChannelNumPerIPM     = protocol.CTCP48ChannelNumPerIPM
	cTCP48MaxColorCamera       = protocol.CTCP48MaxColorCamera
	cTCP48MaxNIRCamera         = protocol.CTCP48MaxNIRCamera
	cTCP48AnalogDensitySlots   = protocol.CTCP48AnalogDensitySlots
	cTCP48MaxCameraDelayInts   = protocol.CTCP48MaxCameraDelayInts

	cTCPServerMaxNoticeLength    = protocol.CTCPServerMaxNoticeLength
	cTCPServerMaxQualityGradeNum = protocol.CTCPServerMaxQualityGradeNum

	stStatisticsExpectedSize = protocol.StStatisticsExpectedSize

	MAX_EXIT_NUM    = protocol.MAX_EXIT_NUM
	MAX_TEXT_LENGTH = protocol.MAX_TEXT_LENGTH

	Chinese = protocol.Chinese
	English = protocol.English
	Spanish = protocol.Spanish
)

func StGlobalLayoutReport() string {
	return protocol.StGlobalLayoutReport()
}

func stStatisticsWireSize() int {
	return protocol.StStatisticsWireSize()
}

func stStatisticsSizeSummary() string {
	return protocol.StStatisticsSizeSummary()
}
