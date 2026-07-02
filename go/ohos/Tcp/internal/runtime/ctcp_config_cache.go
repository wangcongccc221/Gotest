package tcp

import (
	"fmt"
	"strings"
	"sync"
	"time"
	"unsafe"
)

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

var (
	cTCPStParasImageFieldsMu        sync.Mutex
	cTCPStParasImageFieldsLatest    cTCPStParasImageFieldsSnapshot
	cTCPStParasImageFieldsHasLatest bool
	cTCPStParasImageFieldsStop      chan struct{}

	cTCPStGlobalExitInfoMu        sync.Mutex
	cTCPStGlobalExitInfoLatest    cTCPStGlobalExitInfoSnapshot
	cTCPStGlobalExitInfoHasLatest bool
	cTCPStGlobalExitInfoStop      chan struct{}
)

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
