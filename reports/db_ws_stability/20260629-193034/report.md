# 数据库 + WebSocket 稳定性测试报告

- 更新时间: 2026-06-29T19:30:34+08:00
- 开始时间: 2026-06-29T19:30:34+08:00
- 停止原因: once complete
- Base URL: `http://127.0.0.1:2778`
- WebSocket URL: `ws://127.0.0.1:2778/ws`
- Run ID: `20260629-193034`
- 测试 FType: `9000`
- 计划时长: 60 秒
- 每轮数据量: 10 行
- 每轮间隔: 1 秒

## 汇总

- 已开始轮次: 1
- 成功轮次: 0
- 失败轮次: 1
- 尝试写入行数: 10
- 已验证一致行数: 10
- HTTP/API 失败: 0
- WebSocket 失败: 1
- 数据比对失败: 0
- 最短轮次耗时: 0.000 秒
- 最长轮次耗时: 0.000 秒
- 平均轮次耗时: 0.000 秒
- 最近错误: websocket upgrade accepted with unexpected Sec-WebSocket-Accept

## 最近步骤耗时

| 步骤 | 耗时秒 | 结果 | 说明 |
| --- | ---: | --- | --- |
| readiness | 0.078 | OK |  |
| choose_empty_fault_type | 0.043 | OK |  |
| save_base_faults | 0.029 | OK |  |
| read_base_faults | 0.026 | OK |  |
| websocket_probe | 0.016 | FAIL | websocket upgrade accepted with unexpected Sec-WebSocket-Accept |

## 判断口径

- 写入一致性: `/Api/Fault/SaveBaseFaults` 保存测试专用 `FType` 的完整集合后，立即用 `/Api/Fault/GetListBaseFaultInfo` 查回并逐字段比对。
- WebSocket 稳定性: 连接 `/ws`，等待 `ready`，发送 JSON `ping`，并按配置额外发送 `querySortLog` 或 `queryFruitInfo` 数据库查询探针。
- 事件流水: 见下方 JSONL 文件，可用于定位具体失败轮次。

事件流水文件: `reports\db_ws_stability\20260629-193034\events.jsonl`
