package client

import (
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	tcpWriteTimeout = 3 * time.Second //超时设置
	STX             = 0x02            // 同步头（RIS模式）
	SBC_INFO_REQ    = 0x18            // 设备信息请求
)

var (
	activeTCPClientMu sync.Mutex
	activeTCPClient   *TcpClient
)

// TcpClient TCP客户端 - 事件驱动设计
type TcpClient struct {
	mu         sync.Mutex
	conn       net.Conn
	remoteAddr string
	localAddr  string

	// 事件回调函数
	OnConnected    func()
	OnDataReceived func([]byte)
	OnError        func(string)
	OnClosed       func()

	reading         bool
	autoSendInitCmd bool // 是否在连接后自动发送初始化命令
	initCmdSent     bool // 初始化命令是否已发送
}

// NewTcpClient 1.创建新的TCP客户端
func NewTcpClient(remoteAddr, localAddr string) *TcpClient {
	return &TcpClient{
		remoteAddr: remoteAddr,
		localAddr:  localAddr,
	}
}

func StartTCPClient(remoteIP string, remotePort int, localIP string) int {
	remoteIP = strings.TrimSpace(remoteIP)
	localIP = strings.TrimSpace(localIP)
	if remoteIP == "" || remotePort <= 0 || remotePort > 65535 {
		return -1
	}

	remoteAddr := net.JoinHostPort(remoteIP, strconv.Itoa(remotePort))

	activeTCPClientMu.Lock()
	defer activeTCPClientMu.Unlock()

	if activeTCPClient != nil && activeTCPClient.IsConnected() {
		if activeTCPClient.remoteAddr == remoteAddr && activeTCPClient.localAddr == localIP {
			return remotePort
		}
		activeTCPClient.Close()
		activeTCPClient = nil
	}

	client := NewTcpClient(remoteAddr, localIP)
	if !client.ConnectServer() {
		return -1
	}
	activeTCPClient = client
	return remotePort
}

func StopTCPClient() int {
	activeTCPClientMu.Lock()
	client := activeTCPClient
	activeTCPClient = nil
	activeTCPClientMu.Unlock()

	if client != nil {
		client.Close()
	}
	return 0
}

func TCPClientSend(message string) string {
	activeTCPClientMu.Lock()
	client := activeTCPClient
	activeTCPClientMu.Unlock()

	if client == nil || !client.Send([]byte(message)) {
		return ""
	}
	return "sent"
}

// 2.ConnectServer 连接到服务器，异步处理
func (tc *TcpClient) ConnectServer() bool {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if tc.conn != nil {
		return true // 已连接
	}

	conn, err := tc.dialTCP() //拨号连接
	if err != nil {
		tc.emitError(err.Error())
		return false
	}

	tc.conn = conn
	tc.reading = true

	// 启动异步读取协程
	go tc.readLoop()

	tc.emitConnected()
	return true
}

// 5.IsConnected 判断是否有连接对象
func (tc *TcpClient) IsConnected() bool {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	return tc.conn != nil
}

// 3.Send 发送数据，同步操作
func (tc *TcpClient) Send(data []byte) bool {
	if len(data) == 0 {
		return false
	}

	tc.mu.Lock()
	if tc.conn == nil {
		tc.mu.Unlock()
		return false
	}
	conn := tc.conn
	tc.mu.Unlock()

	_ = conn.SetWriteDeadline(time.Now().Add(tcpWriteTimeout))
	if err := tc.writeAll(conn, data); err != nil {
		tc.handleDisconnect(err.Error())
		return false
	}
	_ = conn.SetWriteDeadline(time.Time{})
	return true
}

// 4.Close 关闭连接
func (tc *TcpClient) Close() {
	//异步读取 需要用到互斥锁
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.closeSocketLocked()
}

// dialTCP 创建TCP连接
func (tc *TcpClient) dialTCP() (net.Conn, error) {
	dialer := net.Dialer{
		Timeout:   500 * time.Millisecond,
		KeepAlive: 30 * time.Second,
	}

	// 如果指定了本地IP，绑定到该地址
	if tc.localAddr != "" {
		localTCPAddr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(tc.localAddr, "0"))
		if err != nil {
			return nil, err
		}
		dialer.LocalAddr = localTCPAddr
	}

	conn, err := dialer.Dial("tcp", tc.remoteAddr)
	if err != nil {
		return nil, err
	}

	// 设置TCP选项：保活和禁用Nagle算法
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		_ = tcpConn.SetKeepAlive(true)
		_ = tcpConn.SetNoDelay(true)
	}

	return conn, nil
}

// readLoop 后台读取循环
func (tc *TcpClient) readLoop() {
	buffer := make([]byte, 32*1024)
	for {
		tc.mu.Lock()
		if !tc.reading || tc.conn == nil {
			tc.mu.Unlock()
			return
		}
		conn := tc.conn
		tc.mu.Unlock()

		n, err := conn.Read(buffer)
		if n > 0 {
			tc.emitDataReceived(append([]byte(nil), buffer[:n]...))
		}
		if err != nil {
			tc.handleDisconnect(err.Error())
			return
		}
	}
}

// writeAll 确保所有数据都被写入
func (tc *TcpClient) writeAll(conn net.Conn, data []byte) error {
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

// handleDisconnect 处理断开连接
func (tc *TcpClient) handleDisconnect(errMsg string) {
	tc.mu.Lock()
	tc.closeSocketLocked()
	tc.mu.Unlock()

	tc.emitError(errMsg)
	tc.emitClosed()
}

// closeSocketLocked 关闭socket（内部持锁调用）
func (tc *TcpClient) closeSocketLocked() {
	if tc.conn != nil {
		_ = tc.conn.Close()
		tc.conn = nil
	}
	tc.reading = false
}

// 事件发射函数（不持锁）
func (tc *TcpClient) emitConnected() {
	if tc.OnConnected != nil {
		go tc.OnConnected()
	}
}

func (tc *TcpClient) emitDataReceived(data []byte) {
	if tc.OnDataReceived != nil {
		go tc.OnDataReceived(data)
	}
}

func (tc *TcpClient) emitError(errMsg string) {
	if tc.OnError != nil {
		go tc.OnError(errMsg)
	}
}

func (tc *TcpClient) emitClosed() {
	if tc.OnClosed != nil {
		go tc.OnClosed()
	}
}
