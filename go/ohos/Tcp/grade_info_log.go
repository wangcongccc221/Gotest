package tcp

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	stGradeInfoPeriodicLogInterval = 10 * time.Second
	stMotorInfoPeriodicLogInterval = 20 * time.Second
)

var (
	lastStGradeInfoSnapshot   StGradeInfo
	lastStGradeInfoSnapshotOK bool
	lastStGradeInfoSnapshotMu sync.RWMutex

	lastStExitInfosSnapshot   StExitInfos
	lastStExitInfosFSMID      int32
	lastStExitInfosSnapshotOK bool
	lastStExitInfosSnapshotMu sync.RWMutex

	lastStMotorInfoSnapshot   StMotorInfo
	lastStMotorInfoFSMID      int32
	lastStMotorInfoSnapshotOK bool
	lastStMotorInfoSnapshotMu sync.RWMutex

	stGradeInfoPeriodicLogMu  sync.Mutex
	stGradeInfoPeriodicCancel context.CancelFunc

	stMotorInfoPeriodicLogMu  sync.Mutex
	stMotorInfoPeriodicCancel context.CancelFunc
)

// setLastStGradeInfoSnapshot 在收到/发送等级数据时更新，供周期日志读取。
func setLastStGradeInfoSnapshot(grade StGradeInfo) {
	lastStGradeInfoSnapshotMu.Lock()
	lastStGradeInfoSnapshot = grade
	lastStGradeInfoSnapshotOK = true
	lastStGradeInfoSnapshotMu.Unlock()
}

func latestStGradeInfoSnapshot() (StGradeInfo, bool) {
	lastStGradeInfoSnapshotMu.RLock()
	defer lastStGradeInfoSnapshotMu.RUnlock()
	return lastStGradeInfoSnapshot, lastStGradeInfoSnapshotOK
}

// setLastStExitInfosSnapshot 在收到出口本地配置时更新，供日志/排查读取。
func setLastStExitInfosSnapshot(fsmID int32, exitInfos StExitInfos) {
	lastStExitInfosSnapshotMu.Lock()
	lastStExitInfosSnapshot = exitInfos
	lastStExitInfosFSMID = fsmID
	lastStExitInfosSnapshotOK = true
	lastStExitInfosSnapshotMu.Unlock()
}

func latestStExitInfosSnapshot() (int32, StExitInfos, bool) {
	lastStExitInfosSnapshotMu.RLock()
	defer lastStExitInfosSnapshotMu.RUnlock()
	return lastStExitInfosFSMID, lastStExitInfosSnapshot, lastStExitInfosSnapshotOK
}

// setLastStMotorInfoSnapshot 在收到有效电机下发数据时更新，供周期日志读取。
func setLastStMotorInfoSnapshot(fsmID int32, motor StMotorInfo) {
	lastStMotorInfoSnapshotMu.Lock()
	lastStMotorInfoSnapshot = motor
	lastStMotorInfoFSMID = fsmID
	lastStMotorInfoSnapshotOK = true
	lastStMotorInfoSnapshotMu.Unlock()
}

func latestStMotorInfoSnapshot() (int32, StMotorInfo, bool) {
	lastStMotorInfoSnapshotMu.RLock()
	defer lastStMotorInfoSnapshotMu.RUnlock()
	return lastStMotorInfoFSMID, lastStMotorInfoSnapshot, lastStMotorInfoSnapshotOK
}

// DumpStGradeInfo 格式化 StGradeInfo 全部字段。
// 后续若要改输出格式（只打部分字段、uint8 转字符串等），改这个函数即可。
func DumpStGradeInfo(grade StGradeInfo) string {
	text, err := FormatDataFullJSON(grade)
	if err != nil {
		return fmt.Sprintf("StGradeInfo JSON 序列化失败: %v", err)
	}
	return text
}

// LogStGradeInfo 输出一次 StGradeInfo 全量字段到 CTCP 日志（HiLog 搜「StGradeInfo 全量」）。
func LogStGradeInfo(grade StGradeInfo) {
	appendCTCPLogChunks("StGradeInfo 全量", DumpStGradeInfo(grade))
}

// LogQualityGradeDiagnostics 输出每个品质等级在各尺寸档位下的分级条件。
func LogQualityGradeDiagnostics(source string, grade StGradeInfo) {
	var b strings.Builder
	qualityCount := min(int(grade.NQualityGradeNum), cTCPServerMaxSizeGradeNum)
	sizeCount := min(int(grade.NSizeGradeNum), cTCPServerMaxSizeGradeNum)
	if sizeCount <= 0 {
		sizeCount = 1
	}
	fmt.Fprintf(
		&b,
		"source=%s NSizeGradeNum=%d NQualityGradeNum=%d\n",
		source,
		grade.NSizeGradeNum,
		grade.NQualityGradeNum,
	)

	for qualityIndex := 0; qualityIndex < qualityCount; qualityIndex++ {
		qualityName := realtimeSaveFixedName(grade.StrQualityGradeName[:], qualityIndex)
		fmt.Fprintf(&b, "\n品质[%d] name=%q\n", qualityIndex+1, qualityName)
		for sizeIndex := 0; sizeIndex < sizeCount; sizeIndex++ {
			gradeIndex := qualityIndex*cTCPServerMaxSizeGradeNum + sizeIndex
			item := grade.Grades[gradeIndex]
			sizeName := realtimeSaveFixedName(grade.StrSizeGradeName[:], sizeIndex)
			fmt.Fprintf(
				&b,
				"  尺寸[%d] name=%q Grades[%d] sizeRange=[%g,%g] fruitNum=%d exit=%d "+
					"颜色=%s 形状=%s 固形物=%s 瑕疵=%s 碰伤=%s 腐烂=%s 糖度=%s 酸度=%s "+
					"空心=%s 浮皮=%s 褐变=%s 糖心=%s 硬度=%s 水心=%s 标签=%d\n",
				sizeIndex+1,
				sizeName,
				gradeIndex,
				item.NMinSize,
				item.NMaxSize,
				item.NFruitNum,
				item.Exit(),
				formatQualitySelector(grade.StrColorGradeName[:], item.NColorGrade),
				formatQualitySelector(grade.StrShapeGradeName[:], item.SbShapeSize),
				formatQualitySelector(grade.StDensityGradeName[:], item.SbDensity),
				formatQualitySelector(grade.StFlawareaGradeName[:], item.SbFlawArea),
				formatQualitySelector(grade.StBruiseGradeName[:], item.SbBruise),
				formatQualitySelector(grade.StRotGradeName[:], item.SbRot),
				formatQualitySelector(grade.StSugarGradeName[:], item.SbSugar),
				formatQualitySelector(grade.StAcidityGradeName[:], item.SbAcidity),
				formatQualitySelector(grade.StHollowGradeName[:], item.SbHollow),
				formatQualitySelector(grade.StSkinGradeName[:], item.SbSkin),
				formatQualitySelector(grade.StBrownGradeName[:], item.SbBrown),
				formatQualitySelector(grade.StTangxinGradeName[:], item.SbTangxin),
				formatQualitySelector(grade.StRigidityGradeName[:], item.SbRigidity),
				formatQualitySelector(grade.StWaterGradeName[:], item.SbWater),
				item.SbLabelbyGrade,
			)
		}
	}
	appendCTCPLogChunks("[DEBUG-QUALITY-GRADES-0615] Go接收", strings.TrimSpace(b.String()))
}

func formatQualitySelector(names []uint8, value int8) string {
	if value == 0x7F {
		return "全部(127)"
	}
	if value < 0 {
		return fmt.Sprintf("无效(%d)", value)
	}
	name := realtimeSaveFixedName(names, int(value))
	if name == "" {
		return fmt.Sprintf("索引%d", value)
	}
	return fmt.Sprintf("%q(索引%d)", name, value)
}

// DumpStMotorInfo 格式化 StMotorInfo 全部字段。
func DumpStMotorInfo(motor StMotorInfo) string {
	text, err := FormatDataFullJSON(motor)
	if err != nil {
		return fmt.Sprintf("StMotorInfo JSON 序列化失败: %v", err)
	}
	return text
}

// LogStMotorInfo 输出一次 StMotorInfo 全量字段到 CTCP 日志（HiLog 搜「StMotorInfo Go结构体」）。
func LogStMotorInfo(motor StMotorInfo) {
	appendCTCPLogChunks("StMotorInfo Go结构体", fmt.Sprintf("%+v", motor))
	appendCTCPLogChunks("StMotorInfo 全量JSON", DumpStMotorInfo(motor))
}

// LogLatestStMotorInfo 输出最近一次缓存的 StMotorInfo；尚无数据时只打提示。
func LogLatestStMotorInfo() {
	_, motor, ok := latestStMotorInfoSnapshot()
	if !ok {
		setCTCPServerLastMessage("StMotorInfo 周期输出: 尚无缓存数据（等待 WebSocket saveMotorInfo）")
		return
	}
	LogStMotorInfo(motor)
}

// LogLatestStGradeInfo 输出最近一次缓存的 StGradeInfo；尚无数据时只打提示。
func LogLatestStGradeInfo() {
	grade, ok := latestStGradeInfoSnapshot()
	if !ok {
		setCTCPServerLastMessage("StGradeInfo 周期输出: 尚无缓存数据（等待 FSM_CMD_CONFIG 或 WebSocket save）")
		return
	}
	LogStGradeInfo(grade)
}

// StartStGradeInfoPeriodicLog 每 10 秒输出一次最近一次 StGradeInfo 全量字段。
func StartStGradeInfoPeriodicLog() {
	StartStGradeInfoPeriodicLogWithInterval(stGradeInfoPeriodicLogInterval)
}

// StartStGradeInfoPeriodicLogWithInterval 按指定间隔周期输出 StGradeInfo。
func StartStGradeInfoPeriodicLogWithInterval(interval time.Duration) {
	if interval <= 0 {
		interval = stGradeInfoPeriodicLogInterval
	}

	stGradeInfoPeriodicLogMu.Lock()
	defer stGradeInfoPeriodicLogMu.Unlock()
	if stGradeInfoPeriodicCancel != nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	stGradeInfoPeriodicCancel = cancel

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		setCTCPServerLastMessage("StGradeInfo 周期输出已启动: 每 %s 打印一次全量字段", interval)
		for {
			select {
			case <-ctx.Done():
				setCTCPServerLastMessage("StGradeInfo 周期输出已停止")
				return
			case <-ticker.C:
				LogLatestStGradeInfo()
			}
		}
	}()
}

// StartStMotorInfoPeriodicLog 每 20 秒输出一次最近一次 StMotorInfo 全量字段。
func StartStMotorInfoPeriodicLog() {
	StartStMotorInfoPeriodicLogWithInterval(stMotorInfoPeriodicLogInterval)
}

// StartStMotorInfoPeriodicLogWithInterval 按指定间隔周期输出 StMotorInfo。
func StartStMotorInfoPeriodicLogWithInterval(interval time.Duration) {
	if interval <= 0 {
		interval = stMotorInfoPeriodicLogInterval
	}

	stMotorInfoPeriodicLogMu.Lock()
	defer stMotorInfoPeriodicLogMu.Unlock()
	if stMotorInfoPeriodicCancel != nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	stMotorInfoPeriodicCancel = cancel

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		setCTCPServerLastMessage("StMotorInfo 周期输出已启动: 每 %s 打印一次全量字段", interval)
		for {
			select {
			case <-ctx.Done():
				setCTCPServerLastMessage("StMotorInfo 周期输出已停止")
				return
			case <-ticker.C:
				LogLatestStMotorInfo()
			}
		}
	}()
}

// StopStGradeInfoPeriodicLog 停止周期输出。
func StopStGradeInfoPeriodicLog() {
	stGradeInfoPeriodicLogMu.Lock()
	cancel := stGradeInfoPeriodicCancel
	stGradeInfoPeriodicCancel = nil
	stGradeInfoPeriodicLogMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// StopStMotorInfoPeriodicLog 停止周期输出。
func StopStMotorInfoPeriodicLog() {
	stMotorInfoPeriodicLogMu.Lock()
	cancel := stMotorInfoPeriodicCancel
	stMotorInfoPeriodicCancel = nil
	stMotorInfoPeriodicLogMu.Unlock()
	if cancel != nil {
		cancel()
	}
}
