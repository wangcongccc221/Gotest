# 鸿蒙 Go 后端数据库 + WebSocket 稳定性测试

本文档记录 `tools/db_ws_stability_test.py` 的用途、运行方法和报告解读方式。它用于晚上长时间压测电脑通过 `hdc fport` 访问鸿蒙设备里的 Go 后端，重点观察 SQLite 写入/查回一致性和 WebSocket 长连接稳定性。

## 测试前准备

1. 确认鸿蒙应用里的 Go 后端已经启动。
2. 在电脑命令行执行端口转发：

```powershell
hdc fport tcp:2778 tcp:18080
```

看到下面结果说明转发成功：

```text
Forwardport result:OK
```

3. 在仓库根目录运行测试脚本。

```powershell
cd E:\goTest
python tools\db_ws_stability_test.py --once --batch-size 100
```

如果这条短测能生成报告，再跑通宵测试。

## 推荐命令

先跑一轮短测，确认端口转发、ORM、WebSocket 都正常：

```powershell
python tools\db_ws_stability_test.py --once --batch-size 100
```

然后建议用 1000 行跑 12 小时。这个量更适合先看长期稳定性，不容易一开始就把 SQLite 写锁打满：

```powershell
python tools\db_ws_stability_test.py --duration-seconds 43200 --batch-size 1000 --interval-seconds 60
```

如果 1000 行连续几小时都没有失败，再逐步加压到 3000 或 5000：

```powershell
python tools\db_ws_stability_test.py --duration-seconds 43200 --batch-size 3000 --interval-seconds 90
python tools\db_ws_stability_test.py --duration-seconds 43200 --batch-size 5000 --interval-seconds 120
```

`10000` 行属于极限压力测试。上来直接跑 10000 行时，`/Api/Fault/SaveBaseFaults` 可能超过 30 秒 HTTP 超时；后端事务还没结束，下一轮继续写就容易触发 SQLite `database is locked (SQLITE_BUSY)`。需要测 10000 行时，建议提高超时和间隔：

```powershell
python tools\db_ws_stability_test.py --duration-seconds 43200 --batch-size 10000 --interval-seconds 300 --http-timeout 120
```

## 输出文件

每次运行会生成一个带时间戳的目录：

```text
reports\db_ws_stability\<run-id>\
```

里面有三类文件：

- `report.md`: 人看的 Markdown 报告，建议明天优先看这个。
- `summary.json`: 机器可读汇总，包括成功轮次、失败轮次、错误数、耗时。
- `events.jsonl`: 每一步事件流水，一行一个 JSON，适合定位第几轮失败。

脚本每轮都会刷新这些文件。即使中途 Ctrl+C 或出错，已经完成的结果也会保存。

脚本遇到 SQLite busy 或 HTTP timeout 时会自动退避一段时间，默认连续失败 5 次后停止，并在 `停止原因` 中写明原因，避免整晚都在重复失败。

## 测试做了什么

### HTTP + SQLite 一致性

脚本会：

1. 请求 `/ping`，确认 Go HTTP 服务可访问。
2. 请求 `/orm`，确认 SQLite ORM 已初始化且状态为 `ready`。
3. 自动在 `9000..9999` 中找一个空的测试专用 `FType`。
4. 构造大量 `tb_Base_Fault` 测试数据。
5. 调用 `/Api/Fault/SaveBaseFaults` 写入完整集合。
6. 调用 `/Api/Fault/GetListBaseFaultInfo` 查回。
7. 按 `FCode` 对齐，比对 `FType`、`FName`、`FGroupName`、`FMessage`、条件字段、出口、时长等字段是否完全一致。
8. 结束时删除本次测试写入的数据。

注意：`SaveBaseFaults` 对同一个 `FType` 是整组替换，所以脚本默认只选择空的高位测试类型，避免覆盖业务数据。

### WebSocket 稳定性

默认 WebSocket URL 是：

```text
ws://127.0.0.1:2778/ws
```

每轮会：

1. 建立 WebSocket 连接。
2. 等待服务端返回 `{"type":"ready"}`。
3. 发送 JSON 心跳 `{"type":"ping","requestId":"..."}`。
4. 等待同 requestId 的 `pong`。
5. 默认发送 `querySortLog`，通过 WebSocket 走一次数据库查询探针。

可选模式：

- `--ws-mode sort_log`: 默认，WebSocket 查询分选运行日志。
- `--ws-mode fruit_info`: WebSocket 查询果品分页数据。
- `--ws-mode ping`: 只测 WebSocket 连接和心跳。
- `--ws-mode off`: 不测 WebSocket。

当前后端没有故障基础表的 WebSocket 查询类型，所以“大量写入后精确查回一致性”走 HTTP 数据库 API；WebSocket 负责验证同一端口上的长连接、心跳和已有数据库查询 API 是否稳定。

## 明天怎么看结果

打开最新目录里的 `report.md`，重点看：

- `成功轮次` 是否持续增长。
- `失败轮次` 是否为 0。
- `尝试写入行数` 与 `已验证一致行数` 是否相等。
- `HTTP/API 失败` 是否为 0。
- `WebSocket 失败` 是否为 0。
- `数据比对失败` 是否为 0。
- `最近错误` 是否为 `无`。

如果出现失败，再看 `events.jsonl` 的最后几十行。常见判断：

- `readiness` 失败：Go 后端没启动、`hdc fport` 断了、或者 ORM 没初始化。
- `save_base_faults` 失败：SQLite 写入/API 处理失败。
- `database is locked (SQLITE_BUSY)`：SQLite 正被长事务占用，通常是批量太大、HTTP 超时后继续发下一轮、或设备端还有其他写库任务。
- `read_base_faults` 失败：写入后查询失败。
- `database compare failed`：查回的数据和脚本写入的数据不一致。
- `websocket_probe` 失败：WebSocket 建连、心跳或查询探针失败。

## 常用参数

```powershell
# 短测一轮
python tools\db_ws_stability_test.py --once --batch-size 100

# 12 小时，每轮 1000 行，推荐先用这个看稳定性
python tools\db_ws_stability_test.py --duration-seconds 43200 --batch-size 1000 --interval-seconds 60

# 12 小时，每轮 5000 行，确认 1000/3000 稳定后再用
python tools\db_ws_stability_test.py --duration-seconds 43200 --batch-size 5000 --interval-seconds 120

# 极限测试 10000 行，提高 HTTP 超时和轮次间隔
python tools\db_ws_stability_test.py --duration-seconds 43200 --batch-size 10000 --interval-seconds 300 --http-timeout 120

# 只测 HTTP + SQLite，不测 WebSocket
python tools\db_ws_stability_test.py --ws-mode off

# WebSocket 改成果品数据查询探针
python tools\db_ws_stability_test.py --ws-mode fruit_info

# 只测 WebSocket 心跳
python tools\db_ws_stability_test.py --ws-mode ping

# SQLite busy/timeout 后多等一会儿再重试
python tools\db_ws_stability_test.py --batch-size 5000 --failure-backoff-seconds 300
```

## 注意事项

- 测试期间不要断开 USB、不要关闭鸿蒙应用。
- Windows 电脑不要睡眠，否则端口转发和长跑脚本都会中断。
- 默认脚本结束会清理测试数据。需要保留现场时加 `--no-cleanup`。
- 如果 Ctrl+C，中断前已完成的报告仍然保存在 `reports\db_ws_stability\<run-id>\`。
