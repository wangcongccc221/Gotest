# Repository Guidelines

## Project Structure & Module Organization

```
entry/src/main/ets/pages/   - ArkTS/HarmonyOS frontend (Index.ets entry point)
entry/src/main/cpp/          - C++ NAPI bridge (napi_init.cpp -> libohos.so)
go/ohos/                     - Go backend (compiles to libohos.so)
  main.go                    - C ABI exports (//export GoStartServer, etc.)
  Tcp/                       - CTCP server, WebSocket hub, data parsing
  database/                  - GORM models, migrations, REST routes
  modbus/                    - Modbus TCP server
  opcua/                     - OPC UA server
docs/                        - Architecture & field-mapping docs
tools/                       - Verification scripts (.mjs)
entry/src/ohosTest/          - HarmonyOS instrumented tests
```

Call chain: ArkTS -> C++ NAPI -> Go exports -> Go modules.

## Build, Test, and Development Commands

| Command | Purpose |
|---|---|
| `.\go\ohos\build_ohos.ps1` | Rebuild Go -> libohos.so (required after any Go change) |
| DevEco Studio build | Build full HarmonyOS app (links libohos.so via CMake) |
| `cd go\ohos; go test ./...` | Run Go unit tests |
| `python test_local_http.py` | Integration test against local HTTP API |

After adding a new Go export: rebuild with build_ohos.ps1, update napi_init.cpp, add to napi_property_descriptor array, call from ArkTS via libentry.so.

## Coding Style & Naming Conventions

- **Go**: gofmt formatting; exported functions use `//export GoFunctionName` prefix; table models follow `tb_*.go` naming.
- **ArkTS/ETS**: Follow HarmonyOS style; pages go in `entry/src/main/ets/pages/`.
- **C++**: clang-tidy checks configured in .clang-tidy; includes via include/ directory.
- Commit prefixes: `feat:`, `fix:`, `chore:`, `docs:` -- mixed Chinese/English body text is fine.

## Testing Guidelines

- **Go tests**: Place alongside source (e.g., `database/fault_api_test.go`); run with `go test ./...`.
- **Python tests**: Root-level `test_*.py` files for HTTP integration checks.
- **HarmonyOS tests**: `entry/src/ohosTest/ets/test/` -- use @ohos/hypium framework.

## Commit & Pull Request Guidelines

- Use conventional commit prefixes: `feat:`, `fix:`, `chore:`, `docs:`.
- Keep messages short and descriptive; Chinese is acceptable.
- Before merging: run `go test ./...`, rebuild libohos.so, verify the app deploys without crashes.

## Agent-Specific Instructions

- CodeGraph is available (`.codegraph/` exists) -- use `codegraph explore` before grep/find for code navigation.
- When fixing bugs: reproduce first, identify root cause, state minimal change scope, then verify no regressions.
- **Memory**: C strings returned to ArkTS must be freed via `GoFreeCString()`.
- **TLS**: Go targets GOOS=android with a HarmonyOS TLS shim; do not remove `ohos_android_tls_wrap.go`.
