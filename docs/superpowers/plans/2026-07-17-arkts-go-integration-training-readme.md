# ArkTS 与 Go 联调培训 README Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 基于 `E:\new\my_harmony` 前端与 `E:\gotest` 后端的当前代码，生成一份可直接拆成 40 分钟培训 PPT 的联调 README。

**Architecture:** 文档以两个独立 HarmonyOS bundle 的真实部署边界为起点，说明 ArkTS 前端如何通过 localhost HTTP/WebSocket 连接 Go 后端宿主，以及 Go 如何经 NAPI、动态库、TCP 和 GORM 完成设备通信与持久化。主案例采用“等级配置下发—设备回读—页面更新”的闭环，并用分层检查表解释数据不更新时的定位顺序。

**Tech Stack:** HarmonyOS ArkTS/ArkUI、NAPI/C++、Go c-shared、Gin、Gorilla WebSocket、TCP/CTCP、GORM、SQLite。

---

### Task 1: 固定文档结构与代码事实

**Files:**
- Modify: `E:\gotest\ArkTs+Go.md`

- [x] **Step 1: 写明受众、目标和 40 分钟边界**

  文档面向熟悉水果分选项目、具有五年以上开发经验、但没有鸿蒙开发经验的听众；不讲 ArkTS/Go 基础语法，也不展开 Qt/C# 对比。

- [x] **Step 2: 写明当前部署边界**

  明确前端 bundle `com.nutpi.My_Project` 与后端宿主 bundle `com.nutpi.gotest` 独立；前端当前代码不负责启动后端，只连接 `127.0.0.1:18080`。

- [x] **Step 3: 画出总体架构和启动时序**

  使用 Mermaid 展示 ArkTS UI、HTTP/WebSocket、Go 服务、NAPI/C++、`libohos.so`、TCP 下位机和 GORM/SQLite 的关系，并列出两侧 Ability 生命周期。

### Task 2: 写出两条真实联调链路

**Files:**
- Modify: `E:\gotest\ArkTs+Go.md`

- [x] **Step 1: 写等级配置主闭环**

  逐层引用 `LevelContent.handleSaveData`、`ConfigSender.sendFullGradeInfoTracked`、`HarmonyWebSocketClient.sendGradeInfoCommand`、Go `handleIncoming`、`SendGradeInfoData`、`StartCTCPClient`、`StGlobal` 回读与 `GlobalDataInterface.updateGlobalConfigFromDeviceEcho`。

- [x] **Step 2: 写 HTTP/GORM 辅助链路**

  说明 `LocalWebApiClient` 如何调用 `/Api/*`，Gin 如何注册路由，GORM 如何使用 SQLite WAL、迁移表结构并返回统一响应。

- [x] **Step 3: 区分四种成功语义**

  分开说明 WebSocket 发送成功、Go 处理成功、TCP 写入成功、设备回读生效，避免把任一层成功误认为完整闭环成功。

### Task 3: 写联调排错和 PPT 拆页建议

**Files:**
- Modify: `E:\gotest\ArkTs+Go.md`

- [x] **Step 1: 写六层排错检查表**

  检查顺序为：后端进程与端口 → WebSocket/HTTP → Go 命令处理 → TCP/设备 → 前端状态门控与渲染。

- [x] **Step 2: 收录当前代码中的典型“数据不更新”原因**

  包括后端未启动、WebSocket 重连窗口、`StGlobal` 未就绪、等级回包未同步、ACK 语义不完整、FSM 选择错误、消息字段或结构体不匹配、UI 监听/AppStorage 未触发。

- [x] **Step 3: 添加 40 分钟讲稿节奏与 PPT 页映射**

  控制主讲约 32～35 分钟，预留 5～8 分钟讨论；每节给出演示重点和建议截图/日志。

### Task 4: 验证文档

**Files:**
- Verify: `E:\gotest\ArkTs+Go.md`

- [x] **Step 1: 检查必需章节**

  Run: `rg -n "40 分钟|总体架构|生命周期|等级配置|HTTP|WebSocket|数据不更新|排错|优势|代价|PPT" E:\gotest\ArkTs+Go.md`

  Expected: 每个主题至少命中一次。

- [x] **Step 2: 核对关键代码符号仍存在**

  Run: `rg -n "sendFullGradeInfoTracked|sendGradeInfoCommand|updateGlobalConfigFromDeviceEcho" E:\new\my_harmony\entry\src\main\ets && rg -n "GoStartServer|SendGradeInfoData|StartCTCPClient|PublishWebSocketJSON" E:\gotest\go\ohos E:\gotest\entry\src\main\cpp`

  Expected: 前后端关键符号均有命中。

- [x] **Step 3: 检查范围与工作区差异**

  Run: `git -C E:\gotest diff -- ArkTs+Go.md docs/superpowers/plans/2026-07-17-arkts-go-integration-training-readme.md`

  Expected: 只包含培训 README 与本计划，不改动业务代码。
