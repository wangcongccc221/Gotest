package tcp

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	fsmStatsCSVDirEnv            = "FSM_STATS_CSV_DIR"
	fsmStatsCSVDefaultDir        = "diagnostics/fsm_stats"
	fsmStatsCSVReceiveMinSpacing = time.Second
)

type fsmStatsCSVReceiveState struct {
	LastLoggedAt time.Time
}

type fsmStatsCSVLogger struct {
	mu               sync.Mutex
	file             *os.File
	writer           *csv.Writer
	path             string
	receiveBySubsys  map[int32]fsmStatsCSVReceiveState
	lastNoSnapshotAt time.Time
}

var defaultFSMStatsCSVLogger = &fsmStatsCSVLogger{
	receiveBySubsys: make(map[int32]fsmStatsCSVReceiveState),
}

func recordFSMStatsReceiveCSV(stats StStatistics, receivedAt time.Time) {
	defaultFSMStatsCSVLogger.recordReceive(stats, receivedAt)
}

func recordFSMStatsSnapshotCSV(now time.Time, entries []cTCPStStatisticsSnapshotEntry) {
	defaultFSMStatsCSVLogger.recordSnapshots(now, entries)
}

func recordFSMHomeStatsCSV(now time.Time, payload homeStatsPayload) {
	defaultFSMStatsCSVLogger.recordHome(now, payload)
}

func recordFSMHomeStatsNoSnapshotCSV(now time.Time) {
	defaultFSMStatsCSVLogger.recordNoSnapshot(now)
}

func (l *fsmStatsCSVLogger) recordReceive(stats StStatistics, receivedAt time.Time) {
	if receivedAt.IsZero() {
		receivedAt = cTCPNow()
	}
	subsysID := stats.NSubsysId
	if subsysID <= 0 {
		subsysID = 1
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	last := l.receiveBySubsys[subsysID]
	if !last.LastLoggedAt.IsZero() && receivedAt.Sub(last.LastLoggedAt) < fsmStatsCSVReceiveMinSpacing {
		return
	}
	l.receiveBySubsys[subsysID] = fsmStatsCSVReceiveState{LastLoggedAt: receivedAt}

	row := l.baseRow(receivedAt, "receive")
	row[3] = strconv.FormatInt(int64(subsysID), 10)
	row[4] = formatFSMStatsCSVTime(receivedAt)
	row[5] = "0"
	row[6] = "false"
	row[7] = strconv.FormatUint(homeStatsTotalCount(stats), 10)
	rawWeight := homeStatsTotalWeight(stats)
	row[8] = formatFSMStatsCSVFloat(rawWeight, 0)
	row[9] = formatFSMStatsCSVFloat(rawWeight/homeStatsWeightScale, 6)
	row[10] = strconv.FormatInt(int64(stats.NIntervalSumperminute), 10)
	row[11] = formatFSMStatsCSVFloat(calculateHomeStatsSortSpeedPercent(float64(stats.NIntervalSumperminute)), 2)
	row[16] = strconv.FormatInt(int64(stats.NTotalCupNum), 10)
	row[17] = strconv.FormatUint(sumHomeStatsUint64(stats.NExitCount[:]), 10)
	row[18] = strconv.FormatUint(sumHomeStatsUint64(stats.NGradeCount[:]), 10)
	row[19] = strconv.FormatUint(sumHomeStatsUint64(stats.NExitWeightCount[:]), 10)
	row[20] = strconv.FormatUint(sumHomeStatsUint64(stats.NWeightGradeCount[:]), 10)
	row[21] = strconv.FormatInt(int64(stats.NInterval), 10)
	row[22] = strconv.FormatInt(int64(stats.NPulseInterval), 10)
	l.writeRowLocked(row)
}

func (l *fsmStatsCSVLogger) recordSnapshots(now time.Time, entries []cTCPStStatisticsSnapshotEntry) {
	if now.IsZero() {
		now = cTCPNow()
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	for _, entry := range entries {
		stats := entry.Stats
		subsysID := stats.NSubsysId
		if subsysID <= 0 {
			subsysID = 1
		}
		ageMs := int64(0)
		if !entry.LatestAt.IsZero() {
			ageMs = now.Sub(entry.LatestAt).Milliseconds()
			if ageMs < 0 {
				ageMs = 0
			}
		}
		row := l.baseRow(now, "snapshot")
		row[3] = strconv.FormatInt(int64(subsysID), 10)
		row[4] = formatFSMStatsCSVTime(entry.LatestAt)
		row[5] = strconv.FormatInt(ageMs, 10)
		row[6] = strconv.FormatBool(entry.Stale)
		row[7] = strconv.FormatUint(homeStatsTotalCount(stats), 10)
		rawWeight := homeStatsTotalWeight(stats)
		row[8] = formatFSMStatsCSVFloat(rawWeight, 0)
		row[9] = formatFSMStatsCSVFloat(rawWeight/homeStatsWeightScale, 6)
		row[10] = strconv.FormatInt(int64(stats.NIntervalSumperminute), 10)
		row[11] = formatFSMStatsCSVFloat(calculateHomeStatsSortSpeedPercent(float64(stats.NIntervalSumperminute)), 2)
		row[16] = strconv.FormatInt(int64(stats.NTotalCupNum), 10)
		row[17] = strconv.FormatUint(sumHomeStatsUint64(stats.NExitCount[:]), 10)
		row[18] = strconv.FormatUint(sumHomeStatsUint64(stats.NGradeCount[:]), 10)
		row[19] = strconv.FormatUint(sumHomeStatsUint64(stats.NExitWeightCount[:]), 10)
		row[20] = strconv.FormatUint(sumHomeStatsUint64(stats.NWeightGradeCount[:]), 10)
		row[21] = strconv.FormatInt(int64(stats.NInterval), 10)
		row[22] = strconv.FormatInt(int64(stats.NPulseInterval), 10)
		l.writeRowLocked(row)
	}
}

func (l *fsmStatsCSVLogger) recordHome(now time.Time, payload homeStatsPayload) {
	if now.IsZero() {
		now = cTCPNow()
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	row := l.baseRow(now, "home")
	row[3] = strconv.FormatInt(int64(payload.SubsysId), 10)
	row[7] = strconv.FormatUint(payload.BatchCount, 10)
	row[9] = formatFSMStatsCSVFloat(payload.BatchWeightTon, 6)
	row[10] = formatFSMStatsCSVFloat(payload.SortSpeedPerMinute, 2)
	row[11] = formatFSMStatsCSVFloat(payload.SortSpeedPercent, 2)
	row[12] = formatFSMStatsCSVFloat(payload.CupFillEfficiency, 2)
	row[13] = formatFSMStatsCSVFloat(payload.RealtimeOutputTonPerHour, 6)
	row[14] = formatFSMStatsCSVFloat(payload.RealtimeOutputPercent, 2)
	row[15] = formatFSMStatsCSVFloat(payload.AverageWeightG, 2)
	row[23] = payload.ProgramName
	row[24] = payload.RunningTimeText
	row[25] = strconv.FormatBool(payload.IsProcessing)
	l.writeRowLocked(row)
}

func (l *fsmStatsCSVLogger) recordNoSnapshot(now time.Time) {
	if now.IsZero() {
		now = cTCPNow()
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.lastNoSnapshotAt.IsZero() && now.Sub(l.lastNoSnapshotAt) < 5*time.Second {
		return
	}
	l.lastNoSnapshotAt = now
	l.writeRowLocked(l.baseRow(now, "no_snapshot"))
}

func (l *fsmStatsCSVLogger) baseRow(at time.Time, rowType string) []string {
	row := make([]string, len(fsmStatsCSVHeader()))
	row[0] = formatFSMStatsCSVTime(at)
	row[1] = strconv.FormatInt(at.UnixMilli(), 10)
	row[2] = rowType
	return row
}

func (l *fsmStatsCSVLogger) writeRowLocked(row []string) {
	if err := l.ensureOpenLocked(); err != nil {
		return
	}
	if err := l.writer.Write(row); err != nil {
		return
	}
	l.writer.Flush()
	_ = l.writer.Error()
}

func (l *fsmStatsCSVLogger) ensureOpenLocked() error {
	if l.writer != nil {
		return nil
	}

	dir := stringsTrimmedEnv(fsmStatsCSVDirEnv)
	if dir == "" {
		dir = fsmStatsCSVDefaultDir
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fallback := filepath.Join(os.TempDir(), "fsm_stats")
		if fallbackErr := os.MkdirAll(fallback, 0o755); fallbackErr != nil {
			return fmt.Errorf("mkdir %s: %w; fallback %s: %v", dir, err, fallback, fallbackErr)
		}
		dir = fallback
	}

	path := filepath.Join(dir, "fsm_stats_"+cTCPNow().Format("20060102-150405")+".csv")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	l.file = file
	l.writer = csv.NewWriter(file)
	l.path = path
	if stat, statErr := file.Stat(); statErr == nil && stat.Size() == 0 {
		if writeErr := l.writer.Write(fsmStatsCSVHeader()); writeErr != nil {
			_ = file.Close()
			l.file = nil
			l.writer = nil
			l.path = ""
			return writeErr
		}
		l.writer.Flush()
		if flushErr := l.writer.Error(); flushErr != nil {
			_ = file.Close()
			l.file = nil
			l.writer = nil
			l.path = ""
			return flushErr
		}
	}
	return nil
}

func fsmStatsCSVHeader() []string {
	return []string{
		"local_time",
		"unix_ms",
		"row_type",
		"subsys_id",
		"last_receive_time",
		"last_receive_age_ms",
		"stale",
		"batch_count",
		"batch_weight_raw",
		"batch_weight_ton",
		"sort_speed_per_min",
		"sort_speed_percent",
		"cup_fill_efficiency_percent",
		"realtime_output_ton_per_hour",
		"realtime_output_percent",
		"average_weight_g",
		"total_cup_num",
		"exit_count_sum",
		"grade_count_sum",
		"exit_weight_sum",
		"grade_weight_sum",
		"n_interval",
		"pulse_interval_ms",
		"program_name",
		"running_time_text",
		"is_processing",
	}
}

func formatFSMStatsCSVTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return cTCPLocalTime(value).Format("2006-01-02 15:04:05.000")
}

func formatFSMStatsCSVFloat(value float64, precision int) string {
	if !isFiniteHomeStats(value) {
		return ""
	}
	return strconv.FormatFloat(value, 'f', precision, 64)
}

func stringsTrimmedEnv(name string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return ""
	}
	return filepath.Clean(value)
}
