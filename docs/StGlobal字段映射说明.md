# StGlobal 字段映射说明

本文用于确认 `FSM_CMD_CONFIG(0x1000)` 返回的 `StGlobal` 每块数据在旧 48、Go 后端、鸿蒙前端中的对应关系，方便后续补齐字段映射。

## 数据链路

1. 前端 WebSocket 连接后发送 `{"type":"requestStGlobal"}`。
2. Go 后端调用 `RequestStGlobalFromDefaultFSM()`，向默认 FSM 请求全局配置。
3. FSM 返回 `FSM_CMD_CONFIG(0x1000)`，payload 是 `StGlobal`。
4. Go 后端在 `ctcpserver.go` 中执行 `ParseData[StGlobal](payload)`。
5. Go 后端把 `StGlobal` 转成大写字段 JSON，通过 WebSocket `topic=stglobal` 推给鸿蒙。
6. 鸿蒙 `HarmonyWebSocketClient` 收到后调用 `StGlobalJsonMapper.applyStGlobalJsonToStruct()`，再写入 `GlobalDataInterface.updateGlobalConfig()`。

参考位置：

- C++ 定义：`E:\new\48\RSS\Base\interface.h`
- Go 定义：`go/ohos/Tcp/ctcp_structs.go`
- Go 接收：`go/ohos/Tcp/ctcpserver.go`
- Go JSON：`go/ohos/Tcp/ctcp_stglobal_to_json.go`
- 鸿蒙 JSON 类型：`E:\new\my_harmony\entry\src\main\ets\protocol\StGlobalJson.ets`
- 鸿蒙映射：`E:\new\my_harmony\entry\src\main\ets\protocol\StGlobalJsonMapper.ets`
- 鸿蒙运行态：`E:\new\my_harmony\entry\src\main\ets\protocol\GlobalDataInterface.ets`

## 顶层字段

| C++ `StGlobal` | Go `StGlobal` | WebSocket JSON | 鸿蒙结构 | 当前鸿蒙映射状态 | 主要用途 |
| --- | --- | --- | --- | --- | --- |
| `sys` | `Sys` | `Sys` | `StGlobal.sys` | 已映射 | 系统结构、出口数、子系统数、相机/重量/内部品质能力开关 |
| `grade` | `Grade` | `Grade` | `StGlobal.grade` | 未映射 | 等级表、品质表、拖拽出口映射、贴标、统计等级名称 |
| `gexit` | `GExit` | `GExit` | `StGlobal.gexit` | 未映射 | 全局出口脉宽、贴标脉宽、出口驱动脚、延时/保持时间 |
| `gweight` | `GWeight` | `GWeight` | 暂无对应字段 | 未映射 | 全局称重配置；48 代码里该字段在当前片段中被注释，但 Go 结构里保留 |
| `analogdensity` | `AnalogDensity` | `AnalogDensity` | `StGlobal.analogdensity` | 未映射 | 模拟密度配置 |
| `exit[12]` | `Exit` | `Exit` | `StGlobal.exit[]` | 未映射 | 每通道出口距离、贴标距离、驱动脚 |
| `paras[12]` | `Paras` | `Paras` | `StGlobal.paras[]` | 未映射 | IPM/相机参数、杯数、NIR 参数 |
| `weights[12]` | `Weights` | `Weights` | 暂无对应字段 | 未映射 | 通道称重参数；48 代码里当前片段被注释，Go 结构里保留 |
| `motor[48]` | `Motor` | `Motor` | `StGlobal.motor[]` | 未映射 | 出口电机配置 |
| `cFSMInfo[12]` | `CFSMInfo` | `CFSMInfo` | `StGlobal.cFSMInfo` | 已映射 | FSM 编译日期/版本文本 |
| `cIPMInfo[12]` | `CIPMInfo` | `CIPMInfo` | `StGlobal.cIPMInfo` | 已映射 | IPM 编译日期/版本文本 |
| `nSubsysId` | `NSubsysId` | `NSubsysId` | `StGlobal.nSubsysId` | 已映射 | 子系统/FSM ID |
| `nVersion` | `NVersion` | `NVersion` | `StGlobal.nVersion` | 已映射 | 配置版本 |
| `nNetState` | 当前 Go 未导出 | `NNetState?` | `StGlobal.nNetState` | JSON 预留但 Go 未输出 | IPM 网络状态 |
| `nFsmRestart` | 当前 Go 未导出 | `NFsmRestart?` | `StGlobal.nFsmRestart` | JSON 预留但 Go 未输出 | FSM 重启标记 |
| `nFsmModule` | 当前 Go 未导出 | `NFsmModule?` | `StGlobal.nFsmModule` | JSON 预留但 Go 未输出 | FSM 模块类型 |

当前最大缺口：`StGlobalJsonMapper` 只映射了 `Sys`、版本/ID、`CFSMInfo`、`CIPMInfo`，没有把 `Grade/GExit/AnalogDensity/Exit/Paras/Motor` 写进鸿蒙 `StGlobal`。所以前端如果只依赖 `topic=stglobal`，等级表和出口参数会丢在 JSON 对象里，没有进入运行态。

## Sys 字段

`Sys` 对应 C++ `StSysConfig`，Go `StSysConfig`，鸿蒙 `StSysConfig`。

| Go JSON 字段 | 鸿蒙字段 | 用途 |
| --- | --- | --- |
| `ExitState` | `sys.exitstate` | 出口/系统结构状态位 |
| `NChannelInfo` | `sys.nChannelInfo` | 通道信息 |
| `NImageUV` | `sys.nImageUV` | UV 图像功能配置 |
| `NDataRegistration` | `sys.nDataRegistration` | 数据注册/通道配置 |
| `NImageSugar` | `sys.nImageSugar` | 糖度图像/内部品质相关配置 |
| `NImageUltrasonic` | `sys.nImageUltrasonic` | 超声相关配置 |
| `NCameraDelay` | `sys.nCameraDelay` | 相机延时 |
| `Width` / `Height` | `sys.width` / `sys.height` | 图像宽高 |
| `PacketSize` | `sys.packetSize` | 包大小 |
| `NSystemInfo` | `sys.nSystemInfo` | 系统信息位 |
| `NSubsysNum` | `sys.nSubsysNum` | 子系统数量 |
| `NExitNum` | `sys.nExitNum` | 出口数量 |
| `NClassificationInfo` | `sys.nClassificationInfo` | 分级能力/分类信息 |
| `MultiFreq` | `sys.multiFreq` | 多频配置 |
| `NCameraType` | `sys.nCameraType` | 相机类型 |
| `CIRClassifyType` | `sys.CIRClassifyType` | 视觉/尺寸能力类型 |
| `UVClassifyType` | `sys.UVClassifyType` | UV 能力类型 |
| `WeightClassifyTpye` | `sys.WeightClassifyTpye` | 重量能力类型 |
| `InternalClassifyType` | `sys.InternalClassifyType` | 内部品质能力类型 |
| `UltrasonicClassifyType` | `sys.UltrasonicClassifyType` | 超声能力类型 |
| `IfWIFIEnable` | `sys.IfWIFIEnable` | Wi-Fi 开关 |
| `CheckExit` | `sys.CheckExit` | 抽检出口 |
| `CheckNum` | `sys.CheckNum` | 抽检数量 |
| `NIQSEnable` | `sys.nIQSEnable` | IQS 开关 |

注意：`Sys.*ClassifyType` 是系统能力/硬件功能开关，不是等级表的分选标准。等级表使用 `Grade.NClassifyType`。

## Grade 字段

`Grade` 对应 C++ `StGradeInfo`，Go `StGradeInfo`，鸿蒙 `StGradeInfo`。这是首页等级表、品质表和拖拽出口映射最重要的一块。

| Go JSON 字段 | 鸿蒙字段 | 用途 |
| --- | --- | --- |
| `Intervals[3]` | `grade.intervals[]` | 颜色区间 |
| `Percent[48]` | `grade.percent[]` | 颜色百分比等级范围 |
| `Grades[256]` | `grade.grades[]` | 等级明细矩阵，核心数据 |
| `ExitEnabled[2]` | `grade.ExitEnabled` | 48 出口启用位图 |
| `ColorIntervals[2]` | `grade.ColorIntervals` | 颜色滑块/颜色区间值 |
| `NExitSwitchNum[48]` | `grade.nExitSwitchNum` | 出口切换数量 |
| `NTagInfo[6]` | `grade.nTagInfo` | 标签信息 |
| `NFruitType` | `grade.nFruitType` | 水果类型 |
| `StrFruitName[50]` | `grade.strFruitName` | 水果名称 |
| `UnFlawAreaFactor[12]` | `grade.unFlawAreaFactor` | 瑕疵面积/数量因子 |
| `UnBruiseFactor[12]` | `grade.unBruiseFactor` | 碰伤因子 |
| `UnRotFactor[12]` | `grade.unRotFactor` | 腐烂因子 |
| `FDensityFactor[6]` | `grade.fDensityFactor` | 密度因子 |
| `FSugarFactor[6]` | `grade.fSugarFactor` | 糖度因子 |
| `FAcidityFactor[6]` | `grade.fAcidityFactor` | 酸度因子 |
| `FHollowFactor[6]` | `grade.fHollowFactor` | 空心因子 |
| `FSkinFactor[6]` | `grade.fSkinFactor` | 浮皮因子 |
| `FBrownFactor[6]` | `grade.fBrownFactor` | 褐变因子 |
| `FTangxinFactor[6]` | `grade.fTangxinFactor` | 糖心因子 |
| `FRigidityFactor[6]` | `grade.fRigidityFactor` | 硬度因子 |
| `FWaterFactor[6]` | `grade.fWaterFactor` | 水心/水分因子 |
| `FShapeFactor[6]` | `grade.fShapeFactor` | 形状因子 |
| `StrSizeGradeName[192]` | `grade.strSizeGradeName` | 尺寸/重量等级名称，16 个，每个 12 字节 |
| `StrQualityGradeName[192]` | `grade.strQualityGradeName` | 品质等级名称，16 个，每个 12 字节 |
| `StDensityGradeName[72]` | `grade.stDensityGradeName` | 密度等级名称 |
| `StrColorGradeName[192]` | `grade.strColorGradeName` | 颜色等级名称 |
| `StrShapeGradeName[72]` | `grade.strShapeGradeName` | 形状等级名称 |
| `StFlawareaGradeName[72]` | `grade.stFlawareaGradeName` | 瑕疵等级名称 |
| `StBruiseGradeName[72]` | `grade.stBruiseGradeName` | 碰伤等级名称 |
| `StRotGradeName[72]` | `grade.stRotGradeName` | 腐烂等级名称 |
| `StSugarGradeName[72]` | `grade.stSugarGradeName` | 糖度等级名称 |
| `StAcidityGradeName[72]` | `grade.stAcidityGradeName` | 酸度等级名称 |
| `StHollowGradeName[72]` | `grade.stHollowGradeName` | 空心等级名称 |
| `StSkinGradeName[72]` | `grade.stSkinGradeName` | 浮皮等级名称 |
| `StBrownGradeName[72]` | `grade.stBrownGradeName` | 褐变等级名称 |
| `StTangxinGradeName[72]` | `grade.stTangxinGradeName` | 糖心等级名称 |
| `StRigidityGradeName[72]` | `grade.stRigidityGradeName` | 硬度等级名称 |
| `StWaterGradeName[72]` | `grade.stWaterGradeName` | 水心/水分等级名称 |
| `ColorType` | `grade.ColorType` | 颜色分级方式 |
| `NLabelType` | `grade.nLabelType` | 贴标方式 |
| `NLabelbyExit[48]` | `grade.nLabelbyExit` | 按出口贴标配置 |
| `NSwitchLabel[48]` | `grade.nSwitchLabel` | 出口切换贴标方式 |
| `NSizeGradeNum` | `grade.nSizeGradeNum` | 尺寸/重量等级数量 |
| `NQualityGradeNum` | `grade.nQualityGradeNum` | 品质等级数量 |
| `NClassifyType` | `grade.nClassifyType` | 分选标准/显示维度辅助字段 |
| `NCheckNum` | `grade.nCheckNum` | 抽检数 |
| `ForceChannel` | `grade.ForceChannel` | 强制通道 |

### Grades[256] 明细

| Go JSON 字段 | 鸿蒙字段 | 用途 |
| --- | --- | --- |
| `ExitLow` / `ExitHigh` | `grade.grades[i].exitLow` / `exitHigh` | 64 位出口映射位图，拖拽等级到出口主要改这里 |
| `NMinSize` / `NMaxSize` | `nMinSize` / `nMaxSize` | 等级范围；尺寸/重量共用 |
| `NFruitNum` | `nFruitNum` | 装箱个数/重量 |
| `NColorGrade` | `nColorGrade` | 颜色等级 |
| `SbShapeSize` | `sbShapeSize` | 形状等级 |
| `SbDensity` | `sbDensity` | 密度等级 |
| `SbFlawArea` | `sbFlawArea` | 瑕疵等级 |
| `SbBruise` | `sbBruise` | 碰伤等级 |
| `SbRot` | `sbRot` | 腐烂等级 |
| `SbSugar` | `sbSugar` | 糖度等级 |
| `SbAcidity` | `sbAcidity` | 酸度等级 |
| `SbHollow` | `sbHollow` | 空心等级 |
| `SbSkin` | `sbSkin` | 浮皮等级 |
| `SbBrown` | `sbBrown` | 褐变等级 |
| `SbTangxin` | `sbTangxin` | 糖心等级 |
| `SbRigidity` | `sbRigidity` | 硬度等级 |
| `SbWater` | `sbWater` | 水心/水分等级 |
| `SbLabelbyGrade` | `sbLabelbyGrade` | 按等级贴标 |

### NClassifyType 显示规则

协议注释里的位含义是：

- `1`：重量，克
- `2`：重量，个
- `4`：尺寸，直径
- `8`：尺寸，面积
- `16`：尺寸，体积

但是前端显示等级表时不能只看 `NClassifyType`。以当前设备数据为准：

- `NSizeGradeNum > 0 && NQualityGradeNum == 0`：按“只有尺寸等级”显示，即使 `NClassifyType == 0`。
- `NQualityGradeNum > 0 && NSizeGradeNum == 0`：按“只有品质等级”显示。
- `NQualityGradeNum > 0 && NSizeGradeNum > 0`：按“尺寸/重量等级 × 品质等级”的组合显示。
- 如果 `NClassifyType` 的 `4/8/16` 任一位为 1，范围列明确是尺寸类。
- 如果 `NClassifyType` 的 `1/2` 任一位为 1，范围列明确是重量类。
- 如果 `NClassifyType == 0`，显示层优先相信 `NSizeGradeNum/NQualityGradeNum` 和名称数组，不因为 `NClassifyType` 为 0 丢弃等级。

也就是说，你现在说的“`NClassifyType = 0` 就只有尺寸等级”，在前端映射层应该理解为：当 `NSizeGradeNum` 有值且 `NQualityGradeNum` 为 0 时，按只有尺寸等级处理。

## GExit 字段

`GExit` 对应 C++ `StGlobalExitInfo`，Go `StGlobalExitInfo`，鸿蒙 `StGlobalExitInfo`。

| Go JSON 字段 | 鸿蒙字段 | 用途 |
| --- | --- | --- |
| `NPulse` | `gexit.nPulse` | 电磁阀脉宽 |
| `VersionFlag` | `gexit.versionFlag` | 48/64 口版本标志 |
| `NLabelPulse` | `gexit.nLabelPulse` | 贴标脉宽 |
| `NDriverPin[48]` | `gexit.nDriverPin` | 每个出口的驱动脚 |
| `DelayTime[48]` | `gexit.Delay_time` | 每个出口的启动延时 |
| `HoldTime[48]` | `gexit.Hold_time` | 每个出口的保持时间 |

## Exit 字段

`Exit` 是 12 个通道的出口配置。

| Go JSON 字段 | 鸿蒙字段 | 用途 |
| --- | --- | --- |
| `Exit[i].LabelExit[4].NDis` | `exit[i].labelExit[j].nDis` | 贴标距离 |
| `Exit[i].LabelExit[4].NDriverPin` | `exit[i].labelExit[j].nDriverPin` | 贴标驱动脚 |
| `Exit[i].Exits[48].NDis` | `exit[i].exits[j].nDis` | 出口距离 |
| `Exit[i].Exits[48].NOffset` | `exit[i].exits[j].nOffset` | 出口偏移 |
| `Exit[i].Exits[48].NDriverPin` | `exit[i].exits[j].nDriverPin` | 出口驱动脚 |

## Paras 字段

`Paras` 是 12 个 IPM/通道参数块。

| Go JSON 字段 | 鸿蒙字段 | 用途 |
| --- | --- | --- |
| `Paras[i].CameraParas[3]` | `paras[i].cameraParas[]` | 彩色相机参数 |
| `Paras[i].IRCameraParas[6]` | `paras[i].irCameraParas[]` | 红外/NIR 参数 |
| `Paras[i].NCupNum` | `paras[i].nCupNum` | 杯数 |

`CameraParas` 和 `IRCameraParas` 内部字段主要用于相机标定、ROI、触发延时、快门、检测阈值、Gamma、像素比例等参数页面。当前鸿蒙 `StGlobalJsonMapper` 尚未映射这部分。

## AnalogDensity / Motor / Weight

| Go JSON 字段 | 鸿蒙字段 | 用途 | 当前状态 |
| --- | --- | --- | --- |
| `AnalogDensity.UAnalogDensity[32]` | `analogdensity.uAnalogDensity` | 模拟密度配置 | 未映射 |
| `Motor[48].BExitId` | `motor[i].bExitId` | 出口 ID | 未映射 |
| `Motor[48].BMotorSwitch` | `motor[i].bMotorSwitch` | 电机开关 | 未映射 |
| `Motor[48].NMotorEnableSwitchNum` | `motor[i].nMotorEnableSwitchNum` | 电机切换个数 | 未映射 |
| `Motor[48].NMotorEnableSwitchWeight` | `motor[i].nMotorEnableSwitchWeight` | 电机切换重量 | 未映射 |
| `Motor[48].FDelayTime` | `motor[i].fDelayTime` | 电机延时 | 未映射 |
| `Motor[48].FHoldTime` | `motor[i].fHoldTime` | 电机保持时间 | 未映射 |
| `GWeight` | 暂无鸿蒙字段 | 全局称重配置 | 未映射 |
| `Weights[12]` | 暂无鸿蒙字段 | 每通道称重参数 | 未映射 |

## 前端补映射优先级

建议先补这几块：

1. `Grade`：这是等级表、品质表、拖拽出口、统计等级名称的核心。
2. `GExit`：首页出口卡片里的脉宽、延时、保持时间、驱动脚需要它。
3. `Exit`：如果要显示/编辑每通道出口距离和贴标距离，需要它。
4. `AnalogDensity / Paras / Motor / Weight`：等相关页面需要时再补。

`Grade` 补齐后，`GlobalDataInterface.updateGlobalConfig()` 里的 `latestGradeInfo`、`gradeDescriptors`、`qualityGradeSummary` 才能真正从下位机 `StGlobal.grade` 派生出来。
