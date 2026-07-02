package protocol

const (
	CTCP48StSysConfigWireSize  = cTCP48StSysConfigWireSize
	CTCP48StGradeInfoWireSize  = cTCP48StGradeInfoWireSize
	CTCP48StParasWireSize      = cTCP48StParasWireSize
	CTCP48StMotorInfoWireSize  = cTCP48StMotorInfoWireSize
	CTCP48StGlobalExitInfoSize = cTCP48StGlobalExitInfoSize
	CTCP48StExitInfoWireSize   = cTCP48StExitInfoWireSize
	CTCP48StGlobalExpectedSize = cTCP48StGlobalExpectedSize
	CTCP48MaxChannelNum        = cTCP48MaxChannelNum
	CTCP48MaxIPMNum            = cTCP48MaxIPMNum
	CTCP48MaxExitNum           = cTCP48MaxExitNum
	CTCP48MaxLabelNum          = cTCP48MaxLabelNum
	CTCP48ChannelNumPerIPM     = cTCP48ChannelNumPerIPM
	CTCP48MaxColorCamera       = cTCP48MaxColorCamera
	CTCP48MaxNIRCamera         = cTCP48MaxNIRCamera
	CTCP48AnalogDensitySlots   = cTCP48AnalogDensitySlots
	CTCP48MaxCameraDelayInts   = cTCP48MaxCameraDelayInts

	CTCPServerMaxNoticeLength    = cTCPServerMaxNoticeLength
	CTCPServerMaxQualityGradeNum = cTCPServerMaxQualityGradeNum

	StStatisticsExpectedSize = stStatisticsExpectedSize
)

func StStatisticsWireSize() int {
	return stStatisticsWireSize()
}

func StStatisticsSizeSummary() string {
	return stStatisticsSizeSummary()
}
