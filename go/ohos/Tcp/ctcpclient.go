package tcp

import (
	// "encoding/binary"
	// "encoding/binary"
	"encoding/binary"
	"io"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	cTCPHcID                = int32(0x1000)      // ConstPreDefine::HC_ID
	cTCPSync                = uint32(0x434e5953) // "SYNC"
	cTCPDefaultConnectCount = 1
	cTCPDefaultSendTime     = 1000
)

const (
	cTCPLCIPAddrTemplate = "192.168.0."
	cTCPACSIPAddr        = "192.168.0.13"
	cTCPSIMIPAddr        = "192.168.0.14"

	cTCPFSMPort = 1279
	cTCPIPMPort = 1289
	cTCPSIMPort = 1129
	cTCPWAMPort = 1299
	cTCPACSPort = 4127

	cTCPMaxSubsysNum  = 4
	cTCPMaxChannelNum = 12
	cTCPMaxExitNum    = 48

	cTCPHCDisplayOff = int32(0x0000)
	cTCPHCDisplayOn  = int32(0x0019)
	cTCPHCSysConfig  = int32(0x0050)
)

var cTCPMutex sync.Mutex

type SendCMD struct {
	SYNC    uint32
	NSrcId  int32
	NDestId int32
	NCmd    int32
}

type CTcpClient struct {
	mIsConnected     []bool
	nTryConnectCount int
	nTrySendTime     int
}

func NewCTcpClient() *CTcpClient {
	return &CTcpClient{
		nTryConnectCount: cTCPDefaultConnectCount,
		nTrySendTime:     cTCPDefaultSendTime,
	}
}

func (tc *CTcpClient) SetConnected(isConnected []bool) {
	tc.mIsConnected = isConnected
}

func (tc *CTcpClient) AllSysSyncRequest(nCmd int32, data []byte, bChannelInfo []uint8) int {
	arrayID := make([]int32, 0, cTCPMaxSubsysNum)
	result := 0

	if nCmd != cTCPHCSysConfig {
		arrayID = getAllSysID(bChannelInfo)
	} else {
		for i := 0; i < cTCPMaxSubsysNum; i++ {
			if channelCountAt(bChannelInfo, i) > 0 || sysConfigChannelCount(data, i) > 0 {
				arrayID = append(arrayID, encodeSubsys(i))
			}
		}
	}

	for _, nDestID := range arrayID {
		if !tc.SubsysIsConnected(getSubsysIndex(nDestID)) {
			result = -1
			continue
		}

		strIP, nPortNum := resolveCTCPTarget(nDestID, nCmd, "", 0)
		socket, err := tc.ConnectServer(strIP, nPortNum)
		if socket != nil {
			_ = socket.Close()
		}
		if err != nil {
			return -1
		}
	}

	if result == 0 {
		for _, nDestID := range arrayID {
			if !tc.SyncRequest(nDestID, nCmd, data, "", 0) {
				result = -1
			}
		}
	}

	return result
}

func (tc *CTcpClient) SyncRequest(nDestID int32, nCmd int32, data []byte, ip string, port int) bool {
	boRC := false
	var conn net.Conn

	if strings.TrimSpace(ip) == "" {
		if nCmd != cTCPHCDisplayOff &&
			nCmd != cTCPHCDisplayOn &&
			!tc.SubsysIsConnected(getSubsysIndex(nDestID)) &&
			!isMAX2WAMCommand(nCmd) {
			return false
		}
	}

	strIP, nPortNum := resolveCTCPTarget(nDestID, nCmd, ip, port)

	if !tc.Lock(1000) {
		return false
	}
	defer tc.UnLock()

	boRC = true
	socket, err := tc.ConnectServer(strIP, nPortNum)
	boRC = err == nil
	conn = socket
	if conn != nil {
		defer conn.Close()
	}

	if boRC {
		cmd := SendCMD{
			SYNC:    cTCPSync,
			NSrcId:  cTCPHcID,
			NDestId: nDestID,
			NCmd:    nCmd,
		}

		boRC = tc.Send(conn, cmd.Bytes())
		if boRC {
			if len(data) > 0 {
				boRC = tc.Send(conn, data)
			}
		}
	}

	return boRC
}

func (tc *CTcpClient) Lock(nTimeOut int) bool {
	if nTimeOut <= 0 {
		cTCPMutex.Lock()
		return true
	}

	deadline := time.Now().Add(time.Duration(nTimeOut) * time.Millisecond)
	for {
		if cTCPMutex.TryLock() {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func (tc *CTcpClient) UnLock() bool {
	cTCPMutex.Unlock()
	return true
}

func (tc *CTcpClient) Ping(hostIP string) bool {
	hostIP = strings.TrimSpace(hostIP)
	if hostIP == "" {
		return false
	}

	cmd := exec.Command("ping", hostIP, "-c", "1", "-w", "1")
	return cmd.Run() == nil
}

func (tc *CTcpClient) SubsysIsConnected(nSubsysIdx int) bool {
	if len(tc.mIsConnected) == 0 {
		return false
	}

	if nSubsysIdx == -1 {
		for _, isConnected := range tc.mIsConnected {
			if isConnected {
				return true
			}
		}
		return false
	}

	if nSubsysIdx < 0 || nSubsysIdx >= len(tc.mIsConnected) {
		return false
	}

	return tc.mIsConnected[nSubsysIdx]
}

func (tc *CTcpClient) ConnectServer(remoteIP string, remotePort int) (net.Conn, error) {
	addr := net.JoinHostPort(remoteIP, strconv.Itoa(remotePort))
	var conn net.Conn
	var err error

	tryConnectCount := tc.nTryConnectCount
	if tryConnectCount <= 0 {
		tryConnectCount = cTCPDefaultConnectCount
	}

	for i := 0; i < tryConnectCount; i++ {
		conn, err = net.DialTimeout("tcp", addr, 500*time.Millisecond)
		if err == nil {
			return conn, nil
		}
	}

	return nil, err
}

func (tc *CTcpClient) Send(conn net.Conn, data []byte) bool {
	if conn == nil {
		return false
	}
	if len(data) == 0 {
		return false
	}

	if tc.nTrySendTime > 0 {
		_ = conn.SetWriteDeadline(time.Now().Add(time.Duration(tc.nTrySendTime) * time.Millisecond))
		defer conn.SetWriteDeadline(time.Time{})
	}

	return writeAll(conn, data) == nil
}

func (cmd SendCMD) Bytes() []byte {
	data := make([]byte, 16)
	binary.LittleEndian.PutUint32(data[0:4], cmd.SYNC)
	binary.LittleEndian.PutUint32(data[4:8], uint32(cmd.NSrcId))
	binary.LittleEndian.PutUint32(data[8:12], uint32(cmd.NDestId))
	binary.LittleEndian.PutUint32(data[12:16], uint32(cmd.NCmd))
	return data
}

func resolveCTCPTarget(nDestID int32, nCmd int32, ip string, port int) (string, int) {
	if strings.TrimSpace(ip) != "" {
		return strings.TrimSpace(ip), port
	}

	nSubsysID := getSubSysID(nDestID)
	nPortNum := cTCPFSMPort

	if nCmd >= 0x2000 && nCmd < 0x3000 {
		nPortNum = cTCPIPMPort
		nSubsysID = getIPMID(nDestID)
	} else if nCmd >= 0x0110 && nCmd < 0x2000 {
		nPortNum = cTCPWAMPort
		nSubsysID = getWAMIP(nDestID)
	} else if nCmd >= 0x9000 {
		nPortNum = cTCPACSPort
		nSubsysID = ipLastByte(cTCPACSIPAddr)
	}

	strIP := cTCPLCIPAddrTemplate + strconv.Itoa(nSubsysID)
	if nCmd >= 0x7000 && nCmd <= 0x7102 {
		nPortNum = cTCPSIMPort
		strIP = cTCPSIMIPAddr
	}
	if port > 0 {
		nPortNum = port
	}

	return strIP, nPortNum
}

func isMAX2WAMCommand(nCmd int32) bool {
	return nCmd >= 0x0110 && nCmd < 0x2000
}

func getSubsysIndex(id int32) int {
	return int(((id & 0x0F00) >> 8) - 1)
}

func getSubSysID(id int32) int {
	return int((id & 0x0F00) >> 4)
}

func getIPMID(id int32) int {
	return int((id & 0x0FF0) >> 4)
}

func getWAMIP(id int32) int {
	return int((id | 0x00D0) >> 4)
}

func encodeChannel(x int, y int, z int) int32 {
	return int32((x+1)<<8 | (y+1)<<4 | (z + 1))
}

func encodeSubsys(x int) int32 {
	return encodeChannel(x, -1, -1)
}

func getAllSysID(bChannelInfo []uint8) []int32 {
	arrayID := make([]int32, 0, cTCPMaxSubsysNum)
	for i := cTCPMaxSubsysNum - 1; i >= 0; i-- {
		for j := cTCPMaxChannelNum - 1; j >= 0; j-- {
			if channelCountAt(bChannelInfo, i) > j {
				arrayID = append(arrayID, encodeSubsys(i))
				break
			}
		}
	}
	return arrayID
}

func channelCountAt(bChannelInfo []uint8, index int) int {
	if index < 0 || index >= len(bChannelInfo) {
		return 0
	}
	return int(bChannelInfo[index])
}

func sysConfigChannelCount(data []byte, index int) int {
	offset := cTCPMaxExitNum*2*4 + index
	if index < 0 || offset >= len(data) {
		return 0
	}
	return int(data[offset])
}

func ipLastByte(ip string) int {
	items := strings.Split(strings.TrimSpace(ip), ".")
	if len(items) == 0 {
		return 0
	}
	value, err := strconv.Atoi(items[len(items)-1])
	if err != nil {
		return 0
	}
	return value
}

func StartCTCPClient(remoteIP string, remotePort int, destID int32, cmd int32, data []byte) int {
	client := NewCTcpClient()
	if !client.SyncRequest(destID, cmd, data, remoteIP, remotePort) {
		return -1
	}
	return 0
}

func writeAll(conn net.Conn, data []byte) error {
	for len(data) > 0 {
		n, err := conn.Write(data)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		data = data[n:]
	}
	return nil
}
