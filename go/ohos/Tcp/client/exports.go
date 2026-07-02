package client

const (
	CTCPHcID                = cTCPHcID
	CTCPSync                = cTCPSync
	CTCPDefaultFSMID        = cTCPDefaultFSMID
	CTCPDefaultConnectCount = cTCPDefaultConnectCount
	CTCPDefaultSendTime     = cTCPDefaultSendTime

	CTCPLCIPAddrTemplate = cTCPLCIPAddrTemplate
	CTCPACSIPAddr        = cTCPACSIPAddr
	CTCPSIMIPAddr        = cTCPSIMIPAddr

	CTCPFSMPort = cTCPFSMPort
	CTCPIPMPort = cTCPIPMPort
	CTCPSIMPort = cTCPSIMPort
	CTCPWAMPort = cTCPWAMPort
	CTCPACSPort = cTCPACSPort

	CTCPMaxSubsysNum  = cTCPMaxSubsysNum
	CTCPMaxChannelNum = cTCPMaxChannelNum
	CTCPMaxExitNum    = cTCPMaxExitNum

	CTCPHCDisplayOff       = cTCPHCDisplayOff
	CTCPHCClearData        = cTCPHCClearData
	CTCPHCSaveCurrentData  = cTCPHCSaveCurrentData
	CTCPHCTestCupOn        = cTCPHCTestCupOn
	CTCPHCTestCupOff       = cTCPHCTestCupOff
	CTCPHCFruitGradeOn     = cTCPHCFruitGradeOn
	CTCPHCFruitGradeOff    = cTCPHCFruitGradeOff
	CTCPHCMotorEnable      = cTCPHCMotorEnable
	CTCPHCSaveParas        = cTCPHCSaveParas
	CTCPHCDisplayOn        = cTCPHCDisplayOn
	CTCPHCExitClear        = cTCPHCExitClear
	CTCPHCSysConfig        = cTCPHCSysConfig
	CTCPHCGradeInfo        = cTCPHCGradeInfo
	CTCPHCExitInfo         = cTCPHCExitInfo
	CTCPHCParasInfo        = cTCPHCParasInfo
	CTCPHCTestVolve        = cTCPHCTestVolve
	CTCPHCTestAllLaneVolve = cTCPHCTestAllLaneVolve
	CTCPHCGlobalExitInfo   = cTCPHCGlobalExitInfo
	CTCPHCMotorInfo        = cTCPHCMotorInfo
	CTCPHCColorGradeInfo   = cTCPHCColorGradeInfo
	CTCPHCDensityInfo      = cTCPHCDensityInfo

	CTCPHCSingleSample        = cTCPHCSingleSample
	CTCPHCContinuousSampleOn  = cTCPHCContinuousSampleOn
	CTCPHCContinuousSampleOff = cTCPHCContinuousSampleOff
	CTCPHCShowBlobOn          = cTCPHCShowBlobOn
	CTCPHCAutoBalanceOnCamera = cTCPHCAutoBalanceOnCamera
	CTCPHCAutoBalanceOn       = cTCPHCAutoBalanceOn
	CTCPHCSingleSampleSpot    = cTCPHCSingleSampleSpot
	CTCPHCIpmShutdown         = cTCPHCIpmShutdown
	CTCPHCShutterAdjustOn     = cTCPHCShutterAdjustOn
	CTCPHCShutterAdjustOff    = cTCPHCShutterAdjustOff

	CTCPHCWAMWeightOn          = cTCPHCWAMWeightOn
	CTCPHCWAMWeightReset       = cTCPHCWAMWeightReset
	CTCPHCWAMCupStateReset     = cTCPHCWAMCupStateReset
	CTCPHCWAMSimulatedPulseOn  = cTCPHCWAMSimulatedPulseOn
	CTCPHCWAMSimulatedPulseOff = cTCPHCWAMSimulatedPulseOff
	CTCPHCWAMTestCupOn         = cTCPHCWAMTestCupOn
	CTCPHCWAMTestCupOff        = cTCPHCWAMTestCupOff
	CTCPHCWAMWaveFormOn        = cTCPHCWAMWaveFormOn
	CTCPHCWAMWaveFormOff       = cTCPHCWAMWaveFormOff
	CTCPHCWAMDataTrackingOn    = cTCPHCWAMDataTrackingOn
	CTCPHCWAMDataTrackingOff   = cTCPHCWAMDataTrackingOff
	CTCPHCWAMResetAD           = cTCPHCWAMResetAD
	CTCPHCWAMGlobalWeightInfo  = cTCPHCWAMGlobalWeightInfo
	CTCPHCWAMWeightInfo        = cTCPHCWAMWeightInfo
	CTCPHCWAMGetWAMInfo        = cTCPHCWAMGetWAMInfo
)

func ResolveCTCPTarget(destID int32, cmd int32, ip string, port int) (string, int) {
	return resolveCTCPTarget(destID, cmd, ip, port)
}

func IsMAX2WAMCommand(cmd int32) bool {
	return isMAX2WAMCommand(cmd)
}

func GetSubsysIndex(id int32) int {
	return getSubsysIndex(id)
}

func GetSubSysID(id int32) int {
	return getSubSysID(id)
}

func GetIPMID(id int32) int {
	return getIPMID(id)
}

func GetWAMIP(id int32) int {
	return getWAMIP(id)
}

func EncodeChannel(x int, y int, z int) int32 {
	return encodeChannel(x, y, z)
}

func EncodeSubsys(x int) int32 {
	return encodeSubsys(x)
}

func EncodeWAMID(subsysIndex int) int32 {
	return encodeWAMID(subsysIndex)
}

func GetAllSysID(channelInfo []uint8) []int32 {
	return getAllSysID(channelInfo)
}

func ChannelCountAt(channelInfo []uint8, index int) int {
	return channelCountAt(channelInfo, index)
}

func SysConfigChannelCount(data []byte, index int) int {
	return sysConfigChannelCount(data, index)
}

func IPLastByte(ip string) int {
	return ipLastByte(ip)
}
