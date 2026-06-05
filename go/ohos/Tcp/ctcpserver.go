package tcp

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"
)

const (
	cTCPServerListenIP  = "0.0.0.0"
	cTCPServerHCIP      = "192.168.0.15"
	cTCPServerImagePort = 1127
	cTCPServerStatPort  = 1128

	cTCPServerMaxPayload       = 256 * 1024 * 1024
	cTCPServerReadTimeout      = 5 * time.Second
	cTCPServerIdleReadTimeout  = 500 * time.Millisecond
	cTCPServerAcceptRetryDelay = 50 * time.Millisecond

	cTCPServerMaxSubsysNum       = 4
	cTCPServerMaxCameraNum       = 9
	cTCPServerStSysConfigExit48  = 48
	cTCPServerStSysConfigExit64  = 64
	cTCPServerStSysConfigMaxChan = 12

	cTCPServerMaxColorIntervalNum = 3
	cTCPServerMaxColorGradeNum    = 16

	cTCPServerMaxSizeGradeNum    = 16
	cTCPServerMaxTextLength      = 12
	cTCPServerMaxFruitNameLength = 50
	cTCPServerParasTagInfoNum    = 6
	cTCPServerMinorGradeNum      = 6

	cTCPStStatisticsSpeedPublishInterval = time.Second
	cTCPStStatisticsSpeedStaleInterval   = 2500 * time.Millisecond
)

const (
	cmdFSMConfig            = int32(0x1000)
	cmdFSMStatistics        = int32(0x1001)
	cmdFSMGradeInfo         = int32(0x1002)
	cmdFSMWeightInfo        = int32(0x1003)
	cmdFSMWaveInfo          = int32(0x1004)
	cmdFSMVersionError      = int32(0x1005)
	cmdFSMBurnFlashProgress = int32(0x1006)
	cmdFSMBurnDebug         = int32(0x1007)
	cmdFSMGetVersion        = int32(0x1008)
	cmdFSMBootFlashProgress = int32(0x1009)

	cmdWAMWeightGlobal = int32(0x0120)
	cmdWAMWeightInfo   = int32(0x0121)
	cmdWAMWaveInfo     = int32(0x0122)
	cmdWAMVersionInfo  = int32(0x0123)

	cmdIPMImage                   = int32(0x3000)
	cmdIPMAutoBalanceCoefficient  = int32(0x3001)
	cmdIPMImageSplice             = int32(0x3002)
	cmdIPMImageSpot               = int32(0x3003)
	cmdIPMShutterAdjust           = int32(0x3004)
	cmdACSExitStop                = int32(0x8000)
	cmdSIMDisplayOn               = int32(0x7000)
	cmdSIMInspectionOn            = int32(0x7001)
	cmdSIMInspectionBadNumber     = int32(0x7002)
	cmdSIMInspectionOff           = int32(0x7003)
	cmdSIMInspectionConveyorStart = int32(0x7004)
	cmdSIMInspectionConveyorClose = int32(0x7005)
)

type cTCPServerCommandHead struct {
	NSrcId int32
	NDstId int32
	NCmdId int32
}

type cTCPStoredPayload struct {
	Head       cTCPServerCommandHead
	Payload    []byte
	Port       int
	ServerName string
	RemoteAddr string
	ReceivedAt time.Time
}

type cTCPStStatisticsSpeedState struct {
	Latest    StStatistics
	LatestAt  time.Time
	HasLatest bool
}

type cTCPStParasImageFieldsSnapshot struct {
	RemoteAddr string
	SrcID      int32
	DstID      int32
	SubsysID   int32
	ReceivedAt time.Time
	Paras      [12]StParas
}

type cTCPStGlobalExitInfoSnapshot struct {
	RemoteAddr string
	SrcID      int32
	DstID      int32
	SubsysID   int32
	ReceivedAt time.Time
	GExit      StGlobalExitInfo
}

type cTCPServer struct {
	name     string
	port     int
	listener net.Listener
	stop     chan struct{}
	wg       sync.WaitGroup
}

var (
	cTCPServerMu          sync.Mutex
	cTCPServerImage       *cTCPServer
	cTCPServerStat        *cTCPServer
	cTCPServerLastMu      sync.Mutex
	cTCPServerLastMessage = "CTCP server not started"
	cTCPServerMessages    []string

	cTCPPayloadMu    sync.Mutex
	cTCPPayloadByCmd = make(map[string]cTCPStoredPayload)

	cTCPStParasImageFieldsMu        sync.Mutex
	cTCPStParasImageFieldsLatest    cTCPStParasImageFieldsSnapshot
	cTCPStParasImageFieldsHasLatest bool
	cTCPStParasImageFieldsStop      chan struct{}

	cTCPStGlobalExitInfoMu        sync.Mutex
	cTCPStGlobalExitInfoLatest    cTCPStGlobalExitInfoSnapshot
	cTCPStGlobalExitInfoHasLatest bool
	cTCPStGlobalExitInfoStop      chan struct{}

	cTCPStStatisticsSpeedMu    sync.Mutex
	cTCPStStatisticsSpeedBySys = make(map[int32]*cTCPStStatisticsSpeedState)
	cTCPStStatisticsSpeedStop  chan struct{}
)

func StartCTCPServer() int {
	cTCPServerMu.Lock()
	defer cTCPServerMu.Unlock()

	if cTCPServerImage != nil && cTCPServerStat != nil {
		setCTCPServerLastMessage("CTCP servers already listening")
		return cTCPServerStatPort
	}

	imageServer, err := newCTCPServer("image", cTCPServerImagePort)
	if err != nil {
		setCTCPServerLastMessage("CTCP image server listen failed: %v", err)
		return -1
	}

	statServer, err := newCTCPServer("stat", cTCPServerStatPort)
	if err != nil {
		imageServer.stopServer()
		setCTCPServerLastMessage("CTCP stat server listen failed: %v", err)
		return -1
	}

	cTCPServerImage = imageServer
	cTCPServerStat = statServer
	imageServer.start()
	statServer.start()

	setCTCPServerLastMessage("CTCP servers listening on %s:%d and %s:%d, HC_IP=%s, HC_ID=0x%04X",
		cTCPServerListenIP,
		cTCPServerImagePort,
		cTCPServerListenIP,
		cTCPServerStatPort,
		cTCPServerHCIP,
		uint32(cTCPHcID))

	goSz := int(unsafe.Sizeof(StGlobal{}))
	setCTCPServerLastMessage("CTCP startup: sizeof(StGlobal)=%d 字节, 线宽常量 cTCP48StGlobalExpectedSize=%d",
		goSz, cTCP48StGlobalExpectedSize)
	LoadStExitInfosFromLocalConfig()
	LoadExitDisplayInfoFromLocalConfig()
	LoadExitAdditionalTextInfoFromLocalConfig()
	LoadLevelAuxConfigInfoFromLocalConfig()
	StartStGlobalExitInfoPeriodicLog()
	StartStStatisticsSpeedPublisher()
	return cTCPServerStatPort
}

// appendCTCPLogChunks 将长文本切成多段写入 CTCP 日志环形缓冲，避免单条 HiLog 过长被截断。
func appendCTCPLogChunks(tag string, content string) {
	const maxChunk = 3800
	n := len(content)
	if n == 0 {
		setCTCPServerLastMessage("%s: <empty>", tag)
		return
	}
	total := (n + maxChunk - 1) / maxChunk
	for part, at := 1, 0; at < n; part++ {
		end := at + maxChunk
		if end > n {
			end = n
		}
		setCTCPServerLastMessage("%s part %d/%d chars [%d:%d)\n%s", tag, part, total, at, end, content[at:end])
		at = end
	}
}

// appendPayloadHexChunks 将原始 payload 以十六进制输出，不带 offset，方便直接看下位机发来的字节内容。
func appendPayloadHexChunks(tag string, payload []byte) {
	if len(payload) == 0 {
		setCTCPServerLastMessage("%s 原始字节HEX: <empty payload>", tag)
		return
	}
	var b strings.Builder
	for i := 0; i < len(payload); i++ {
		if i > 0 {
			if i%32 == 0 {
				b.WriteByte('\n')
			} else {
				b.WriteByte(' ')
			}
		}
		fmt.Fprintf(&b, "%02X", payload[i])
	}
	appendCTCPLogChunks(tag+" 原始字节HEX", b.String())
}

const cTCPFruitGradeInvalidValue = uint32(0x7f7f7f7f)

func fruitParamHasAnyData(param StFruitParam) bool {
	if param.UnGrade == cTCPFruitGradeInvalidValue {
		return false
	}
	return param.FWeight != 0 ||
		param.FDensity != 0 ||
		param.UnGrade != 0 ||
		param.UnWhichExit != 0 ||
		param.VisionParam.UnColorRate0 != 0 ||
		param.VisionParam.UnColorRate1 != 0 ||
		param.VisionParam.UnColorRate2 != 0 ||
		param.VisionParam.UnArea != 0 ||
		param.VisionParam.UnFlawArea != 0 ||
		param.VisionParam.UnVolume != 0 ||
		param.VisionParam.UnFlawNum != 0 ||
		param.VisionParam.UnMaxR != 0 ||
		param.VisionParam.UnMinR != 0 ||
		param.VisionParam.UnSelectBasis != 0 ||
		param.VisionParam.FDiameterRatio != 0 ||
		param.VisionParam.FMinDRatio != 0 ||
		param.UVParam.UnBruiseArea != 0 ||
		param.UVParam.UnBruiseNum != 0 ||
		param.UVParam.UnRotArea != 0 ||
		param.UVParam.UnRotNumy != 0 ||
		param.UVParam.UnRigidity != 0 ||
		param.UVParam.UnWater != 0 ||
		param.NIRParam.FSugar != 0 ||
		param.NIRParam.FAcidity != 0 ||
		param.NIRParam.FHollow != 0 ||
		param.NIRParam.FSkin != 0 ||
		param.NIRParam.FBrown != 0 ||
		param.NIRParam.FTangxin != 0
}

type cTCPFruitGradeInfoRealtimeFrame struct {
	SrcID          int32            `json:"srcId"`
	FruitGradeInfo StFruitGradeInfo `json:"fruitGradeInfo"`
}

func cTCPGradeInfoIPMIndex(srcID int32) int {
	ipmIndex := int((srcID>>4)&0x0F) - 1
	if ipmIndex < 0 {
		return 0
	}
	return ipmIndex
}

func cTCPFruitGradeInfoSourceID(head cTCPServerCommandHead, grade StFruitGradeInfo) int32 {
	if grade.NRouteId > 0 {
		return grade.NRouteId
	}
	return head.NSrcId
}

func sanitizeFruitGradeInfoForRealtime(grade StFruitGradeInfo) StFruitGradeInfo {
	for index := range grade.Param {
		if grade.Param[index].UnGrade == cTCPFruitGradeInvalidValue {
			grade.Param[index] = StFruitParam{}
		}
	}
	return grade
}

func appendFruitGradeInfoLog(remoteAddr string, head cTCPServerCommandHead, grade StFruitGradeInfo, payloadBytes int) {
	var b strings.Builder
	sourceID := cTCPFruitGradeInfoSourceID(head, grade)
	ipmIndex := cTCPGradeInfoIPMIndex(sourceID)
	fmt.Fprintf(
		&b,
		"CTCP StFruitGradeInfo 回推: remote=%s, src=0x%04X, route=0x%04X, dst=0x%04X, cmd=%s, payload=%d bytes",
		remoteAddr,
		uint32(head.NSrcId),
		uint32(sourceID),
		uint32(head.NDstId),
		cTCPCommandName(head.NCmdId),
		payloadBytes,
	)
	count := 0
	for laneIndex := 0; laneIndex < len(grade.Param); laneIndex++ {
		param := grade.Param[laneIndex]
		if !fruitParamHasAnyData(param) {
			continue
		}
		channelNo := ipmIndex*len(grade.Param) + laneIndex + 1
		gradeText := fmt.Sprintf("%d", param.UnGrade)
		if param.UnGrade == cTCPFruitGradeInvalidValue {
			gradeText = "INVALID(0x7F7F7F7F)"
		}
		fmt.Fprintf(
			&b,
			"\n  detectChannel=%d ipm=%d lane=%d route=%d data{Weight=%.3f, Density=%.3f, Grade=%s, Outlet=%d, Diameter=%.3f, MaxR=%.3f, MinR=%.3f, Area=%d, Volume=%d, Color=[%d,%d,%d], FlawArea=%d, FlawNum=%d, BruiseArea=%d, BruiseNum=%d, RotArea=%d, RotNum=%d, Sugar=%.3f, Acidity=%.3f, Hollow=%.3f, Skin=%.3f, Brown=%.3f, Tangxin=%.3f, Rigidity=%d, Water=%d}",
			channelNo,
			ipmIndex,
			laneIndex,
			grade.NRouteId,
			param.FWeight,
			param.FDensity,
			gradeText,
			param.UnWhichExit,
			param.VisionParam.UnSelectBasis,
			param.VisionParam.UnMaxR,
			param.VisionParam.UnMinR,
			param.VisionParam.UnArea,
			param.VisionParam.UnVolume,
			param.VisionParam.UnColorRate0,
			param.VisionParam.UnColorRate1,
			param.VisionParam.UnColorRate2,
			param.VisionParam.UnFlawArea,
			param.VisionParam.UnFlawNum,
			param.UVParam.UnBruiseArea,
			param.UVParam.UnBruiseNum,
			param.UVParam.UnRotArea,
			param.UVParam.UnRotNumy,
			param.NIRParam.FSugar,
			param.NIRParam.FAcidity,
			param.NIRParam.FHollow,
			param.NIRParam.FSkin,
			param.NIRParam.FBrown,
			param.NIRParam.FTangxin,
			param.UVParam.UnRigidity,
			param.UVParam.UnWater,
		)
		count++
	}
	if count == 0 {
		b.WriteString("\n  <no non-empty channel data>")
	}
	appendCTCPLogChunks("CTCP StFruitGradeInfo 回推", b.String())
}

func cacheStParasImageFields(remoteAddr string, head cTCPServerCommandHead, stg StGlobal) {
	cTCPStParasImageFieldsMu.Lock()
	cTCPStParasImageFieldsLatest = cTCPStParasImageFieldsSnapshot{
		RemoteAddr: remoteAddr,
		SrcID:      head.NSrcId,
		DstID:      head.NDstId,
		SubsysID:   stg.NSubsysId,
		ReceivedAt: time.Now(),
		Paras:      stg.Paras,
	}
	cTCPStParasImageFieldsHasLatest = true
	cTCPStParasImageFieldsMu.Unlock()
}

func latestStParasImageFieldsSnapshot() (cTCPStParasImageFieldsSnapshot, bool) {
	cTCPStParasImageFieldsMu.Lock()
	defer cTCPStParasImageFieldsMu.Unlock()
	return cTCPStParasImageFieldsLatest, cTCPStParasImageFieldsHasLatest
}

func cacheStGlobalExitInfo(remoteAddr string, head cTCPServerCommandHead, stg StGlobal) {
	cTCPStGlobalExitInfoMu.Lock()
	cTCPStGlobalExitInfoLatest = cTCPStGlobalExitInfoSnapshot{
		RemoteAddr: remoteAddr,
		SrcID:      head.NSrcId,
		DstID:      head.NDstId,
		SubsysID:   stg.NSubsysId,
		ReceivedAt: time.Now(),
		GExit:      stg.GExit,
	}
	cTCPStGlobalExitInfoHasLatest = true
	cTCPStGlobalExitInfoMu.Unlock()
}

func latestStGlobalExitInfoSnapshot() (cTCPStGlobalExitInfoSnapshot, bool) {
	cTCPStGlobalExitInfoMu.Lock()
	defer cTCPStGlobalExitInfoMu.Unlock()
	return cTCPStGlobalExitInfoLatest, cTCPStGlobalExitInfoHasLatest
}

func formatStFruitCupImageFields(index int, cup StFruitCup) string {
	return fmt.Sprintf(
		"cup%d{NLeft=%v,NTop=%d,NBottom=%d,NOffsetX=%d,NOffsetY=%d}",
		index,
		cup.NLeft,
		cup.NTop,
		cup.NBottom,
		cup.NOffsetX,
		cup.NOffsetY,
	)
}

func formatStColorCameraParasFields(index int, camera StCameraParas) string {
	return fmt.Sprintf(
		"Color%d{MeanValue={R=%d,G=%d,B=%d},NShutter=%d,FGammaCorrection=%.3f,NTriggerDelay=%d,NDetectionThreshold=%v,NDetectWhiteTh=%v,FPixelRatio=%v,FFruitCupRangeTh=%v,NXYEdgeBreakTh=%v,NROIOffsetY=%v,CCameraNum=%d,%s,%s}",
		index,
		camera.MeanValue.MeanR,
		camera.MeanValue.MeanG,
		camera.MeanValue.MeanB,
		camera.NShutter,
		camera.FGammaCorrection,
		camera.NTriggerDelay,
		camera.NDetectionThreshold,
		camera.NDetectWhiteTh,
		camera.FPixelRatio,
		camera.FFruitCupRangeTh,
		camera.NXYEdgeBreakTh,
		camera.NROIOffsetY,
		camera.CCameraNum,
		formatStFruitCupImageFields(0, camera.Cup[0]),
		formatStFruitCupImageFields(1, camera.Cup[1]),
	)
}

func formatStIRCameraParasFields(index int, camera StIRCameraParas) string {
	return fmt.Sprintf(
		"NIR%d{NShutter=%d,FGammaCorrection=%.3f,NTriggerDelay=%d,NIRDetectionThreshold=%v,FPixelRatio=%v,FFruitCupRangeTh=%v,NXYEdgeBreakTh=%v,NROIOffsetY=%v,CCameraNum=%d,%s,%s}",
		index,
		camera.NShutter,
		camera.FGammaCorrection,
		camera.NTriggerDelay,
		camera.NIRDetectionThreshold,
		camera.FPixelRatio,
		camera.FFruitCupRangeTh,
		camera.NXYEdgeBreakTh,
		camera.NROIOffsetY,
		camera.CCameraNum,
		formatStFruitCupImageFields(0, camera.Cup[0]),
		formatStFruitCupImageFields(1, camera.Cup[1]),
	)
}

func formatStParasImageFieldsLine(ipmIndex int, paras StParas) string {
	parts := []string{fmt.Sprintf("ipm%02d NCupNum=%d", ipmIndex+1, paras.NCupNum)}
	for cameraIndex, camera := range paras.CameraParas {
		parts = append(parts, formatStColorCameraParasFields(cameraIndex, camera))
	}
	for cameraIndex, camera := range paras.IRCameraParas {
		parts = append(parts, formatStIRCameraParasFields(cameraIndex, camera))
	}
	return strings.Join(parts, " | ")
}

func LogLatestStParasImageFields() {
}

func StartStParasImageFieldsPeriodicLog() {
	StartStParasImageFieldsPeriodicLogWithInterval(10 * time.Second)
}

func StartStParasImageFieldsPeriodicLogWithInterval(interval time.Duration) {
	if interval <= 0 {
		interval = 10 * time.Second
	}

	cTCPStParasImageFieldsMu.Lock()
	if cTCPStParasImageFieldsStop != nil {
		cTCPStParasImageFieldsMu.Unlock()
		return
	}
	stop := make(chan struct{})
	cTCPStParasImageFieldsStop = stop
	cTCPStParasImageFieldsMu.Unlock()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		setCTCPServerLastMessage("StParas 图像区域/偏移周期输出已启动: 每 %s 打印一次", interval)
		for {
			select {
			case <-stop:
				setCTCPServerLastMessage("StParas 图像区域/偏移周期输出已停止")
				return
			case <-ticker.C:
				LogLatestStParasImageFields()
			}
		}
	}()
}

func StopStParasImageFieldsPeriodicLog() {
	cTCPStParasImageFieldsMu.Lock()
	stop := cTCPStParasImageFieldsStop
	cTCPStParasImageFieldsStop = nil
	cTCPStParasImageFieldsLatest = cTCPStParasImageFieldsSnapshot{}
	cTCPStParasImageFieldsHasLatest = false
	cTCPStParasImageFieldsMu.Unlock()
	if stop != nil {
		close(stop)
	}
}

func formatStGlobalExitInfoFields(info StGlobalExitInfo) string {
	return fmt.Sprintf(
		"NPulse=%d, VersionFlag=%d, NLabelPulse=%d, NDriverPin=%v, DelayTime=%v, HoldTime=%v",
		info.NPulse,
		info.VersionFlag,
		info.NLabelPulse,
		info.NDriverPin,
		info.DelayTime,
		info.HoldTime,
	)
}

func LogLatestStGlobalExitInfo() {
	size := int(unsafe.Sizeof(StGlobalExitInfo{}))
	snapshot, ok := latestStGlobalExitInfoSnapshot()
	if !ok {
		setCTCPServerLastMessage(
			"CTCP StGlobalExitInfo 全字段 5秒打印: sizeof(StGlobalExitInfo)=%d bytes, expected=%d bytes, 尚无缓存数据（等待 FSM_CMD_CONFIG/StGlobal）",
			size,
			cTCP48StGlobalExitInfoSize,
		)
		return
	}
	setCTCPServerLastMessage(
		"CTCP StGlobalExitInfo 全字段 5秒打印: remote=%s, src=0x%04X, dst=0x%04X, nSubsysId=%d, cachedAt=%s, sizeof(StGlobalExitInfo)=%d bytes, expected=%d bytes, %s",
		snapshot.RemoteAddr,
		uint32(snapshot.SrcID),
		uint32(snapshot.DstID),
		snapshot.SubsysID,
		snapshot.ReceivedAt.Format("15:04:05.000"),
		size,
		cTCP48StGlobalExitInfoSize,
		formatStGlobalExitInfoFields(snapshot.GExit),
	)
}

func StartStGlobalExitInfoPeriodicLog() {
	StartStGlobalExitInfoPeriodicLogWithInterval(5 * time.Second)
}

func StartStGlobalExitInfoPeriodicLogWithInterval(interval time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Second
	}

	cTCPStGlobalExitInfoMu.Lock()
	if cTCPStGlobalExitInfoStop != nil {
		cTCPStGlobalExitInfoMu.Unlock()
		return
	}
	stop := make(chan struct{})
	cTCPStGlobalExitInfoStop = stop
	cTCPStGlobalExitInfoMu.Unlock()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		setCTCPServerLastMessage("StGlobalExitInfo 全字段周期输出已启动: 每 %s 打印一次", interval)
		LogLatestStGlobalExitInfo()
		for {
			select {
			case <-stop:
				setCTCPServerLastMessage("StGlobalExitInfo 全字段周期输出已停止")
				return
			case <-ticker.C:
				LogLatestStGlobalExitInfo()
			}
		}
	}()
}

func StopStGlobalExitInfoPeriodicLog() {
	cTCPStGlobalExitInfoMu.Lock()
	stop := cTCPStGlobalExitInfoStop
	cTCPStGlobalExitInfoStop = nil
	cTCPStGlobalExitInfoLatest = cTCPStGlobalExitInfoSnapshot{}
	cTCPStGlobalExitInfoHasLatest = false
	cTCPStGlobalExitInfoMu.Unlock()
	if stop != nil {
		close(stop)
	}
}

func StartStStatisticsSpeedPublisher() {
	StartStStatisticsSpeedPublisherWithInterval(cTCPStStatisticsSpeedPublishInterval)
}

func StartStStatisticsSpeedPublisherWithInterval(interval time.Duration) {
	if interval <= 0 {
		interval = cTCPStStatisticsSpeedPublishInterval
	}

	cTCPStStatisticsSpeedMu.Lock()
	if cTCPStStatisticsSpeedStop != nil {
		cTCPStStatisticsSpeedMu.Unlock()
		return
	}
	stop := make(chan struct{})
	cTCPStStatisticsSpeedStop = stop
	cTCPStStatisticsSpeedMu.Unlock()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		setCTCPServerLastMessage("CTCP StStatistics 分选速度后端计算已启动: 每 %s 推送一次", interval)
		for {
			select {
			case <-stop:
				setCTCPServerLastMessage("CTCP StStatistics 分选速度后端计算已停止")
				return
			case now := <-ticker.C:
				publishLatestStStatisticsSpeed(now)
			}
		}
	}()
}

func StopStStatisticsSpeedPublisher() {
	cTCPStStatisticsSpeedMu.Lock()
	stop := cTCPStStatisticsSpeedStop
	cTCPStStatisticsSpeedStop = nil
	cTCPStStatisticsSpeedBySys = make(map[int32]*cTCPStStatisticsSpeedState)
	cTCPStStatisticsSpeedMu.Unlock()
	if stop != nil {
		close(stop)
	}
}

func sanitizeStStatisticsSpeed(speed int32) int32 {
	if speed <= 0 {
		return 0
	}
	return speed
}

func calculateStStatisticsSpeedFromPulse(state StStatistics) (int32, bool) {
	if state.NInterval <= 0 {
		return 0, true
	}
	if state.NPulseInterval > 2000 {
		return 0, true
	}
	if state.NPulseInterval > 0 {
		speed := math.Round(60000.0 / float64(state.NPulseInterval))
		if speed <= 0 {
			return 0, true
		}
		if speed > 2147483647 {
			return 2147483647, true
		}
		return int32(speed), true
	}
	return 0, false
}

func resolveStStatisticsDisplaySpeed(state StStatistics) int32 {
	if speed, ok := calculateStStatisticsSpeedFromPulse(state); ok {
		return speed
	}
	return sanitizeStStatisticsSpeed(state.NIntervalSumperminute)
}

func cacheStStatisticsForSpeed(state StStatistics, receivedAt time.Time) {
	subsysID := state.NSubsysId
	if subsysID <= 0 {
		subsysID = 1
	}

	cTCPStStatisticsSpeedMu.Lock()
	entry := cTCPStStatisticsSpeedBySys[subsysID]
	if entry == nil {
		entry = &cTCPStStatisticsSpeedState{}
		cTCPStStatisticsSpeedBySys[subsysID] = entry
	}
	state.NIntervalSumperminute = resolveStStatisticsDisplaySpeed(state)
	entry.Latest = state
	entry.LatestAt = receivedAt
	entry.HasLatest = true
	cTCPStStatisticsSpeedMu.Unlock()
	maybeSaveRealtimeStatistics(receivedAt)
}

func latestStStatisticsSpeedSnapshots(now time.Time) []StStatistics {
	cTCPStStatisticsSpeedMu.Lock()
	defer cTCPStStatisticsSpeedMu.Unlock()

	snapshots := make([]StStatistics, 0, len(cTCPStStatisticsSpeedBySys))
	for _, entry := range cTCPStStatisticsSpeedBySys {
		if entry == nil || !entry.HasLatest {
			continue
		}
		latest := entry.Latest
		if now.Sub(entry.LatestAt) > cTCPStStatisticsSpeedStaleInterval {
			latest.NIntervalSumperminute = 0
		}
		snapshots = append(snapshots, latest)
	}
	return snapshots
}

func publishLatestStStatisticsSpeed(now time.Time) {
	snapshots := latestStStatisticsSpeedSnapshots(now)
	for _, state := range snapshots {
		stateJSON, jsonErr := FormatDataFullJSON(state)
		if stateJSON == "" || jsonErr != nil {
			setCTCPServerLastMessage("CTCP StStatistics 后端分选速度 JSON 生成失败: %v", jsonErr)
			continue
		}
		if err := PublishWebSocketJSON(webSocketTopicStatistics, stateJSON); err != nil {
			setCTCPServerLastMessage("CTCP StStatistics 后端分选速度 WebSocket 推送失败: %v", err)
		}
	}
	publishLatestHomeStats(now)
}

func normalizeStStatisticsSubsysID(srcID int32, payloadSubsysID int32) int32 {
	if idx := getSubsysIndex(srcID); idx >= 0 && idx < cTCPServerMaxSubsysNum {
		return int32(idx + 1)
	}
	if payloadSubsysID >= 1 && payloadSubsysID <= cTCPServerMaxSubsysNum {
		return payloadSubsysID
	}
	if idx := getSubsysIndex(payloadSubsysID); idx >= 0 && idx < cTCPServerMaxSubsysNum {
		return int32(idx + 1)
	}
	return 1
}

func StopCTCPServer() int {
	cTCPServerMu.Lock()
	imageServer := cTCPServerImage
	statServer := cTCPServerStat
	cTCPServerImage = nil
	cTCPServerStat = nil
	cTCPServerMu.Unlock()

	if imageServer != nil {
		imageServer.stopServer()
	}
	if statServer != nil {
		statServer.stopServer()
	}

	StopStGradeInfoPeriodicLog()
	StopStMotorInfoPeriodicLog()
	StopStParasImageFieldsPeriodicLog()
	StopStGlobalExitInfoPeriodicLog()
	StopStStatisticsSpeedPublisher()
	resetRealtimeSaveState()
	setCTCPServerLastMessage("CTCP servers stopped")
	return 0
}

// LastCTCPServerMessage 每次只取出队列中的一条日志（FIFO）。
// 原先曾把所有条目 join 成一条字符串，ArkTS 单次 hilog 有长度上限，会导致 StGlobal 十六进制等大段内容被截断。
func LastCTCPServerMessage() string {
	cTCPServerLastMu.Lock()
	defer cTCPServerLastMu.Unlock()
	if len(cTCPServerMessages) == 0 {
		return ""
	}
	msg := cTCPServerMessages[0]
	cTCPServerMessages = cTCPServerMessages[1:]
	return msg
}

func newCTCPServer(name string, port int) (*cTCPServer, error) {
	addr := net.JoinHostPort(cTCPServerListenIP, strconv.Itoa(port))
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	return &cTCPServer{
		name:     name,
		port:     port,
		listener: listener,
		stop:     make(chan struct{}),
	}, nil
}

func (s *cTCPServer) start() {
	s.wg.Add(1)
	go s.acceptLoop()
}

func (s *cTCPServer) stopServer() {
	select {
	case <-s.stop:
	default:
		close(s.stop)
	}
	if s.listener != nil {
		_ = s.listener.Close()
	}
	s.wg.Wait()
}

func (s *cTCPServer) acceptLoop() {
	defer s.wg.Done()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.stop:
				return
			default:
				setCTCPServerLastMessage("CTCP %s server accept error on port %d: %v", s.name, s.port, err)
				time.Sleep(cTCPServerAcceptRetryDelay)
				continue
			}
		}

		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.handleConnection(conn)
		}()
	}
}

func (s *cTCPServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	remoteAddr := conn.RemoteAddr().String()
	_ = conn.SetDeadline(time.Now().Add(cTCPServerReadTimeout))

	if err := recvCTCPSync(conn); err != nil {
		setCTCPServerLastMessage("CTCP %s server rejected %s on port %d: %v", s.name, remoteAddr, s.port, err)
		return
	}

	head, err := recvCTCPCommand(conn)
	if err != nil {
		setCTCPServerLastMessage("CTCP %s server read command failed from %s on port %d: %v", s.name, remoteAddr, s.port, err)
		return
	}
	if head.NDstId != cTCPHcID {
		setCTCPServerLastMessage("CTCP %s server ignored %s on port %d: dst=0x%04X, want HC_ID=0x%04X, src=0x%04X, cmd=%s",
			s.name,
			remoteAddr,
			s.port,
			uint32(head.NDstId),
			uint32(cTCPHcID),
			uint32(head.NSrcId),
			cTCPCommandName(head.NCmdId))
		return
	}

	payload, totalAfterHead, readMode, err := recvCTCPPayload(conn, head)

	if err != nil {
		return
	}

	if head.NCmdId != cmdFSMStatistics {
		setCTCPServerLastMessage("CTCP %s server received from %s on port %d: src=0x%04X, dst=0x%04X, cmd=%s, data=%d bytes, totalAfterHead=%d bytes, mode=%s",
			s.name,
			remoteAddr,
			s.port,
			uint32(head.NSrcId),
			uint32(head.NDstId),
			cTCPCommandName(head.NCmdId),
			len(payload),
			totalAfterHead,
			readMode)
	}

	s.handleCommandPayload(remoteAddr, head, payload)
}

func recvCTCPSync(conn net.Conn) error {
	data := make([]byte, 4)
	if _, err := io.ReadFull(conn, data); err != nil {
		return err
	}
	if string(data) != "SYNC" {
		return fmt.Errorf("invalid sync %q", string(data))
	}
	return nil
}

func recvCTCPCommand(conn net.Conn) (cTCPServerCommandHead, error) {

	data := make([]byte, 12)
	if _, err := io.ReadFull(conn, data); err != nil {
		return cTCPServerCommandHead{}, err
	}

	head := cTCPServerCommandHead{
		NSrcId: int32(binary.LittleEndian.Uint32(data[0:4])),
		NDstId: int32(binary.LittleEndian.Uint32(data[4:8])),
		NCmdId: int32(binary.LittleEndian.Uint32(data[8:12])),
	}
	if head.NSrcId == -1 || head.NCmdId == -1 {
		return cTCPServerCommandHead{}, errors.New("invalid command head")
	}
	return head, nil
}

func recvCTCPPayload(conn net.Conn, head cTCPServerCommandHead) ([]byte, int, string, error) { //根据命令头读取数据
	if cTCPCommandHasPacketLength(head.NCmdId) {
		lengthData := make([]byte, 4)
		if _, err := io.ReadFull(conn, lengthData); err != nil {
			return nil, 0, "packet-length", err
		}

		length := int(int32(binary.LittleEndian.Uint32(lengthData)))
		if length < 0 || length > cTCPServerMaxPayload {
			return nil, len(lengthData), "packet-length", fmt.Errorf("invalid packet length %d", length)
		}

		payload := make([]byte, length)
		if length > 0 {
			if _, err := io.ReadFull(conn, payload); err != nil {
				return payload, len(lengthData), "packet-length", err
			}
		}
		return payload, len(lengthData) + len(payload), "packet-length", nil
	}

	if length, ok := cTCPKnownPayloadLength(head.NCmdId); ok {
		if length == 0 {
			return nil, 0, "known-zero", nil
		}
		payload := make([]byte, length)
		if _, err := io.ReadFull(conn, payload); err != nil {
			return payload, len(payload), "known-length", err
		}
		return payload, len(payload), "known-length", nil
	}

	payload, err := readUntilIdle(conn)
	return payload, len(payload), "read-until-idle", err
}

func cTCPCommandHasPacketLength(cmd int32) bool {
	switch cmd {
	case cmdIPMImageSpot, cmdIPMImage, cmdIPMImageSplice, cmdACSExitStop:
		return true
	default:
		return false
	}
}

func cTCPKnownPayloadLength(cmd int32) (int, bool) {
	switch cmd {
	case cmdFSMGetVersion, cmdWAMVersionInfo:
		return 64, true
	case cmdFSMVersionError, cmdFSMBurnFlashProgress, cmdFSMBootFlashProgress:
		return 4, true
	case cmdSIMDisplayOn, cmdSIMInspectionOff:
		return 0, true
	default:
		return 0, false
	}
}

func readUntilIdle(conn net.Conn) ([]byte, error) {
	buffer := make([]byte, 4096)
	payload := make([]byte, 0, 4096)

	for {
		_ = conn.SetReadDeadline(time.Now().Add(cTCPServerIdleReadTimeout))
		n, err := conn.Read(buffer)
		if n > 0 {
			payload = append(payload, buffer[:n]...)
		}
		if err == nil {
			continue
		}

		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return payload, nil
		}
		if errors.Is(err, io.EOF) {
			return payload, nil
		}
		return payload, err
	}
}

func (s *cTCPServer) handleCommandPayload(remoteAddr string, head cTCPServerCommandHead, payload []byte) {
	saveCTCPPayload(s.name, s.port, remoteAddr, head, payload) //保存数据s.name服务器名字 s.port端口号 remoteAddr远程地址 head命令头 payload数据内容

	switch head.NCmdId {
	case cmdFSMConfig:
		stg, err := ParseData[StGlobal](payload)
		if err != nil {
			return
		}
		cacheStParasImageFields(remoteAddr, head, stg)
		cacheStGlobalExitInfo(remoteAddr, head, stg)
		cacheHomeStatsGlobalConfig(stg)
		cacheRealtimeSaveGlobalConfig(stg)
		cacheLatestGradeInfo(head.NSrcId, stg.Grade)
		stgJSON, jsonErr := FormatDataFullJSON(stg)
		goSz := int(unsafe.Sizeof(StGlobal{}))
		setCTCPServerLastMessage(
			"CTCP %s: sizeof(StGlobal)=%d, payload=%d bytes, nSubsysNum=%d, nSubsysId=%d, nVersion=%d, gradeActiveExits=%s, gradeByExit=%s",
			cTCPCommandName(head.NCmdId),
			goSz,
			len(payload),
			stg.Sys.NSubsysNum,
			stg.NSubsysId,
			stg.NVersion,
			summarizeGradeExitMappings(stg.Grade),
			summarizeGradeExitMappingsByExit(stg.Grade),
		)
		if jsonErr == nil && stgJSON != "" {
			setCTCPLastStGlobalFullJSON(stgJSON)
			if err := PublishWebSocketJSON(webSocketTopicStGlobal, stgJSON); err != nil { //发送给前端
				setCTCPServerLastMessage("CTCP StGlobal WebSocket 推送失败: %v", err)
			}
		} else {
			setCTCPServerLastMessage("CTCP StGlobal 全量 JSON 生成失败: %v", jsonErr)
		}

	case cmdFSMStatistics: //0x1001
		state, err := ParseData[StStatistics](payload)
		if err != nil {
			setCTCPServerLastMessage("CTCP StStatistics 解析失败: %v", err)
			return
		}
		state.NSubsysId = normalizeStStatisticsSubsysID(head.NSrcId, state.NSubsysId)
		cacheStStatisticsForSpeed(state, time.Now())
		return

	case cmdFSMGradeInfo: // 0x1002
		// -------------
		grade, err := ParseData[StFruitGradeInfo](payload)
		if err != nil {
			setCTCPServerLastMessage("CTCP handled %s: parse failed (%v), payload=%d bytes, need sizeof=%d",
				cTCPCommandName(head.NCmdId), err, len(payload), int(unsafe.Sizeof(StFruitGradeInfo{})))
			return
		}
		appendFruitGradeInfoLog(remoteAddr, head, grade, len(payload))
		appendPayloadHexChunks("CTCP StFruitGradeInfo 回推", payload)
		//--------
		sourceID := cTCPFruitGradeInfoSourceID(head, grade)
		gradeJSON, jsonErr := FormatDataFullJSON(cTCPFruitGradeInfoRealtimeFrame{
			SrcID:          sourceID,
			FruitGradeInfo: sanitizeFruitGradeInfoForRealtime(grade),
		}) //转成json 字符串
		if gradeJSON != "" && jsonErr == nil {
			//----------------
			if err := PublishWebSocketJSON(webSocketTopicGrade, gradeJSON); err != nil {
				setCTCPServerLastMessage("CTCP StFruitGradeInfo WebSocket 推送失败: %v", err)
			}
			return
		}
		setCTCPServerLastMessage("CTCP StFruitGradeInfo JSON 生成失败: %v", jsonErr)

	case cmdFSMGetVersion, cmdWAMVersionInfo:
		setCTCPServerLastMessage("CTCP handled %s: version bytes=%q", cTCPCommandName(head.NCmdId), strings.TrimRight(string(payload), "\x00\r\n "))
	case cmdFSMWeightInfo, cmdWAMWeightInfo:
		weight, err := ParseData[StWeightResult](payload)
		if err != nil {
			setCTCPServerLastMessage("CTCP handled %s: parse StWeightResult failed (%v), payload=%d bytes, need sizeof=%d",
				cTCPCommandName(head.NCmdId), err, len(payload), int(unsafe.Sizeof(StWeightResult{})))
			return
		}
		setCTCPServerLastMessage(
			"CTCP StWeightResult 回推: remote=%s, src=0x%04X, dst=0x%04X, cmd=%s, payload=%d bytes, channel=0x%04X, data{CupId=%d, FruitWeight=%.3f, CupWeight=%.3f, DataADFruit=%d, DataADVehicle=%d}, paras{CupAverageWeight=%.3f, AD0=%d, AD1=%d, StandardAD0=%d, StandardAD1=%d}, FVehicleWeight0=%.3f, FVehicleWeight1=%.3f, State=%d",
			remoteAddr,
			uint32(head.NSrcId),
			uint32(head.NDstId),
			cTCPCommandName(head.NCmdId),
			len(payload),
			uint32(weight.NChannelId),
			weight.Data.NVehicleId,
			weight.Data.FFruitWeight,
			weight.Data.FVehicleWeight,
			weight.Data.NADFruit,
			weight.Data.NADVehicle,
			weight.Paras.FCupAverageWeight,
			weight.Paras.NAD0,
			weight.Paras.NAD1,
			weight.Paras.NStandardAD0,
			weight.Paras.NStandardAD1,
			weight.FVehicleWeight0,
			weight.FVehicleWeight1,
			weight.State,
		)
		appendPayloadHexChunks("CTCP StWeightResult 回推", payload)
		weightJSON, jsonErr := FormatDataFullJSON(weight)
		if weightJSON != "" && jsonErr == nil {
			if err := PublishWebSocketJSON(webSocketTopicWeightInfo, weightJSON); err != nil {
				setCTCPServerLastMessage("CTCP StWeightResult WebSocket 推送失败: %v", err)
			}
			return
		}
		setCTCPServerLastMessage("CTCP StWeightResult JSON 生成失败: %v", jsonErr)
	case cmdFSMWaveInfo, cmdWAMWaveInfo:
		setCTCPServerLastMessage("CTCP handled %s: raw StWaveInfo saved=%d bytes", cTCPCommandName(head.NCmdId), len(payload))
	case cmdFSMVersionError, cmdFSMBurnFlashProgress, cmdFSMBootFlashProgress:
		value := int32(0)
		if len(payload) >= 4 {
			value = int32(binary.LittleEndian.Uint32(payload[:4]))
		}
		setCTCPServerLastMessage("CTCP handled %s: intValue=%d, raw saved=%d bytes", cTCPCommandName(head.NCmdId), value, len(payload))
	case cmdSIMDisplayOn, cmdSIMInspectionOn, cmdSIMInspectionOff:
		setCTCPServerLastMessage("CTCP handled %s: raw SIM payload saved=%d bytes", cTCPCommandName(head.NCmdId), len(payload))
	case cmdWAMWeightGlobal:
		weightGlobal, err := ParseData[StWeightGlobal](payload)
		if err != nil {
			setCTCPServerLastMessage("CTCP handled %s: parse StWeightGlobal failed (%v), payload=%d bytes, need sizeof=%d",
				cTCPCommandName(head.NCmdId), err, len(payload), int(unsafe.Sizeof(StWeightGlobal{})))
			return
		}
		setCTCPServerLastMessage(
			"CTCP StWeightGlobal 回读: remote=%s, src=0x%04X, dst=0x%04X, nWAMId=0x%04X, gWeight{FFilterParam=%.6f, AD_Filter_ALG=%d, NEffectCupThreshold=%d, NMinGradeThreshold=%d, NCupDeviationThreshold=%d, NCupBreakageThreshold=%d, NBaseCupNum=%d, NTotalCupNums=%v, RefWeight=%d, WeightTh=%d}, weight0{FGADParam=[%.6f %.6f], FTemperatureParams=%.6f, WaveInterval=[%d %d]}",
			remoteAddr,
			uint32(head.NSrcId),
			uint32(head.NDstId),
			uint32(weightGlobal.NWAMId),
			weightGlobal.GWeight.FFilterParam,
			weightGlobal.GWeight.AD_Filter_ALG,
			weightGlobal.GWeight.NEffectCupThreshold,
			weightGlobal.GWeight.NMinGradeThreshold,
			weightGlobal.GWeight.NCupDeviationThreshold,
			weightGlobal.GWeight.NCupBreakageThreshold,
			weightGlobal.GWeight.NBaseCupNum,
			weightGlobal.GWeight.NTotalCupNums,
			weightGlobal.GWeight.RefWeight,
			weightGlobal.GWeight.WeightTh,
			weightGlobal.Weights[0].FGADParam[0],
			weightGlobal.Weights[0].FGADParam[1],
			weightGlobal.Weights[0].FTemperatureParams,
			weightGlobal.Weights[0].WaveInterval[0],
			weightGlobal.Weights[0].WaveInterval[1],
		)
		weightGlobalJSON, jsonErr := FormatDataFullJSON(weightGlobal)
		if weightGlobalJSON != "" && jsonErr == nil {
			if err := PublishWebSocketJSON(webSocketTopicWeightGlobal, weightGlobalJSON); err != nil {
				setCTCPServerLastMessage("CTCP StWeightGlobal WebSocket 推送失败: %v", err)
			}
			return
		}
		setCTCPServerLastMessage("CTCP StWeightGlobal JSON 生成失败: %v", jsonErr)
	case cmdIPMImage, cmdIPMImageSplice, cmdIPMImageSpot:
		setCTCPServerLastMessage("CTCP handled %s: raw image payload saved=%d bytes", cTCPCommandName(head.NCmdId), len(payload))
	case cmdIPMAutoBalanceCoefficient:
		setCTCPServerLastMessage("CTCP handled %s: raw StWhiteBalanceCoefficient saved=%d bytes", cTCPCommandName(head.NCmdId), len(payload))
	case cmdIPMShutterAdjust:
		setCTCPServerLastMessage("CTCP handled %s: raw StShutterAdjust saved=%d bytes", cTCPCommandName(head.NCmdId), len(payload))
	case cmdACSExitStop:
		setCTCPServerLastMessage("CTCP handled %s: raw ACS exit payload saved=%d bytes", cTCPCommandName(head.NCmdId), len(payload))
	default:
		setCTCPServerLastMessage("CTCP handled %s: raw payload saved=%d bytes", cTCPCommandName(head.NCmdId), len(payload))
	}
}

func saveCTCPPayload(serverName string, port int, remoteAddr string, head cTCPServerCommandHead, payload []byte) {
	key := fmt.Sprintf("%s:%04X:%04X", serverName, uint32(head.NSrcId), uint32(head.NCmdId))
	payloadCopy := append([]byte(nil), payload...)

	cTCPPayloadMu.Lock()
	cTCPPayloadByCmd[key] = cTCPStoredPayload{
		Head:       head,
		Payload:    payloadCopy,
		Port:       port,
		ServerName: serverName,
		RemoteAddr: remoteAddr,
		ReceivedAt: time.Now(),
	}
	cTCPPayloadMu.Unlock()
}

// ------------------------------数据处理----------------- 分开处理 后期修改成一个函数接口

// // 处理FSM_static  数据
// func parseStStatistics(payload []byte) (StStatistics, error) {
// 	n := int(unsafe.Sizeof(StStatistics{}))
// 	if len(payload) < n {
// 		return StStatistics{}, fmt.Errorf("payload too short for StStatistics: need %d, got %d", n, len(payload))
// 	}
// 	return *(*StStatistics)(unsafe.Pointer(&payload[0])), nil
// }

// // 处理StGlobal数据
// func parseStGlobal(payload []byte) (StGlobal, error) {
// 	n := int(unsafe.Sizeof(StGlobal{}))
// 	if len(payload) < n {
// 		return StGlobal{}, fmt.Errorf("payload too short for StGlobal: need %d, got %d", n, len(payload))
// 	}
// 	return *(*StGlobal)(unsafe.Pointer(&payload[0])), nil
// }

func ParseData[T any](payload []byte) (T, error) {
	var zero T
	n := int(unsafe.Sizeof(zero))
	if len(payload) < n {
		return zero, fmt.Errorf("payload too short for %T: need %d, got %d", zero, n, len(payload))
	}
	return *(*T)(unsafe.Pointer(&payload[0])), nil
}

// func parseStStatistics(payload []byte) (StStatistics, error) {
// 	if len(payload) < 7152 {
// 		return StStatistics{}, fmt.Errorf("payload too short for StStatistics: need %d, got %d", 7152, len(payload))
// 	}

// 	var state StStatistics
// 	dst := unsafe.Slice((*byte)(unsafe.Pointer(&state)), int(unsafe.Sizeof(state)))
// 	copy(dst, payload)
// 	return state, nil
// }

// 处理拿到的StGlobal数据，保存快照
func cleanTCPServerCString(data []byte) string {
	end := len(data)
	for i, value := range data {
		if value == 0 {
			end = i
			break
		}
	}

	builder := strings.Builder{}
	for _, value := range data[:end] {
		if value >= 0x20 && value <= 0x7E {
			builder.WriteByte(value)
			continue
		}
		if value == '\t' {
			builder.WriteByte(' ')
		}
	}
	return strings.TrimSpace(builder.String())
}

func cTCPCommandName(cmd int32) string {
	switch cmd {
	case cmdFSMConfig:
		return "FSM_CMD_CONFIG(0x1000)"
	case cmdFSMStatistics:
		return "FSM_CMD_STATISTICS(0x1001)"
	case cmdFSMGradeInfo:
		return "FSM_CMD_GRADEINFO(0x1002)"
	case cmdFSMWeightInfo:
		return "FSM_CMD_WEIGHTINFO(0x1003)"
	case cmdFSMWaveInfo:
		return "FSM_CMD_WAVEINFO(0x1004)"
	case cmdFSMVersionError:
		return "FSM_CMD_VERSIONERROR(0x1005)"
	case cmdFSMBurnFlashProgress:
		return "FSM_CMD_BURN_FLASH_PROGRESS(0x1006)"
	case cmdFSMBurnDebug:
		return "FSM_CMD_BURN_DEBUG(0x1007)"
	case cmdFSMGetVersion:
		return "FSM_CMD_GETVERSION(0x1008)"
	case cmdFSMBootFlashProgress:
		return "FSM_CMD_BOOT_FLASH_PROGRESS(0x1009)"
	case cmdWAMWeightGlobal:
		return "WAM__CMD_WEIGHT_INFO(0x0120)"
	case cmdWAMWeightInfo:
		return "WAM_CMD_WEIGHTINFO(0x0121)"
	case cmdWAMWaveInfo:
		return "WAM_CMD_WAVEINFO(0x0122)"
	case cmdWAMVersionInfo:
		return "WAM_CMD_REP_WAM_INFO(0x0123)"
	case cmdIPMImage:
		return "IPM_CMD_IMAGE(0x3000)"
	case cmdIPMAutoBalanceCoefficient:
		return "IPM_CMD_AUTOBALANCE_COEFFICIENT(0x3001)"
	case cmdIPMImageSplice:
		return "IPM_CMD_IMAGE_SPLICE(0x3002)"
	case cmdIPMImageSpot:
		return "IPM_CMD_IMAGE_SPOT(0x3003)"
	case cmdIPMShutterAdjust:
		return "IPM_CMD_SHUTTER_ADJUST(0x3004)"
	case cmdACSExitStop:
		return "ACS_HMI_EXIT_STOP(0x8000)"
	case cmdSIMDisplayOn:
		return "SIM_HMI_DISPLAY_ON(0x7000)"
	case cmdSIMInspectionOn:
		return "SIM_HMI_INSPECTION_ON(0x7001)"
	case cmdSIMInspectionBadNumber:
		return "SIM_HMI_INSPECTION__BADNUMBER(0x7002)"
	case cmdSIMInspectionOff:
		return "SIM_HMI_INSPECTION_OFF(0x7003)"
	case cmdSIMInspectionConveyorStart:
		return "SIM_HMI_P_START(0x7004)"
	case cmdSIMInspectionConveyorClose:
		return "SIM_HMI_P_CLOSE(0x7005)"
	default:
		return fmt.Sprintf("UNKNOWN(0x%04X)", uint32(cmd))
	}
}

func setCTCPServerLastMessage(format string, args ...any) { //格式化字符串并保存最后的消息
	message := time.Now().Format("15:04:05.000 ") + fmt.Sprintf(format, args...)
	cTCPServerLastMu.Lock()
	cTCPServerLastMessage = message
	cTCPServerMessages = append(cTCPServerMessages, message)
	const cTCPServerMaxBufferedMessages = 2048
	if len(cTCPServerMessages) > cTCPServerMaxBufferedMessages {
		cTCPServerMessages = cTCPServerMessages[len(cTCPServerMessages)-cTCPServerMaxBufferedMessages:]
	}
	cTCPServerLastMu.Unlock()
	fmt.Println(message)
}
