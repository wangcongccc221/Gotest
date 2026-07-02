# 数据库 + WebSocket 稳定性测试报告

- 更新时间: 2026-06-30T17:08:28+08:00
- 开始时间: 2026-06-30T17:08:28+08:00
- 停止原因: once complete
- Base URL: `http://127.0.0.1:2778`
- WebSocket URL: `ws://127.0.0.1:2778/ws`
- Run ID: `20260630-170828`
- 测试 CustomerID: `9901`
- 计划时长: 28800 秒
- 每轮数据量: 50 行
- 每轮间隔: 60 秒

## 汇总

- 已开始轮次: 1
- 成功轮次: 0
- 失败轮次: 1
- 尝试写入行数: 50
- 已验证一致行数: 0
- HTTP/API 失败: 0
- WebSocket 失败: 0
- 数据比对失败: 2
- 最短轮次耗时: 0.075 秒
- 最长轮次耗时: 0.075 秒
- 平均轮次耗时: 0.075 秒
- 最近错误: database compare failed: {"ok": false, "expected_count": 50, "actual_count": 20, "missing_keys": [[9901, 21], [9901, 22], [9901, 23], [9901, 24], [9901, 25], [9901, 26], [9901, 27], [9901, 28], [9901, 29], [9901, 30], [9901, 31], [9901, 32], [9901, 33], [9901, 34], [9901, 35], [9901, 36], [9901, 37], [9901, 38], [9901, 39], [9901, 40], [9901, 41], [9901, 42], [9901, 43], [9901, 44], [9901, 45], [9901, 46], [9901, 47], [9901, 48], [9901, 49], [9901, 50]], "extra_keys": [], "mismatched_keys": [], "mismatch_details": [], "missing_count": 30, "extra_count": 0, "mismatched_count": 0}

## 最近步骤耗时

| 步骤 | 耗时秒 | 结果 | 说明 |
| --- | ---: | --- | --- |
| readiness | 0.054 | OK |  |
| save_sys_fruit_infos | 0.058 | OK |  |
| read_sys_fruit_infos | 0.016 | OK |  |

## 判断口径

- 写入一致性: `/Api/SysFruitInfo/SaveSysFruitInfos` 保存测试专用 `CustomerID` 的完整集合后，立即用 `/Api/SysFruitInfo/GetSysFruitInfo` 查回并逐字段比对。
- WebSocket 稳定性: 连接 `/ws`，等待 `ready`，发送 JSON `ping`，并按配置额外发送 `querySortLog` 或 `queryFruitInfo` 数据库查询探针。
- 事件流水: 见下方 JSONL 文件，可用于定位具体失败轮次。

事件流水文件: `reports\db_ws_stability\20260630-170828\events.jsonl`
