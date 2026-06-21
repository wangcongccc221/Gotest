# claude 帮我解释一下代码：setExitConfig(fsm: FsmKey, config:
_Exported on 06/21/2026 at 09:05:51 GMT+8 from Claude via WayLog_


**User**

claude 帮我解释一下代码：setExitConfig(fsm: FsmKey, config: OutletConfig, markManualEnabled: boolean = false): void {
    const index = config.exitIndex < 1 ? 1 : config.exitIndex
    const array = fsm === 'FSM1' ? this.configsFSM1.slice() : this.configsFSM2.slice()
    const pos = array.findIndex((item: OutletConfig) => item.exitIndex === index)
    const merged = this.normalizeConfig(config, index)
    if (pos >= 0) {
      array[pos] = merged
    } else {
      array.push(merged)
    }
    if (fsm === 'FSM1') {
      this.configsFSM1 = array
    } else {
      this.configsFSM2 = array
    }
    this.saveToStorage(fsm)
    if (markManualEnabled) {
      this.setManualExitEnabledValue(fsm, index, merged.enabled && merged.outletEnable === '开启')
    }
  }

---


**Claude**

API Error: Request rejected (429) · Service Unavailable

---
