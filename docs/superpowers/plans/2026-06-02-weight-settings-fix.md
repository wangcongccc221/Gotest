# Weight Settings Fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the Harmony weight settings page match the 48 reference behavior for WAM/FSM commands, weight field mapping, calibration/zero semantics, and result feedback.

**Architecture:** Keep the UI page as the orchestrator, `ConfigSender.ets` as the command facade, `HarmonyWebSocketClient.ets` as the WebSocket transport, and Go as the CTCP command bridge. Fix command domain and field semantics before polishing UI behavior.

**Tech Stack:** ArkTS/Harmony UI, Go CTCP backend, 48 C++ reference.

---

## File Map

- `E:/new/my_harmony/entry/src/main/ets/components/dialogs/pages/WeightSettingsPage.ets`
  - Fix UI state mapping, calibration/zero behavior, apply payloads, cleanup on page leave.
- `E:/new/my_harmony/entry/src/main/ets/protocol/ConfigSender.ets`
  - Add/adjust FSM test cup commands and tracked WAM result handling.
- `E:/new/my_harmony/entry/src/main/ets/utils/network/HarmonyWebSocketClient.ets`
  - Add FSM test cup WebSocket messages and expose command ack results.
- `E:/goTest/go/ohos/Tcp/websocket.go`
  - Route test cup to FSM command, keep WAM logs, print receive/ack details.
- `E:/goTest/go/ohos/Tcp/ctcpclient.go`
  - Add/send FSM test cup command helpers if no existing helper covers it.
- `E:/goTest/go/ohos/Tcp/ctcp_structs.go`
  - Confirm `StGlobalWeightBaseInfo.WeightTh` matches the real L4/BL4 wire type.

---

## Task 1: Fix Test Cup Command Domain

**Why:** 48 sends test cup to FSM (`HC_CMD_TEST_CUP_ON/OFF`, dest `-1`), but Harmony currently sends WAM `0x0115/0x0116`.

**Files:**
- Modify: `E:/new/my_harmony/entry/src/main/ets/components/dialogs/pages/WeightSettingsPage.ets`
- Modify: `E:/new/my_harmony/entry/src/main/ets/protocol/ConfigSender.ets`
- Modify: `E:/new/my_harmony/entry/src/main/ets/utils/network/HarmonyWebSocketClient.ets`
- Modify: `E:/goTest/go/ohos/Tcp/websocket.go`
- Modify: `E:/goTest/go/ohos/Tcp/ctcpclient.go`

- [ ] Add frontend methods `sendFsmTestCupOn()` and `sendFsmTestCupOff()`.
- [ ] Change `handleTestCupChange()` to call FSM methods instead of WAM methods.
- [ ] Add WebSocket message types such as `fsmTestCupOn` and `fsmTestCupOff`.
- [ ] In Go, route those types through `handleSimpleFSMCommand(...)`.
- [ ] Verify logs show FSM test cup commands, not WAM test cup commands.

Expected log shape:

```text
WebSocket fsmTestCupOn: sending FSM cmd=HC_CMD_TEST_CUP_ON, dest=-1
WebSocket fsmTestCupOn success: ...
```

---

## Task 2: Separate Current Weight/Threshold From Wave Interval

**Why:** 48 displays `CurrentWeight` from `StGlobalWeightBaseInfo.RefWeight` and `Threshold` from `WeightTh`; `waveinterval[0/1]` is a separate wave parameter.

**Files:**
- Modify: `E:/new/my_harmony/entry/src/main/ets/components/dialogs/pages/WeightSettingsPage.ets`

- [ ] In `syncStateFromWeightSnapshot()`, keep:

```typescript
this.currentWeight = gweight.RefWeight
this.threshold = gweight.WeightTh
```

- [ ] Remove the later overwrite:

```typescript
this.currentWeight = weight.waveinterval[0]
this.threshold = weight.waveinterval[1]
```

- [ ] Add separate state for wave interval if the page still needs to edit it:

```typescript
@State waveIntervalStart: number = Number.NaN
@State waveIntervalEnd: number = Number.NaN
```

- [ ] Save `waveIntervalStart/End` only into `StWeightBaseInfo.waveinterval`.
- [ ] Save `currentWeight/threshold` only into `StGlobalWeightBaseInfo.RefWeight/WeightTh`.

---

## Task 3: Correct G-AD, Calibration, And Zero Field Semantics

**Why:** `fGADParam[0]` is AD0 G-AD coefficient, `fGADParam[1]` is AD1 G-AD coefficient, and zero baseline is runtime AD baseline, not a saved `fGADParam` field.

**Files:**
- Modify: `E:/new/my_harmony/entry/src/main/ets/components/dialogs/pages/WeightSettingsPage.ets`

- [ ] Rename UI state mentally/code-wise:

```typescript
@State gad0Coef: number = Number.NaN
@State gad1Coef: number = Number.NaN
@State standardAD0: number = 0
```

- [ ] Map readback:

```typescript
this.gad0Coef = Number(weight.fGADParam[0].toFixed(6))
this.gad1Coef = Number(weight.fGADParam[1].toFixed(6))
```

- [ ] For the current AD0-only panel, send:

```typescript
payload.fGADParam[0] = this.clampFloat(this.gad0Coef, -999999, 999999)
payload.fGADParam[1] = this.clampFloat(this.gad1Coef, -999999, 999999)
```

- [ ] `归零` should send `HC_WAM_CMD_RESET_AD` and update only `standardAD0` / display, not overwrite `fGADParam[1]`.
- [ ] `标定` should compute `gad0Coef = weightInput / (adValue - standardAD0)` and send `HC_WAM_CMD_WEIGHT_INFO`.
- [ ] Do not treat the right-side zero field as a persisted zero offset.

---

## Task 4: Make WAM Ack Meaningful

**Why:** Current page treats WebSocket send success as device execution success. Go already sends `commandAck`, but the Harmony client only logs it.

**Files:**
- Modify: `E:/new/my_harmony/entry/src/main/ets/utils/network/HarmonyWebSocketClient.ets`
- Modify: `E:/new/my_harmony/entry/src/main/ets/protocol/ConfigSender.ets`
- Modify: `E:/new/my_harmony/entry/src/main/ets/components/dialogs/pages/WeightSettingsPage.ets`

- [ ] Store pending command promises keyed by `topic + cmdId + destId`.
- [ ] Resolve pending promises in `handleCommandAck()`.
- [ ] Return ack result from `sendWamWeightInfoTracked()` and `sendWamGlobalWeightInfoTracked()`.
- [ ] Update local cache only when ack `ok === true`.
- [ ] If ack `ok === false`, show failure and do not overwrite UI cache.

Expected behavior:

```text
result=0  -> 页面显示成功并触发回读
result=-1 -> 页面显示失败，不写本地缓存
```

---

## Task 5: Fix Page Leave Cleanup

**Why:** 48 leaves the weight page by turning off internal signal, test cup, and data tracking. Harmony currently only removes listeners.

**Files:**
- Modify: `E:/new/my_harmony/entry/src/main/ets/components/dialogs/pages/WeightSettingsPage.ets`

- [ ] In `aboutToDisappear()`, if internal signal is on, send simulated pulse off.
- [ ] If test cup is on, send FSM test cup off.
- [ ] If data tracking is on, send WAM data tracking off for the selected channel.
- [ ] Reset local toggles after sending off commands.

---

## Task 6: Confirm And Fix `WeightTh` Wire Type

**Why:** 48 uses `ushort WeightTh` under `BL4`, but Go currently uses `uint8`. If the real machine is BL4/L4-compatible with ushort, values above 255 will be truncated.

**Files:**
- Modify if needed: `E:/goTest/go/ohos/Tcp/ctcp_structs.go`
- Modify if needed: `E:/new/my_harmony/entry/src/main/ets/components/dialogs/pages/WeightSettingsPage.ets`

- [ ] Confirm the real macro branch for this machine.
- [ ] If it is `ushort`, change Go `WeightTh` to `uint16`.
- [ ] Change Harmony clamp from `0..255` to `0..65535`.
- [ ] Verify Go log prints the exact threshold sent from Harmony.

---

## Task 7: Multi-Subsystem Global Weight Behavior

**Why:** 48 updates/mirrors `nTotalCupNums` and sends global weight info to every WAM subsystem. Harmony currently sends one selected WAM payload and fills all totals with one value.

**Files:**
- Modify: `E:/new/my_harmony/entry/src/main/ets/components/dialogs/pages/WeightSettingsPage.ets`
- Modify if needed: `E:/new/my_harmony/entry/src/main/ets/protocol/ConfigSender.ets`

- [ ] Keep single-subsystem behavior unchanged.
- [ ] If `nSubsysNum > 1`, mirror `nTotalCupNums` like 48.
- [ ] Send global weight info to every WAM subsystem, not only the selected WAM.

---

## Execution Order

1. Test cup command domain.
2. Current weight/threshold vs wave interval split.
3. G-AD/标定/归零 semantics.
4. Ack handling.
5. Page leave cleanup.
6. `WeightTh` wire type confirmation/fix.
7. Multi-subsystem behavior.

## Verification Notes

- Do not run compile/build unless the user explicitly asks.
- Use Go/Harmony logs to verify command type, destination, payload fields, and readback.
- After each task, compare against the 48 reference function named in the task.
