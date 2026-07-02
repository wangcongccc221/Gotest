package tcp

import tcpclient "gotest/ohos/Tcp/client"

type CTcpClient = tcpclient.CTcpClient

const (
	cTCPHcID                = tcpclient.CTCPHcID
	cTCPSync                = tcpclient.CTCPSync
	cTCPDefaultFSMID        = tcpclient.CTCPDefaultFSMID
	cTCPDefaultConnectCount = tcpclient.CTCPDefaultConnectCount
	cTCPDefaultSendTime     = tcpclient.CTCPDefaultSendTime

	cTCPLCIPAddrTemplate = tcpclient.CTCPLCIPAddrTemplate
	cTCPACSIPAddr        = tcpclient.CTCPACSIPAddr
	cTCPSIMIPAddr        = tcpclient.CTCPSIMIPAddr

	cTCPFSMPort = tcpclient.CTCPFSMPort
	cTCPIPMPort = tcpclient.CTCPIPMPort
	cTCPSIMPort = tcpclient.CTCPSIMPort
	cTCPWAMPort = tcpclient.CTCPWAMPort
	cTCPACSPort = tcpclient.CTCPACSPort

	cTCPMaxSubsysNum  = tcpclient.CTCPMaxSubsysNum
	cTCPMaxChannelNum = tcpclient.CTCPMaxChannelNum
	cTCPMaxExitNum    = tcpclient.CTCPMaxExitNum

	cTCPHCDisplayOff       = tcpclient.CTCPHCDisplayOff
	cTCPHCClearData        = tcpclient.CTCPHCClearData
	cTCPHCSaveCurrentData  = tcpclient.CTCPHCSaveCurrentData
	cTCPHCTestCupOn        = tcpclient.CTCPHCTestCupOn
	cTCPHCTestCupOff       = tcpclient.CTCPHCTestCupOff
	cTCPHCFruitGradeOn     = tcpclient.CTCPHCFruitGradeOn
	cTCPHCFruitGradeOff    = tcpclient.CTCPHCFruitGradeOff
	cTCPHCMotorEnable      = tcpclient.CTCPHCMotorEnable
	cTCPHCSaveParas        = tcpclient.CTCPHCSaveParas
	cTCPHCDisplayOn        = tcpclient.CTCPHCDisplayOn
	cTCPHCExitClear        = tcpclient.CTCPHCExitClear
	cTCPHCSysConfig        = tcpclient.CTCPHCSysConfig
	cTCPHCGradeInfo        = tcpclient.CTCPHCGradeInfo
	cTCPHCExitInfo         = tcpclient.CTCPHCExitInfo
	cTCPHCParasInfo        = tcpclient.CTCPHCParasInfo
	cTCPHCTestVolve        = tcpclient.CTCPHCTestVolve
	cTCPHCTestAllLaneVolve = tcpclient.CTCPHCTestAllLaneVolve
	cTCPHCGlobalExitInfo   = tcpclient.CTCPHCGlobalExitInfo
	cTCPHCMotorInfo        = tcpclient.CTCPHCMotorInfo
	cTCPHCColorGradeInfo   = tcpclient.CTCPHCColorGradeInfo
	cTCPHCDensityInfo      = tcpclient.CTCPHCDensityInfo

	cTCPHCSingleSample        = tcpclient.CTCPHCSingleSample
	cTCPHCContinuousSampleOn  = tcpclient.CTCPHCContinuousSampleOn
	cTCPHCContinuousSampleOff = tcpclient.CTCPHCContinuousSampleOff
	cTCPHCShowBlobOn          = tcpclient.CTCPHCShowBlobOn
	cTCPHCAutoBalanceOnCamera = tcpclient.CTCPHCAutoBalanceOnCamera
	cTCPHCAutoBalanceOn       = tcpclient.CTCPHCAutoBalanceOn
	cTCPHCSingleSampleSpot    = tcpclient.CTCPHCSingleSampleSpot
	cTCPHCIpmShutdown         = tcpclient.CTCPHCIpmShutdown
	cTCPHCShutterAdjustOn     = tcpclient.CTCPHCShutterAdjustOn
	cTCPHCShutterAdjustOff    = tcpclient.CTCPHCShutterAdjustOff

	cTCPHCWAMWeightOn          = tcpclient.CTCPHCWAMWeightOn
	cTCPHCWAMWeightReset       = tcpclient.CTCPHCWAMWeightReset
	cTCPHCWAMCupStateReset     = tcpclient.CTCPHCWAMCupStateReset
	cTCPHCWAMSimulatedPulseOn  = tcpclient.CTCPHCWAMSimulatedPulseOn
	cTCPHCWAMSimulatedPulseOff = tcpclient.CTCPHCWAMSimulatedPulseOff
	cTCPHCWAMTestCupOn         = tcpclient.CTCPHCWAMTestCupOn
	cTCPHCWAMTestCupOff        = tcpclient.CTCPHCWAMTestCupOff
	cTCPHCWAMWaveFormOn        = tcpclient.CTCPHCWAMWaveFormOn
	cTCPHCWAMWaveFormOff       = tcpclient.CTCPHCWAMWaveFormOff
	cTCPHCWAMDataTrackingOn    = tcpclient.CTCPHCWAMDataTrackingOn
	cTCPHCWAMDataTrackingOff   = tcpclient.CTCPHCWAMDataTrackingOff
	cTCPHCWAMResetAD           = tcpclient.CTCPHCWAMResetAD
	cTCPHCWAMGlobalWeightInfo  = tcpclient.CTCPHCWAMGlobalWeightInfo
	cTCPHCWAMWeightInfo        = tcpclient.CTCPHCWAMWeightInfo
	cTCPHCWAMGetWAMInfo        = tcpclient.CTCPHCWAMGetWAMInfo
)

func NewCTcpClient() *CTcpClient {
	return tcpclient.NewCTcpClient()
}

func StartCTCPClient(remoteIP string, remotePort int, destID int32, cmd int32, data []byte) int {
	return tcpclient.StartCTCPClient(remoteIP, remotePort, destID, cmd, data)
}

func RequestStGlobalFromDefaultFSM() int {
	return tcpclient.RequestStGlobalFromDefaultFSM()
}

func RequestStGlobalFromFSM(destID int32) int {
	return tcpclient.RequestStGlobalFromFSM(destID)
}

func resolveCTCPTarget(destID int32, cmd int32, ip string, port int) (string, int) {
	return tcpclient.ResolveCTCPTarget(destID, cmd, ip, port)
}

func isMAX2WAMCommand(cmd int32) bool {
	return tcpclient.IsMAX2WAMCommand(cmd)
}

func getSubsysIndex(id int32) int {
	return tcpclient.GetSubsysIndex(id)
}

func getSubSysID(id int32) int {
	return tcpclient.GetSubSysID(id)
}

func getIPMID(id int32) int {
	return tcpclient.GetIPMID(id)
}

func getWAMIP(id int32) int {
	return tcpclient.GetWAMIP(id)
}

func encodeChannel(x int, y int, z int) int32 {
	return tcpclient.EncodeChannel(x, y, z)
}

func encodeSubsys(x int) int32 {
	return tcpclient.EncodeSubsys(x)
}

func encodeWAMID(subsysIndex int) int32 {
	return tcpclient.EncodeWAMID(subsysIndex)
}

func getAllSysID(channelInfo []uint8) []int32 {
	return tcpclient.GetAllSysID(channelInfo)
}

func channelCountAt(channelInfo []uint8, index int) int {
	return tcpclient.ChannelCountAt(channelInfo, index)
}

func sysConfigChannelCount(data []byte, index int) int {
	return tcpclient.SysConfigChannelCount(data, index)
}

func ipLastByte(ip string) int {
	return tcpclient.IPLastByte(ip)
}
