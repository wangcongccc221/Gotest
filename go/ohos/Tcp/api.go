package tcp

import tcpclient "gotest/ohos/Tcp/client"
import runtime "gotest/ohos/Tcp/internal/runtime"

func StartServer() int {
	return runtime.StartServer()
}

func StopServer() int {
	return runtime.StopServer()
}

func StartCTCPServer() int {
	return runtime.StartCTCPServer()
}

func StopCTCPServer() int {
	return runtime.StopCTCPServer()
}

func LastCTCPServerMessage() string {
	return runtime.LastCTCPServerMessage()
}

func StGlobalLayoutReport() string {
	return runtime.StGlobalLayoutReport()
}

func StartTCPClient(remoteIP string, remotePort int, localIP string) int {
	return tcpclient.StartTCPClient(remoteIP, remotePort, localIP)
}

func StopTCPClient() int {
	return tcpclient.StopTCPClient()
}

func TCPClientSend(message string) string {
	return tcpclient.TCPClientSend(message)
}

func StartCTCPClient(remoteIP string, remotePort int, destID int32, cmd int32, data []byte) int {
	return tcpclient.StartCTCPClient(remoteIP, remotePort, destID, cmd, data)
}
