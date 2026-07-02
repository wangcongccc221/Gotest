# 数据库 + WebSocket 稳定性测试报告

- 更新时间: 2026-06-30T17:07:03+08:00
- 开始时间: 2026-06-30T17:07:03+08:00
- 停止原因: startup failed
- Base URL: `http://127.0.0.1:2778`
- WebSocket URL: `ws://127.0.0.1:2778/ws`
- Run ID: `20260630-170703`
- 测试 CustomerID: `None`
- 计划时长: 28800 秒
- 每轮数据量: 50 行
- 每轮间隔: 60 秒

## 汇总

- 已开始轮次: 0
- 成功轮次: 0
- 失败轮次: 0
- 尝试写入行数: 0
- 已验证一致行数: 0
- HTTP/API 失败: 1
- WebSocket 失败: 0
- 数据比对失败: 0
- 最短轮次耗时: 0.000 秒
- 最长轮次耗时: 0.000 秒
- 平均轮次耗时: 0.000 秒
- 最近错误: /Api/SysFruitInfo/GetSysFruitInfo HTTP 404: 404 page not found

## 最近步骤耗时

| 步骤 | 耗时秒 | 结果 | 说明 |
| --- | ---: | --- | --- |
| readiness | 0.062 | OK |  |

## 判断口径

- 写入一致性: `/Api/SysFruitInfo/SaveSysFruitInfos` 保存测试专用 `CustomerID` 的完整集合后，立即用 `/Api/SysFruitInfo/GetSysFruitInfo` 查回并逐字段比对。
- WebSocket 稳定性: 连接 `/ws`，等待 `ready`，发送 JSON `ping`，并按配置额外发送 `querySortLog` 或 `queryFruitInfo` 数据库查询探针。
- 事件流水: 见下方 JSONL 文件，可用于定位具体失败轮次。

事件流水文件: `reports\db_ws_stability\20260630-170703\events.jsonl`
