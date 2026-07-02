# 数据库 + WebSocket 稳定性测试报告

- 更新时间: 2026-06-30T16:27:32+08:00
- 开始时间: 2026-06-30T16:22:27+08:00
- 停止原因: duration reached
- Base URL: `http://127.0.0.1:2778`
- WebSocket URL: `ws://127.0.0.1:2778/ws`
- Run ID: `20260630-162227`
- 测试 FType: `9000`
- 计划时长: 300 秒
- 每轮数据量: 100 行
- 每轮间隔: 5 秒

## 汇总

- 已开始轮次: 40
- 成功轮次: 30
- 失败轮次: 10
- 尝试写入行数: 4000
- 已验证一致行数: 4000
- HTTP/API 失败: 0
- WebSocket 失败: 0
- 数据比对失败: 0
- 最短轮次耗时: 0.070 秒
- 最长轮次耗时: 10.298 秒
- 平均轮次耗时: 2.663 秒
- 最近错误: TimeoutError('timed out')

## 最近步骤耗时

| 步骤 | 耗时秒 | 结果 | 说明 |
| --- | ---: | --- | --- |
| save_base_faults | 0.100 | OK |  |
| read_base_faults | 0.018 | OK |  |
| websocket_probe | 0.010 | OK |  |
| save_base_faults | 0.103 | OK |  |
| read_base_faults | 0.027 | OK |  |
| websocket_probe | 0.011 | OK |  |
| save_base_faults | 0.105 | OK |  |
| read_base_faults | 0.036 | OK |  |
| websocket_probe | 10.023 | FAIL | timed out |
| save_base_faults | 0.124 | OK |  |
| read_base_faults | 0.026 | OK |  |
| websocket_probe | 0.008 | OK |  |
| save_base_faults | 0.109 | OK |  |
| read_base_faults | 0.031 | OK |  |
| websocket_probe | 0.023 | OK |  |
| save_base_faults | 0.089 | OK |  |
| read_base_faults | 0.024 | OK |  |
| websocket_probe | 0.016 | OK |  |
| save_base_faults | 0.091 | OK |  |
| read_base_faults | 0.022 | OK |  |
| websocket_probe | 0.008 | OK |  |
| save_base_faults | 0.081 | OK |  |
| read_base_faults | 0.019 | OK |  |
| websocket_probe | 0.011 | OK |  |
| save_base_faults | 0.120 | OK |  |
| read_base_faults | 0.036 | OK |  |
| websocket_probe | 0.014 | OK |  |
| save_base_faults | 0.120 | OK |  |
| read_base_faults | 0.027 | OK |  |
| websocket_probe | 10.020 | FAIL | timed out |

## 判断口径

- 写入一致性: `/Api/Fault/SaveBaseFaults` 保存测试专用 `FType` 的完整集合后，立即用 `/Api/Fault/GetListBaseFaultInfo` 查回并逐字段比对。
- WebSocket 稳定性: 连接 `/ws`，等待 `ready`，发送 JSON `ping`，并按配置额外发送 `querySortLog` 或 `queryFruitInfo` 数据库查询探针。
- 事件流水: 见下方 JSONL 文件，可用于定位具体失败轮次。

事件流水文件: `reports\db_ws_stability\20260630-162227\events.jsonl`
