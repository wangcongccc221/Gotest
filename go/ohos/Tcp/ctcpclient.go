package tcp

import (
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
	// C++ 这里依赖 Interface/CommonFunction/ConstPreDefine 推导所有子系统目标。
	// 这些依赖还没有移植时，不能自己假造目标列表。
	return -1
}

func (tc *CTcpClient) SyncRequest(nDestID int32, nCmd int32, data []byte, ip string, port int) bool {
	boRC := false
	var conn net.Conn
	var nPortNum int
	var strIP string

	if strings.TrimSpace(ip) == "" {
		//判断nCmd是否为FSM_CMD_ON

		//获取本地IP地址

		return false
	} else {
		nPortNum = port
		strIP = strings.TrimSpace(ip)
	}

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
