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
	cTCPServerMaxQualityGradeNum  = 16
	cTCPServerMaxSizeGradeNum     = 16
	cTCPServerMaxTextLength       = 12
	cTCPServerMaxFruitNameLength  = 50
	cTCPServerParasTagInfoNum     = 6
	cTCPServerMinorGradeNum       = 6
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
)

func StartCTCPServer() int {
	cTCPServerMu.Lock()
	defer cTCPServerMu.Unlock()

	if cTCPServerImage != nil && cTCPServerStat != nil {
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
	return cTCPServerStatPort
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

	setCTCPServerLastMessage("CTCP servers stopped")
	return 0
}

func LastCTCPServerMessage() string {
	cTCPServerLastMu.Lock()
	defer cTCPServerLastMu.Unlock()
	if len(cTCPServerMessages) == 0 {
		return ""
	}

	message := strings.Join(cTCPServerMessages, "\n")
	cTCPServerMessages = cTCPServerMessages[:0]
	return message
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
		setCTCPServerLastMessage("CTCP %s server payload read failed from %s on port %d: src=0x%04X, cmd=%s, mode=%s, received=%d, err=%v",
			s.name,
			remoteAddr,
			s.port,
			uint32(head.NSrcId),
			cTCPCommandName(head.NCmdId),
			readMode,
			len(payload),
			err)
		return
	}

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

func recvCTCPPayload(conn net.Conn, head cTCPServerCommandHead) ([]byte, int, string, error) {
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
	saveCTCPPayload(s.name, s.port, remoteAddr, head, payload)

	switch head.NCmdId {
	case cmdFSMConfig:
		sysConfig, err := parseStGlobalSysConfig(payload)
		if err != nil {
			setCTCPServerLastMessage("CTCP parsed %s failed: %v, raw saved=%d bytes",
				cTCPCommandName(head.NCmdId),
				err,
				len(payload))
			return
		}
		gradeInfo, gradeErr := parseStGlobalGradeInfo(payload, sysConfig)
		if gradeErr != nil {
			setCTCPServerLastMessage("CTCP parsed %s: raw StGlobal saved=%d bytes, StSysConfig{maxExit=%d, channels=%v, imageUV=%v, width=%d, height=%d, packetSize=%d, systemInfo=0x%04X, subsysNum=%d, exitNum=%d, classification=0x%02X, cameraType=%d, checkExit=%d, checkNum=%d}; StGradeInfo parse failed: %v",
				cTCPCommandName(head.NCmdId),
				len(payload),
				sysConfig.MaxExitNum,
				sysConfig.NChannelInfo,
				sysConfig.NImageUV,
				sysConfig.Width,
				sysConfig.Height,
				sysConfig.PacketSize,
				sysConfig.NSystemInfo,
				sysConfig.NSubsysNum,
				sysConfig.NExitNum,
				sysConfig.NClassificationInfo,
				sysConfig.NCameraType,
				sysConfig.CheckExit,
				sysConfig.CheckNum,
				gradeErr)
			return
		}
		configSnapshot := saveCTCPConfigSnapshot(s.name, s.port, remoteAddr, head, payload, sysConfig, gradeInfo)
		setCTCPServerLastMessage("CTCP parsed %s: raw StGlobal saved=%d bytes, StSysConfig{maxExit=%d, channels=%v, imageUV=%v, width=%d, height=%d, packetSize=%d, systemInfo=0x%04X, subsysNum=%d, exitNum=%d, classification=0x%02X, cameraType=%d, checkExit=%d, checkNum=%d}; StGradeInfo{offset=%d, size=%d, maxExit=%d, gradeItemSize=%d, fruitType=%d, fruitName=%q, sizeGradeNum=%d, qualityGradeNum=%d, classifyType=0x%02X, checkNum=%d, forceChannel=%d, colorType=0x%02X, labelType=%d, exitEnabled=%v, colorIntervals=%v, firstGrade={exit=0x%X, min=%.2f, max=%.2f, fruitNum=%d, color=%d, shape=%d, flaw=%d}}",
			cTCPCommandName(head.NCmdId),
			len(payload),
			sysConfig.MaxExitNum,
			sysConfig.NChannelInfo,
			sysConfig.NImageUV,
			sysConfig.Width,
			sysConfig.Height,
			sysConfig.PacketSize,
			sysConfig.NSystemInfo,
			sysConfig.NSubsysNum,
			sysConfig.NExitNum,
			sysConfig.NClassificationInfo,
			sysConfig.NCameraType,
			sysConfig.CheckExit,
			sysConfig.CheckNum,
			gradeInfo.Offset,
			gradeInfo.RawSize,
			gradeInfo.MaxExitNum,
			gradeInfo.GradeItemSize,
			gradeInfo.NFruitType,
			gradeInfo.FruitName,
			gradeInfo.NSizeGradeNum,
			gradeInfo.NQualityGradeNum,
			gradeInfo.NClassifyType,
			gradeInfo.NCheckNum,
			gradeInfo.ForceChannel,
			gradeInfo.ColorType,
			gradeInfo.NLabelType,
			gradeInfo.ExitEnabled,
			gradeInfo.ColorIntervals,
			gradeInfo.FirstGrade.Exit,
			gradeInfo.FirstGrade.NMinSize,
			gradeInfo.FirstGrade.NMaxSize,
			gradeInfo.FirstGrade.NFruitNum,
			gradeInfo.FirstGrade.NColorGrade,
			gradeInfo.FirstGrade.SBShapeSize,
			gradeInfo.FirstGrade.SBFlawArea)
		if configJSON, err := formatCTCPConfigSnapshotJSON(configSnapshot); err == nil {
			setCTCPServerLastMessage("CTCP config JSON:\n%s", configJSON)
		} else {
			setCTCPServerLastMessage("CTCP config JSON marshal failed: %v", err)
		}
	case cmdFSMStatistics:
		setCTCPServerLastMessage("CTCP handled %s: raw StStatistics saved=%d bytes", cTCPCommandName(head.NCmdId), len(payload))
	case cmdFSMGradeInfo:
		setCTCPServerLastMessage("CTCP handled %s: raw StFruitGradeInfo saved=%d bytes", cTCPCommandName(head.NCmdId), len(payload))
	case cmdFSMGetVersion, cmdWAMVersionInfo:
		setCTCPServerLastMessage("CTCP handled %s: version bytes=%q", cTCPCommandName(head.NCmdId), strings.TrimRight(string(payload), "\x00\r\n "))
	case cmdFSMWeightInfo, cmdWAMWeightInfo:
		setCTCPServerLastMessage("CTCP handled %s: raw StWeightResult saved=%d bytes", cTCPCommandName(head.NCmdId), len(payload))
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
		setCTCPServerLastMessage("CTCP handled %s: raw StWeightGlobal saved=%d bytes", cTCPCommandName(head.NCmdId), len(payload))
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

func parseStGlobalSysConfig(payload []byte) (StSysConfig, error) {
	if len(payload) == 0 {
		return StSysConfig{}, errors.New("empty StGlobal payload")
	}

	candidates := []int{cTCPServerStSysConfigExit48, cTCPServerStSysConfigExit64}
	var best StSysConfig
	bestScore := -1

	for _, maxExitNum := range candidates {
		sysConfig, err := parseStSysConfig(payload, maxExitNum)
		if err != nil {
			continue
		}
		score := scoreStSysConfig(sysConfig)
		if score > bestScore {
			best = sysConfig
			bestScore = score
		}
	}

	if bestScore < 0 {
		return StSysConfig{}, fmt.Errorf("payload too short for StSysConfig, got %d bytes", len(payload))
	}
	return best, nil
}

func parseStSysConfig(payload []byte, maxExitNum int) (StSysConfig, error) {
	offset := maxExitNum * 2 * 4
	minSize := offset + cTCPServerMaxSubsysNum*5 + cTCPServerMaxCameraNum*2*4 + 4*3 + 2 + 14
	if len(payload) < minSize {
		return StSysConfig{}, fmt.Errorf("payload too short for maxExit=%d: need %d, got %d", maxExitNum, minSize, len(payload))
	}

	sysConfig := StSysConfig{
		MaxExitNum: maxExitNum,
	}

	copy(sysConfig.NChannelInfo[:], payload[offset:offset+cTCPServerMaxSubsysNum])
	offset += cTCPServerMaxSubsysNum
	copy(sysConfig.NImageUV[:], payload[offset:offset+cTCPServerMaxSubsysNum])
	offset += cTCPServerMaxSubsysNum
	copy(sysConfig.NDataRegistration[:], payload[offset:offset+cTCPServerMaxSubsysNum])
	offset += cTCPServerMaxSubsysNum
	copy(sysConfig.NImageSugar[:], payload[offset:offset+cTCPServerMaxSubsysNum])
	offset += cTCPServerMaxSubsysNum
	copy(sysConfig.NImageUltrasonic[:], payload[offset:offset+cTCPServerMaxSubsysNum])
	offset += cTCPServerMaxSubsysNum

	for i := range sysConfig.NCameraDelay {
		sysConfig.NCameraDelay[i] = int32(binary.LittleEndian.Uint32(payload[offset : offset+4]))
		offset += 4
	}

	sysConfig.Width = int32(binary.LittleEndian.Uint32(payload[offset : offset+4]))
	offset += 4
	sysConfig.Height = int32(binary.LittleEndian.Uint32(payload[offset : offset+4]))
	offset += 4
	sysConfig.PacketSize = int32(binary.LittleEndian.Uint32(payload[offset : offset+4]))
	offset += 4
	sysConfig.NSystemInfo = binary.LittleEndian.Uint16(payload[offset : offset+2])
	offset += 2
	sysConfig.NSubsysNum = payload[offset]
	offset++
	sysConfig.NExitNum = payload[offset]
	offset++
	sysConfig.NClassificationInfo = payload[offset]
	offset++
	sysConfig.MultiFreq = payload[offset]
	offset++
	sysConfig.NCameraType = payload[offset]
	offset++
	sysConfig.CIRClassifyType = payload[offset]
	offset++
	sysConfig.UVClassifyType = payload[offset]
	offset++
	sysConfig.WeightClassifyType = payload[offset]
	offset++
	sysConfig.InternalClassifyType = payload[offset]
	offset++
	sysConfig.UltrasonicClassifyType = payload[offset]
	offset++
	sysConfig.IfWIFIEnable = payload[offset]
	offset++
	sysConfig.CheckExit = payload[offset]
	offset++
	sysConfig.CheckNum = payload[offset]
	offset++
	sysConfig.NIQSEnable = payload[offset]
	offset++
	sysConfig.RawSize = alignTCPServerOffset(offset, 4)

	return sysConfig, nil
}

func parseStGlobalGradeInfo(payload []byte, sysConfig StSysConfig) (StGradeInfo, error) {
	if len(payload) <= sysConfig.RawSize {
		return StGradeInfo{}, fmt.Errorf("payload too short for StGradeInfo: sysSize=%d, got %d", sysConfig.RawSize, len(payload))
	}

	candidates := []struct {
		offset        int
		maxExitNum    int
		gradeItemSize int
	}{
		{sysConfig.RawSize, sysConfig.MaxExitNum, 36},
	}

	var best StGradeInfo
	bestScore := -1
	var lastErr error
	for _, candidate := range candidates {
		gradeInfo, err := parseStGradeInfoAt(payload, candidate.offset, candidate.maxExitNum, candidate.gradeItemSize)
		if err != nil {
			lastErr = err
			continue
		}
		score := scoreStGradeInfo(gradeInfo)
		if score > bestScore {
			best = gradeInfo
			bestScore = score
		}
	}

	if bestScore < 0 {
		if lastErr != nil {
			return StGradeInfo{}, lastErr
		}
		return StGradeInfo{}, errors.New("no valid StGradeInfo layout")
	}
	return best, nil
}

func parseStGradeInfoAt(payload []byte, base int, maxExitNum int, gradeItemSize int) (StGradeInfo, error) {
	if base < 0 || base >= len(payload) {
		return StGradeInfo{}, fmt.Errorf("invalid StGradeInfo offset %d", base)
	}
	if gradeItemSize != 36 {
		return StGradeInfo{}, fmt.Errorf("unsupported StGradeItemInfo size %d", gradeItemSize)
	}

	offset := base
	offset += cTCPServerMaxColorIntervalNum * 4
	offset += cTCPServerMaxColorGradeNum * cTCPServerMaxColorIntervalNum * 2

	gradesOffset := offset
	offset += cTCPServerMaxQualityGradeNum * cTCPServerMaxSizeGradeNum * gradeItemSize
	if len(payload) < offset+16 {
		return StGradeInfo{}, fmt.Errorf("payload too short for StGradeInfo grades: need %d, got %d", offset+16, len(payload))
	}

	gradeInfo := StGradeInfo{
		Offset:        base,
		MaxExitNum:    maxExitNum,
		GradeItemSize: gradeItemSize,
		FirstGrade:    parseStGradeItemInfo(payload[gradesOffset:], gradeItemSize),
	}
	for i := range gradeInfo.ExitEnabled {
		gradeInfo.ExitEnabled[i] = int32(binary.LittleEndian.Uint32(payload[offset : offset+4]))
		offset += 4
	}
	for i := range gradeInfo.ColorIntervals {
		gradeInfo.ColorIntervals[i] = int32(binary.LittleEndian.Uint32(payload[offset : offset+4]))
		offset += 4
	}

	offset += maxExitNum * 4
	offset += cTCPServerParasTagInfoNum
	offset = alignTCPServerOffset(offset, 4)
	if len(payload) < offset+4 {
		return StGradeInfo{}, fmt.Errorf("payload too short for StGradeInfo nFruitType: need %d, got %d", offset+4, len(payload))
	}
	gradeInfo.NFruitType = int32(binary.LittleEndian.Uint32(payload[offset : offset+4]))
	offset += 4

	if len(payload) < offset+cTCPServerMaxFruitNameLength {
		return StGradeInfo{}, fmt.Errorf("payload too short for StGradeInfo strFruitName: need %d, got %d", offset+cTCPServerMaxFruitNameLength, len(payload))
	}
	gradeInfo.FruitName = cleanTCPServerCString(payload[offset : offset+cTCPServerMaxFruitNameLength])
	offset += cTCPServerMaxFruitNameLength
	offset = alignTCPServerOffset(offset, 4)

	offset += cTCPServerMinorGradeNum * 2 * 4
	offset += cTCPServerMinorGradeNum * 2 * 4
	offset += cTCPServerMinorGradeNum * 2 * 4
	offset += cTCPServerMinorGradeNum * 4
	offset += cTCPServerMinorGradeNum * 4
	offset += cTCPServerMinorGradeNum * 4
	offset += cTCPServerMinorGradeNum * 4
	offset += cTCPServerMinorGradeNum * 4
	offset += cTCPServerMinorGradeNum * 4
	offset += cTCPServerMinorGradeNum * 4
	offset += cTCPServerMinorGradeNum * 4
	offset += cTCPServerMinorGradeNum * 4
	offset += cTCPServerMinorGradeNum * 4

	offset += cTCPServerMaxSizeGradeNum * cTCPServerMaxTextLength
	offset += cTCPServerMaxQualityGradeNum * cTCPServerMaxTextLength
	offset += cTCPServerMinorGradeNum * cTCPServerMaxTextLength
	offset += cTCPServerMaxColorGradeNum * cTCPServerMaxTextLength
	offset += cTCPServerMinorGradeNum * cTCPServerMaxTextLength
	offset += cTCPServerMinorGradeNum * cTCPServerMaxTextLength
	offset += cTCPServerMinorGradeNum * cTCPServerMaxTextLength
	offset += cTCPServerMinorGradeNum * cTCPServerMaxTextLength
	offset += cTCPServerMinorGradeNum * cTCPServerMaxTextLength
	offset += cTCPServerMinorGradeNum * cTCPServerMaxTextLength
	offset += cTCPServerMinorGradeNum * cTCPServerMaxTextLength
	offset += cTCPServerMinorGradeNum * cTCPServerMaxTextLength
	offset += cTCPServerMinorGradeNum * cTCPServerMaxTextLength
	offset += cTCPServerMinorGradeNum * cTCPServerMaxTextLength
	offset += cTCPServerMinorGradeNum * cTCPServerMaxTextLength
	offset += cTCPServerMinorGradeNum * cTCPServerMaxTextLength

	needTail := offset + 2 + maxExitNum + maxExitNum + 3
	if len(payload) < needTail {
		return StGradeInfo{}, fmt.Errorf("payload too short for StGradeInfo tail: need %d, got %d", needTail, len(payload))
	}
	gradeInfo.ColorType = payload[offset]
	offset++
	gradeInfo.NLabelType = payload[offset]
	offset++
	offset += maxExitNum
	offset += maxExitNum
	gradeInfo.NSizeGradeNum = payload[offset]
	offset++
	gradeInfo.NQualityGradeNum = payload[offset]
	offset++
	gradeInfo.NClassifyType = payload[offset]
	offset++
	offset = alignTCPServerOffset(offset, 2)
	if len(payload) < offset+4 {
		return StGradeInfo{}, fmt.Errorf("payload too short for StGradeInfo check/force: need %d, got %d", offset+4, len(payload))
	}
	gradeInfo.NCheckNum = int16(binary.LittleEndian.Uint16(payload[offset : offset+2]))
	offset += 2
	gradeInfo.ForceChannel = int16(binary.LittleEndian.Uint16(payload[offset : offset+2]))
	offset += 2
	gradeInfo.RawSize = alignTCPServerOffset(offset-base, 4)

	return gradeInfo, nil
}

func parseStGradeItemInfo(data []byte, itemSize int) StGradeItemInfo {
	grade := StGradeItemInfo{}
	if len(data) < itemSize {
		return grade
	}

	offset := 0
	grade.Exit = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8
	grade.NMinSize = math.Float32frombits(binary.LittleEndian.Uint32(data[offset : offset+4]))
	offset += 4
	grade.NMaxSize = math.Float32frombits(binary.LittleEndian.Uint32(data[offset : offset+4]))
	offset += 4
	grade.NFruitNum = int32(binary.LittleEndian.Uint32(data[offset : offset+4]))
	offset += 4
	grade.NColorGrade = int8(data[offset])
	offset++
	grade.SBShapeSize = int8(data[offset])
	offset++
	grade.SBDensity = int8(data[offset])
	offset++
	grade.SBFlawArea = int8(data[offset])
	offset++
	grade.SBBruise = int8(data[offset])
	offset++
	grade.SBRot = int8(data[offset])
	offset++
	grade.SBSugar = int8(data[offset])
	offset++
	grade.SBAcidity = int8(data[offset])
	offset++
	grade.SBHollow = int8(data[offset])
	offset++
	grade.SBSkin = int8(data[offset])
	offset++
	grade.SBBrown = int8(data[offset])
	offset++
	grade.SBTangxin = int8(data[offset])
	offset++
	grade.SBRigidity = int8(data[offset])
	offset++
	grade.SBWater = int8(data[offset])
	offset++
	grade.SBLabelByGrade = int8(data[offset])

	return grade
}

func scoreStGradeInfo(gradeInfo StGradeInfo) int {
	score := 0
	if gradeInfo.MaxExitNum == cTCPServerStSysConfigExit48 || gradeInfo.MaxExitNum == cTCPServerStSysConfigExit64 {
		score++
	}
	if gradeInfo.GradeItemSize == 36 {
		score++
	}
	if gradeInfo.NSizeGradeNum <= cTCPServerMaxSizeGradeNum {
		score += 2
	}
	if gradeInfo.NQualityGradeNum <= cTCPServerMaxQualityGradeNum {
		score += 2
	}
	if gradeInfo.NFruitType >= 0 && gradeInfo.NFruitType <= 512 {
		score += 3
	} else {
		score -= 4
	}
	if gradeInfo.NClassifyType <= 0x1F {
		score++
	}
	if gradeInfo.NLabelType <= 2 {
		score++
	}
	if !isTCPServerRepeated7F(gradeInfo.ExitEnabled[0]) && !isTCPServerRepeated7F(gradeInfo.ExitEnabled[1]) {
		score += 2
	} else {
		score -= 4
	}
	if !isTCPServerRepeated7F(gradeInfo.ColorIntervals[0]) && !isTCPServerRepeated7F(gradeInfo.ColorIntervals[1]) {
		score += 2
	} else {
		score -= 4
	}
	if gradeInfo.NCheckNum >= 0 && gradeInfo.NCheckNum <= 1000 {
		score++
	}
	if gradeInfo.ForceChannel >= 0 && gradeInfo.ForceChannel <= 1 {
		score++
	}
	if gradeInfo.FirstGrade.NFruitNum >= 0 && gradeInfo.FirstGrade.NFruitNum <= 1000000 {
		score++
	} else {
		score -= 2
	}
	if !math.IsNaN(float64(gradeInfo.FirstGrade.NMinSize)) && !math.IsNaN(float64(gradeInfo.FirstGrade.NMaxSize)) {
		score++
	}
	if gradeInfo.FirstGrade.NMinSize >= -1 && gradeInfo.FirstGrade.NMinSize <= 100000 &&
		gradeInfo.FirstGrade.NMaxSize >= -1 && gradeInfo.FirstGrade.NMaxSize <= 100000 {
		score++
	}
	return score
}

func isTCPServerRepeated7F(value int32) bool {
	return uint32(value) == 0x7F7F7F7F
}

func scoreStSysConfig(sysConfig StSysConfig) int {
	score := 0
	if sysConfig.NSubsysNum <= cTCPServerMaxSubsysNum {
		score += 2
	}
	if sysConfig.NExitNum <= uint8(sysConfig.MaxExitNum) {
		score += 2
	}
	if sysConfig.Width >= 0 && sysConfig.Width <= 10000 {
		score++
	}
	if sysConfig.Height >= 0 && sysConfig.Height <= 10000 {
		score++
	}
	if sysConfig.PacketSize >= 0 && sysConfig.PacketSize <= cTCPServerMaxPayload {
		score++
	}
	if allBytesAtMost(sysConfig.NChannelInfo[:], cTCPServerStSysConfigMaxChan) {
		score += 2
	}
	if allBytesAtMost(sysConfig.NImageUV[:], cTCPServerStSysConfigMaxChan) {
		score++
	}
	if sysConfig.NCameraType <= 10 {
		score++
	}
	if sysConfig.CheckExit <= sysConfig.NExitNum || sysConfig.CheckExit == 0 {
		score++
	}
	return score
}

func allBytesAtMost(values []uint8, max uint8) bool {
	for _, value := range values {
		if value > max {
			return false
		}
	}
	return true
}

func alignTCPServerOffset(offset int, alignment int) int {
	if alignment <= 1 {
		return offset
	}
	remainder := offset % alignment
	if remainder == 0 {
		return offset
	}
	return offset + alignment - remainder
}

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

func setCTCPServerLastMessage(format string, args ...any) {
	message := time.Now().Format("15:04:05.000 ") + fmt.Sprintf(format, args...)
	cTCPServerLastMu.Lock()
	cTCPServerLastMessage = message
	cTCPServerMessages = append(cTCPServerMessages, message)
	if len(cTCPServerMessages) > 100 {
		cTCPServerMessages = cTCPServerMessages[len(cTCPServerMessages)-100:]
	}
	cTCPServerLastMu.Unlock()
	fmt.Println(message)
}
