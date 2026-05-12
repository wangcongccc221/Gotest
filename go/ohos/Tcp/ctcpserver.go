package tcp

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
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
	appendCTCPLogChunks("CTCP startup StGlobalLayoutReport", StGlobalLayoutReport())
	setCTCPServerLastMessage("CTCP startup: 完整字段 JSON 仅在收到 FSM_CMD_CONFIG(0x1000) 后输出；ArkTS 可调用 lastStGlobalFullJSON() 取全文（若已实现 NAPI）。")
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

// appendPayloadHexChunks 将原始 payload 以十六进制按行输出（16 字节/行），再按块写入日志。
func appendPayloadHexChunks(tag string, payload []byte) {
	if len(payload) == 0 {
		setCTCPServerLastMessage("%s hexdump: <empty payload>", tag)
		return
	}
	var b strings.Builder
	for i := 0; i < len(payload); i += 16 {
		end := i + 16
		if end > len(payload) {
			end = len(payload)
		}
		fmt.Fprintf(&b, "%08x:", i)
		for j := i; j < end; j++ {
			fmt.Fprintf(&b, " %02x", payload[j])
		}
		b.WriteByte('\n')
	}
	appendCTCPLogChunks(tag+" hexdump", b.String())
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
			setCTCPServerLastMessage("CTCP handled %s: parse failed (%v), payload=%d bytes",
				cTCPCommandName(head.NCmdId), err, len(payload))
			appendPayloadHexChunks("CTCP StGlobal raw (parse fail)", payload)
			return
		}
		_, fullJSON := saveCTCPGlobalSnapshot(s.name, s.port, remoteAddr, head, payload, stg)
		goSz := int(unsafe.Sizeof(StGlobal{}))
		setCTCPServerLastMessage(
			"CTCP %s: sizeof(StGlobal)=%d, payload=%d bytes, nSubsysId=%d, nVersion=%d",
			cTCPCommandName(head.NCmdId),
			goSz,
			len(payload),
			stg.nSubsysId,
			stg.nVersion,
		)
		if fullJSON != "" {
			setCTCPServerLastMessage("CTCP StGlobal ===== 全量解析(JSON，反射整棵结构体)开始，多段输出 =====")
			appendCTCPLogChunks("CTCP StGlobal 全量", fullJSON)
			setCTCPServerLastMessage("CTCP StGlobal ===== 全量解析(JSON)结束 =====")
		} else {
			setCTCPServerLastMessage("CTCP StGlobal 全量 JSON 生成失败")
		}
		setCTCPServerLastMessage("CTCP StGlobal ===== 以下为原始 payload 十六进制 =====")
		appendPayloadHexChunks("CTCP StGlobal raw wire", payload)
	case cmdFSMStatistics: //0x1001
		stats, err := ParseData[StStatistics](payload)
		if err != nil {
			return
		}
		_ = saveCTCPStatisticsSnapshot(s.name, s.port, remoteAddr, head, stats)

	case cmdFSMGradeInfo: // 0x1002
		grade, err := ParseData[StFruitGradeInfos](payload)
		if err != nil {
			setCTCPServerLastMessage("CTCP handled %s: parse failed (%v), payload=%d bytes, need sizeof=%d",
				cTCPCommandName(head.NCmdId), err, len(payload), int(unsafe.Sizeof(StFruitGradeInfos{})))
			return
		}
		fmt.Println(grade)
		setCTCPServerLastMessage("CTCP parsed %s: sizeof=%d payload=%d FruitGradeInfos[0].nRouteId=%d",
			cTCPCommandName(head.NCmdId),
			int(unsafe.Sizeof(StFruitGradeInfos{})),
			len(payload),
			grade.FruitGradeInfos[0].nRouteId,
		)
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
