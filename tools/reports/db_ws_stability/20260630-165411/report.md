# 数据库 + WebSocket 稳定性测试报告

- 更新时间: 2026-06-30T17:08:06+08:00
- 开始时间: 2026-06-30T16:54:11+08:00
- 停止原因: running
- Base URL: `http://127.0.0.1:2778`
- WebSocket URL: `ws://127.0.0.1:2778/ws`
- Run ID: `20260630-165411`
- 测试 FType: `9000`
- 计划时长: 28800 秒
- 每轮数据量: 5000 行
- 每轮间隔: 60 秒

## 汇总

- 已开始轮次: 11
- 成功轮次: 10
- 失败轮次: 1
- 尝试写入行数: 55000
- 已验证一致行数: 50000
- HTTP/API 失败: 1
- WebSocket 失败: 0
- 数据比对失败: 0
- 最短轮次耗时: 0.033 秒
- 最长轮次耗时: 25.737 秒
- 平均轮次耗时: 20.993 秒
- 最近错误: /Api/Fault/SaveBaseFaults network error: [WinError 10054] 远程主机强迫关闭了一个现有的连接。

## 最近步骤耗时

| 步骤 | 耗时秒 | 结果 | 说明 |
| --- | ---: | --- | --- |
| read_base_faults | 0.524 | OK |  |
| websocket_probe | 0.037 | OK |  |
| save_base_faults | 25.175 | OK |  |
| read_base_faults | 0.532 | OK |  |
| websocket_probe | 0.016 | OK |  |
| save_base_faults | 23.712 | OK |  |
| read_base_faults | 0.541 | OK |  |
| websocket_probe | 0.032 | OK |  |
| save_base_faults | 23.401 | OK |  |
| read_base_faults | 0.261 | OK |  |
| websocket_probe | 0.030 | OK |  |
| save_base_faults | 24.408 | OK |  |
| read_base_faults | 0.498 | OK |  |
| websocket_probe | 0.012 | OK |  |
| save_base_faults | 24.451 | OK |  |
| read_base_faults | 0.431 | OK |  |
| websocket_probe | 0.033 | OK |  |
| save_base_faults | 24.394 | OK |  |
| read_base_faults | 0.568 | OK |  |
| websocket_probe | 0.030 | OK |  |
| save_base_faults | 19.082 | OK |  |
| read_base_faults | 0.320 | OK |  |
| websocket_probe | 0.019 | OK |  |
| save_base_faults | 19.278 | OK |  |
| read_base_faults | 0.354 | OK |  |
| websocket_probe | 0.043 | OK |  |
| save_base_faults | 17.818 | OK |  |
| read_base_faults | 0.300 | OK |  |
| websocket_probe | 0.020 | OK |  |
| save_base_faults | 0.032 | FAIL | /Api/Fault/SaveBaseFaults network error: [WinError 10054] 远程主机强迫关闭了一个现有的连接。 |

## 判断口径

- 写入一致性: `/Api/Fault/SaveBaseFaults` 保存测试专用 `FType` 的完整集合后，立即用 `/Api/Fault/GetListBaseFaultInfo` 查回并逐字段比对。
- WebSocket 稳定性: 连接 `/ws`，等待 `ready`，发送 JSON `ping`，并按配置额外发送 `querySortLog` 或 `queryFruitInfo` 数据库查询探针。
- 事件流水: 见下方 JSONL 文件，可用于定位具体失败轮次。

事件流水文件: `reports\db_ws_stability\20260630-165411\events.jsonl`
