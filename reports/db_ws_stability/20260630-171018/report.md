# 数据库 + WebSocket 稳定性测试报告

- 更新时间: 2026-06-30T17:12:19+08:00
- 开始时间: 2026-06-30T17:10:18+08:00
- 停止原因: stopped after 5 consecutive failures
- Base URL: `http://127.0.0.1:2778`
- WebSocket URL: `ws://127.0.0.1:2778/ws`
- Run ID: `20260630-171018`
- 测试 CustomerID: `9000`
- 计划时长: 3600 秒
- 每轮数据量: 1000 行
- 每轮间隔: 30 秒

## 汇总

- 已开始轮次: 5
- 成功轮次: 0
- 失败轮次: 5
- 尝试写入行数: 5000
- 已验证一致行数: 0
- HTTP/API 失败: 0
- WebSocket 失败: 0
- 数据比对失败: 10
- 最短轮次耗时: 0.074 秒
- 最长轮次耗时: 0.160 秒
- 平均轮次耗时: 0.119 秒
- 最近错误: database compare failed: {"ok": false, "expected_count": 1000, "actual_count": 20, "missing_keys": [[9000, 21], [9000, 22], [9000, 23], [9000, 24], [9000, 25], [9000, 26], [9000, 27], [9000, 28], [9000, 29], [9000, 30], [9000, 31], [9000, 32], [9000, 33], [9000, 34], [9000, 35], [9000, 36], [9000, 37], [9000, 38], [9000, 39], [9000, 40], [9000, 41], [9000, 42], [9000, 43], [9000, 44], [9000, 45], [9000, 46], [9000, 47], [9000, 48], [9000, 49], [9000, 50], [9000, 51], [9000, 52], [9000, 53], [9000, 54], [9000, 55], [9000, 56], [9000, 57], [9000, 58], [9000, 59], [9000, 60], [9000, 61], [9000, 62], [9000, 63], [9000, 64], [9000, 65], [9000, 66], [9000, 67], [9000, 68], [9000, 69], [9000, 70]], "extra_keys": [], "mismatched_keys": [], "mismatch_details": [], "missing_count": 980, "extra_count": 0, "mismatched_count": 0}

## 最近步骤耗时

| 步骤 | 耗时秒 | 结果 | 说明 |
| --- | ---: | --- | --- |
| readiness | 0.054 | OK |  |
| choose_empty_customer_id | 0.012 | OK |  |
| save_sys_fruit_infos | 0.131 | OK |  |
| read_sys_fruit_infos | 0.027 | OK |  |
| save_sys_fruit_infos | 0.097 | OK |  |
| read_sys_fruit_infos | 0.018 | OK |  |
| save_sys_fruit_infos | 0.096 | OK |  |
| read_sys_fruit_infos | 0.011 | OK |  |
| save_sys_fruit_infos | 0.058 | OK |  |
| read_sys_fruit_infos | 0.012 | OK |  |
| save_sys_fruit_infos | 0.105 | OK |  |
| read_sys_fruit_infos | 0.022 | OK |  |

## 判断口径

- 写入一致性: `/Api/SysFruitInfo/SaveSysFruitInfos` 保存测试专用 `CustomerID` 的完整集合后，立即用 `/Api/SysFruitInfo/GetSysFruitInfo` 查回并逐字段比对。
- WebSocket 稳定性: 连接 `/ws`，等待 `ready`，发送 JSON `ping`，并按配置额外发送 `querySortLog` 或 `queryFruitInfo` 数据库查询探针。
- 事件流水: 见下方 JSONL 文件，可用于定位具体失败轮次。

事件流水文件: `reports\db_ws_stability\20260630-171018\events.jsonl`
