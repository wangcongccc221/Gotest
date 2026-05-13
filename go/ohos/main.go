package main

/*
#include <stdlib.h>
*/
import "C"
import (
	"unsafe"

	tcp "gotest/ohos/Tcp"
	"gotest/ohos/database"
	"gotest/ohos/modbus"
	"gotest/ohos/opcua"
)

//export GoAdd
func GoAdd(x, y C.double) C.double {
	return x + y
}

//export GoHello
func GoHello() *C.char {
	return C.CString("Hello, HarmonyOS from Go")
}

//export GoFreeCString
func GoFreeCString(value *C.char) {
	C.free(unsafe.Pointer(value))
}

//export GoStartServer
func GoStartServer() C.int {
	return C.int(tcp.StartServer())
}

//export GoStopServer
func GoStopServer() C.int {
	return C.int(tcp.StopServer())
}

//export GoStartOPCUAServer
func GoStartOPCUAServer() C.int {
	return C.int(opcua.StartOPCUAServer())
}

//export GoStopOPCUAServer
func GoStopOPCUAServer() C.int {
	return C.int(opcua.StopOPCUAServer())
}

//export GoStartModbusServer
func GoStartModbusServer() C.int {
	return C.int(modbus.StartModbusServer())
}

//export GoStopModbusServer
func GoStopModbusServer() C.int {
	return C.int(modbus.StopModbusServer())
}

//export GoInitORM
func GoInitORM() C.int {
	if err := database.InitORM(); err != nil {
		return -1
	}
	return 0
}

//export GoInitORMWithPath
func GoInitORMWithPath(path *C.char) C.int {
	if path == nil {
		return GoInitORM()
	}
	if err := database.InitORMWithPath(C.GoString(path)); err != nil {
		return -1
	}
	return 0
}

//export GoStartTCPServer
func GoStartTCPServer() C.int {
	return C.int(tcp.StartCTCPServer())
}

//export GoStopTCPServer
func GoStopTCPServer() C.int {
	return C.int(tcp.StopCTCPServer())
}

//export GoStartTCPClient
func GoStartTCPClient() C.int {
	return C.int(tcp.StartTCPClient("127.0.0.1", 9998, ""))
}

//export GoStartTCPClientWithAddress
func GoStartTCPClientWithAddress(remoteIP *C.char, remotePort C.int, localIP *C.char) C.int {
	if remoteIP == nil {
		return -1
	}

	local := ""
	if localIP != nil {
		local = C.GoString(localIP)
	}

	return C.int(tcp.StartTCPClient(C.GoString(remoteIP), int(remotePort), local))
}

//export GoStartCTCPClient
func GoStartCTCPClient(remoteIP *C.char, remotePort C.int, destID C.int, cmd C.int) C.int {
	if remoteIP == nil {
		return -1
	}

	return C.int(tcp.StartCTCPClient(
		C.GoString(remoteIP),
		int(remotePort),
		int32(destID),
		int32(cmd),
		nil,
	))
}

//export GoStopTCPClient
func GoStopTCPClient() C.int {
	return C.int(tcp.StopTCPClient())
}

//export GoTCPClientSend
func GoTCPClientSend(message *C.char) *C.char {
	if message == nil {
		return C.CString("")
	}
	return C.CString(tcp.TCPClientSend(C.GoString(message)))
}

//export GoLastTCPServerMessage
func GoLastTCPServerMessage() *C.char {
	return C.CString(tcp.LastCTCPServerMessage())
}

//export GoStGlobalLayoutReport
func GoStGlobalLayoutReport() *C.char {
	return C.CString(tcp.StGlobalLayoutReport())
}

func main() {}
