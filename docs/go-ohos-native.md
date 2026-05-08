# Go Native module for HarmonyOS

This project follows the Huawei Developer article approach:

```text
ArkTS -> libentry.so NAPI wrapper -> C ABI -> Go c-shared library
```

The ArkTS API stays small:

```ts
testNapi.add(2, 3)
testNapi.hello()
testNapi.startServer()
testNapi.stopServer()
testNapi.startOpcuaServer()
testNapi.stopOpcuaServer()
testNapi.startModbusServer()
testNapi.stopModbusServer()
testNapi.initOrm('/data/storage/el2/database/harmony_go_orm.db')
testNapi.startTcpServer()
testNapi.startTcpClient()
testNapi.tcpSend('hello tcp')
testNapi.stopTcpClient()
testNapi.stopTcpServer()
```

The C++ NAPI wrapper in `entry/src/main/cpp/napi_init.cpp` calls the Go exports
declared in `entry/src/main/cpp/include/libohos.h`.

## Files

- `go/ohos/main.go`: Go source exported through cgo.
- `go/ohos/server.go`: Gin router and localhost server lifecycle.
- `go/ohos/tcp.go`: TCP server and TCP client loopback demo.
- `go/ohos/orm.go`: GORM ORM demo backed by pure Go in-memory SQLite.
- `go/ohos/build_ohos.sh`: Builds `entry/libs/arm64-v8a/libohos.so`.
- `native/log_compat`: Project-local Android log compatibility library for Go's
  `GOOS=android` cgo path.
- `entry/libs/arm64-v8a`: Output location consumed by the HarmonyOS app.

## Build order

Use a Linux or WSL shell with Go, CMake, and the HarmonyOS native SDK available.

```bash
export OHOS_SDK=/path/to/ohos-sdk/linux
export OHOS_ARCH=arm64-v8a

./native/log_compat/build_ohos.sh
./go/ohos/build_ohos.sh
```

On this Windows setup, the equivalent scripts are:

```powershell
$env:OHOS_SDK = 'D:\harmSdk\command-line-tools\sdk\default\openharmony'
$env:OHOS_ARCH = 'arm64-v8a'

.\native\log_compat\build_ohos.ps1
.\go\ohos\build_ohos.ps1
```

Then build the HarmonyOS app from DevEco Studio or your usual hvigor command.

The app CMake file intentionally fails if `entry/libs/arm64-v8a/libohos.so` is
missing, because otherwise the UI could appear to work while not actually
loading the Go library.

## Notes

The script keeps the compatibility `android/log.h` and `liblog.so` inside the
project instead of copying files into the HarmonyOS SDK. That makes the build
more repeatable and avoids mutating a shared SDK installation.

Only `arm64-v8a` is wired for this POC. Add more architectures only after the
arm64 device path is confirmed.

## Gin POC

The Go library now embeds Gin and exposes an HTTP server on `0.0.0.0:18080`.
ArkTS can still call it through `127.0.0.1:18080`. The demo routes are:

```text
GET /ping -> {"framework":"gin","message":"pong from Go"}
GET /ws -> WebSocket echo endpoint
GET /opcua -> OPC UA client status
GET /opcua/server -> embedded OPC UA server status
POST /opcua/server/start -> start embedded OPC UA server
POST /opcua/server/stop -> stop embedded OPC UA server
GET /opcua/endpoints?endpoint=opc.tcp://<host>:<port> -> OPC UA endpoint list
GET /opcua/read?endpoint=opc.tcp://<host>:<port>&node=ns=2;s=Demo.Static.Scalar.Int32 -> OPC UA node value
GET /modbus -> Modbus TCP client/server status
GET /modbus/server -> embedded Modbus TCP server status
POST /modbus/server/start -> start embedded Modbus TCP server
POST /modbus/server/stop -> stop embedded Modbus TCP server
GET /modbus/read-holding?target=tcp://127.0.0.1:5020&unit=1&address=0&quantity=2 -> read holding registers
POST /modbus/write-single?target=tcp://127.0.0.1:5020&unit=1&address=10&value=4321 -> write one holding register
GET /orm -> GORM status and record count
POST /orm/init -> initialize GORM and migrate seed data
GET /orm/devices -> list device records
POST /orm/devices -> create a device record
```

The service is started from ArkTS through `testNapi.startServer()`. Keep a
matching `stopServer()` call available for lifecycle cleanup before this grows
past a POC.

Gin pulls in Go's `net/http` stack. The HarmonyOS build uses the `netgo` build
tag so Go uses its pure Go resolver instead of relying on platform resolver
details from the HarmonyOS sysroot.

The `/ws` route uses `github.com/gorilla/websocket` to upgrade the Gin request
into a WebSocket connection. Text messages are returned as `已发送<message>`;
binary messages are echoed with the same payload. External WebSocket tools
should connect to `ws://<device-ip>:18080/ws`.

## OPC UA POC

The Go library includes `github.com/gopcua/opcua` as a pure Go OPC UA client
and server.

The embedded OPC UA server is started from ArkTS through
`testNapi.startOpcuaServer()` and listens on:

```text
opc.tcp://<device-ip>:4840
```

It currently exposes an anonymous, no-security demo namespace named
`HarmonyGo` with these readable nodes:

```text
ns=1;s=Tag1
ns=1;s=Tag2
ns=1;s=Message
ns=1;s=Running
ns=1;s=Timestamp
```

The existing Gin server also exposes OPC UA helper routes:

```text
GET http://<device-ip>:18080/opcua
GET http://<device-ip>:18080/opcua/server
POST http://<device-ip>:18080/opcua/server/start
POST http://<device-ip>:18080/opcua/server/stop
GET http://<device-ip>:18080/opcua/endpoints?endpoint=opc.tcp://192.168.1.20:4840
GET http://<device-ip>:18080/opcua/read?endpoint=opc.tcp://192.168.1.20:4840&node=ns=2;s=Demo.Static.Scalar.Int32
```

The status routes confirm the OPC UA package and embedded server are wired in.
The endpoints route returns security modes and policies advertised by a target
server. The read route connects anonymously with `SecurityPolicy#None`, reads
one NodeId, and returns the value as JSON. Servers requiring username/password,
certificates, or encrypted policies need additional options before production
use.

## Modbus TCP POC

The Go library includes `github.com/aldas/go-modbus-client` for Modbus TCP
client calls and a small embedded Modbus TCP server for local testing. The
server starts from ArkTS through `testNapi.startModbusServer()` and listens on:

```text
tcp://<device-ip>:5020
```

The embedded server currently supports:

```text
FC3  Read Holding Registers
FC6  Write Single Register
FC16 Write Multiple Registers
```

Initial holding registers:

```text
0 -> 100
1 -> 200
2 -> 300
3 -> 400
4 -> 500
```

The existing Gin server also exposes helper routes:

```text
GET  http://<device-ip>:18080/modbus
GET  http://<device-ip>:18080/modbus/server
POST http://<device-ip>:18080/modbus/server/start
POST http://<device-ip>:18080/modbus/server/stop
GET  http://<device-ip>:18080/modbus/read-holding?target=tcp://127.0.0.1:5020&unit=1&address=0&quantity=2
POST http://<device-ip>:18080/modbus/write-single?target=tcp://127.0.0.1:5020&unit=1&address=10&value=4321
```

External Modbus clients should connect directly to `<device-ip>:5020`.

## GORM POC

The Go library includes `gorm.io/gorm` and the pure Go SQLite driver
`github.com/glebarez/sqlite`. This avoids `mattn/go-sqlite3` cgo dependencies
while proving that an ORM framework can run inside the HarmonyOS Go shared
library.

ArkTS can initialize the ORM through:

```text
testNapi.initOrm()
```

The app passes its HarmonyOS sandbox database path into Go:

```text
const context = getContext(this) as common.UIAbilityContext
testNapi.initOrm(`${context.databaseDir}/harmony_go_orm.db`)
```

This creates a real SQLite file in the app sandbox, so data survives normal app
background/foreground transitions and process restarts. Uninstalling or clearing
the app data still removes the database, which is the expected sandbox behavior.

Seeded records describe the existing protocol demos:

```text
Gin HTTP        -> http://127.0.0.1:18080/ping
Gin WebSocket   -> ws://127.0.0.1:18080/ws
OPC UA Demo     -> opc.tcp://127.0.0.1:4840
Modbus TCP Demo -> tcp://127.0.0.1:5020
```

The Gin server exposes:

```text
GET  http://<device-ip>:18080/orm
POST http://<device-ip>:18080/orm/init?path=/data/storage/el2/database/harmony_go_orm.db
GET  http://<device-ip>:18080/orm/devices
POST http://<device-ip>:18080/orm/devices?name=PLC-A&protocol=Modbus%20TCP&endpoint=tcp://127.0.0.1:5020
```

Use this as the compatibility proof for GORM. If no path is passed, the Go side
falls back to the old in-memory SQLite DSN for unit tests and quick local
experiments.

## TCP POC

The Go library also exposes a TCP echo demo on port `18081`:

```text
server listens on: 0.0.0.0:18081
client binds to: 127.0.0.1:18082
client sends: hello tcp
server replies: hello tcp
```

ArkTS starts the server through `testNapi.startTcpServer()`, starts the client
through `testNapi.startTcpClient()`, and then sends a message through
`testNapi.tcpSend(...)`. This proves both TCP server and TCP client code run
inside the Go shared library on HarmonyOS with separate fixed local ports.

The TCP server now echoes raw bytes back to any connected client. It does not
require a trailing newline, so external TCP tools can connect to the device IP
and receive an immediate echo of the bytes they send.
