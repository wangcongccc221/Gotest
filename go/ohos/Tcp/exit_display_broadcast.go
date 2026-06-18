package tcp

import (
	"encoding/binary"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	cTCPExitDisplayBroadcastIP   = "192.168.10.255"
	cTCPExitDisplayBroadcastPort = 4567
	cTCPExitGradeNameWireSize    = 26
	cTCPExitGradeItemWireSize    = 4 + cTCPExitGradeNameWireSize
)

var (
	exitDisplayBroadcastLoopMu   sync.Mutex
	exitDisplayBroadcastLoopStop chan struct{}
	sendExitDisplayUDPPacket     = sendExitDisplayUDPPacketDefault
	exitDisplayBroadcastInterval = 5 * time.Second
)

func StartExitDisplayBroadcastLoop() {
	StartExitDisplayBroadcastLoopWithInterval(exitDisplayBroadcastInterval)
}

func StartExitDisplayBroadcastLoopWithInterval(interval time.Duration) {
	if interval <= 0 {
		interval = exitDisplayBroadcastInterval
	}

	exitDisplayBroadcastLoopMu.Lock()
	if exitDisplayBroadcastLoopStop != nil {
		exitDisplayBroadcastLoopMu.Unlock()
		return
	}
	stop := make(chan struct{})
	exitDisplayBroadcastLoopStop = stop
	exitDisplayBroadcastLoopMu.Unlock()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		setCTCPServerLastMessage("出口屏显示 UDP 广播定时器已启动: interval=%s", interval)
		for {
			select {
			case <-ticker.C:
				BroadcastExitDisplayOnce()
			case <-stop:
				setCTCPServerLastMessage("出口屏显示 UDP 广播定时器已停止")
				return
			}
		}
	}()
}

func StopExitDisplayBroadcastLoop() {
	exitDisplayBroadcastLoopMu.Lock()
	stop := exitDisplayBroadcastLoopStop
	exitDisplayBroadcastLoopStop = nil
	exitDisplayBroadcastLoopMu.Unlock()
	if stop != nil {
		close(stop)
	}
}

func BroadcastExitDisplayOnce() bool {
	if !isExitDisplayBroadcastEnabled() {
		return true
	}

	_, info, ok := latestExitDisplayInfoSnapshot()
	if !ok {
		info = defaultExitDisplayInfo()
	}
	grade, hasGrade := latestStGradeInfoSnapshot()
	packet := buildExitDisplayBroadcastPacket(info, grade, hasGrade)
	if err := sendExitDisplayUDPPacket(packet); err != nil {
		setCTCPServerLastMessage("出口屏显示 UDP 广播失败: %v", err)
		return false
	}
	setCTCPServerLastMessage("出口屏显示 UDP 广播已发送: target=%s:%d, payload=%d bytes", cTCPExitDisplayBroadcastIP, cTCPExitDisplayBroadcastPort, len(packet))
	return true
}

func buildExitDisplayBroadcastPacket(info ExitDisplayInfo, grade StGradeInfo, hasGrade bool) []byte {
	packet := make([]byte, 1+cTCP48MaxExitNum*cTCPExitGradeItemWireSize+1)
	packet[0] = 0xAA
	for exitIndex := 0; exitIndex < cTCP48MaxExitNum; exitIndex++ {
		offset := 1 + exitIndex*cTCPExitGradeItemWireSize
		binary.LittleEndian.PutUint32(packet[offset:offset+4], uint32(exitIndex+1))
		name := resolveExitDisplayGradeName(info, grade, hasGrade, exitIndex)
		copy(packet[offset+4:offset+4+cTCPExitGradeNameWireSize], []byte(name))
	}
	packet[len(packet)-1] = 0x55
	return packet
}

func resolveExitDisplayGradeName(info ExitDisplayInfo, grade StGradeInfo, hasGrade bool, exitIndex int) string {
	if exitDisplayNameEnabled(info.DisplayType, exitIndex) {
		name := strings.TrimSpace(info.DisplayNames[exitIndex])
		if name != "" {
			return name
		}
	}
	if hasGrade {
		name := exitDisplayGradeNameFromGrade(grade, exitIndex)
		if name != "" {
			return name
		}
	}
	return ""
}

func exitDisplayGradeNameFromGrade(grade StGradeInfo, exitIndex int) string {
	qualityCount := 1
	if grade.NClassifyType > 0 && grade.NQualityGradeNum > 0 {
		qualityCount = int(grade.NQualityGradeNum)
	}
	if qualityCount <= 0 {
		qualityCount = 1
	}
	if qualityCount > cTCPServerMaxSizeGradeNum {
		qualityCount = cTCPServerMaxSizeGradeNum
	}

	sizeCount := 1
	if grade.NSizeGradeNum > 0 {
		sizeCount = int(grade.NSizeGradeNum)
	}
	if sizeCount <= 0 {
		sizeCount = 1
	}
	if sizeCount > cTCPServerMaxSizeGradeNum {
		sizeCount = cTCPServerMaxSizeGradeNum
	}

	parts := make([]string, 0, sizeCount)
	for qualityIndex := 0; qualityIndex < qualityCount; qualityIndex++ {
		for sizeIndex := 0; sizeIndex < sizeCount; sizeIndex++ {
			gradeIndex := qualityIndex*cTCPServerMaxSizeGradeNum + sizeIndex
			if gradeIndex < 0 || gradeIndex >= len(grade.Grades) {
				continue
			}
			if !exitDisplayGradeHasExit(grade.Grades[gradeIndex], exitIndex) {
				continue
			}
			sizeName := realtimeSaveFixedName(grade.StrSizeGradeName[:], sizeIndex)
			if sizeName == "" {
				sizeName = realtimeSaveFixedName(grade.StrColorGradeName[:], sizeIndex)
			}
			if sizeName == "" {
				continue
			}
			if grade.NQualityGradeNum > 0 {
				qualityName := realtimeSaveFixedName(grade.StrQualityGradeName[:], qualityIndex)
				if qualityName != "" {
					sizeName += "." + qualityName
				}
			}
			parts = append(parts, sizeName)
		}
	}
	return strings.Join(parts, " ")
}

func exitDisplayGradeHasExit(item StGradeItemInfo, exitIndex int) bool {
	if exitIndex < 0 || exitIndex >= cTCP48MaxExitNum {
		return false
	}
	if exitIndex < 32 {
		return item.ExitLow&(uint32(1)<<uint(exitIndex)) != 0
	}
	return item.ExitHigh&(uint32(1)<<uint(exitIndex-32)) != 0
}

func sendExitDisplayUDPPacketDefault(packet []byte) error {
	addr := &net.UDPAddr{IP: net.ParseIP(cTCPExitDisplayBroadcastIP), Port: cTCPExitDisplayBroadcastPort}
	conn, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	_, err = conn.Write(packet)
	return err
}
