package tcp

import (
	"strings"
	"testing"
	"time"
	"unsafe"
)

func TestNormalizeStStatisticsSubsysIDPrefersHeaderSource(t *testing.T) {
	got := normalizeStStatisticsSubsysID(0x0200, 1)
	if got != 2 {
		t.Fatalf("normalizeStStatisticsSubsysID(0x0200, 1) = %d, want 2", got)
	}
}

func TestNormalizeStStatisticsSubsysIDUsesPayloadFallback(t *testing.T) {
	got := normalizeStStatisticsSubsysID(0, 0x0300)
	if got != 3 {
		t.Fatalf("normalizeStStatisticsSubsysID(0, 0x0300) = %d, want 3", got)
	}
}

func TestNormalizeStStatisticsSubsysIDDefaultsToFSM1(t *testing.T) {
	got := normalizeStStatisticsSubsysID(0, 0)
	if got != 1 {
		t.Fatalf("normalizeStStatisticsSubsysID(0, 0) = %d, want 1", got)
	}
}

func TestParseStStatisticsPayloadReadsBroadcastProgramName(t *testing.T) {
	broadcast := StBroadcastStatistics{}
	broadcast.Statistics.NSubsysId = 2
	broadcast.Statistics.NChannelTotalCount = 42
	copy(broadcast.StrProgramName[:], []byte("程序A"))

	payload := unsafe.Slice((*byte)(unsafe.Pointer(&broadcast)), int(unsafe.Sizeof(broadcast)))
	stats, programName, err := parseStStatisticsPayload(payload)
	if err != nil {
		t.Fatalf("parseStStatisticsPayload() error = %v", err)
	}
	if stats.NSubsysId != 2 || stats.NChannelTotalCount != 42 {
		t.Fatalf("stats = {NSubsysId:%d, NChannelTotalCount:%d}, want {2, 42}", stats.NSubsysId, stats.NChannelTotalCount)
	}
	if programName != "程序A" {
		t.Fatalf("programName = %q, want %q", programName, "程序A")
	}
}

func TestParseStStatisticsPayloadKeepsPlainStatisticsCompatible(t *testing.T) {
	source := StStatistics{
		NSubsysId:             3,
		NChannelTotalCount:    99,
		NIntervalSumperminute: 600,
	}

	payload := unsafe.Slice((*byte)(unsafe.Pointer(&source)), int(unsafe.Sizeof(source)))
	stats, programName, err := parseStStatisticsPayload(payload)
	if err != nil {
		t.Fatalf("parseStStatisticsPayload() error = %v", err)
	}
	if stats.NSubsysId != 3 || stats.NChannelTotalCount != 99 || stats.NIntervalSumperminute != 600 {
		t.Fatalf(
			"stats = {NSubsysId:%d, NChannelTotalCount:%d, NIntervalSumperminute:%d}, want {3, 99, 600}",
			stats.NSubsysId,
			stats.NChannelTotalCount,
			stats.NIntervalSumperminute,
		)
	}
	if programName != "" {
		t.Fatalf("programName = %q, want empty for plain StStatistics", programName)
	}
}

func TestFormatHomeBatchStatsLinePrintsBatchCountAndWeightFields(t *testing.T) {
	stats := StStatistics{
		NSubsysId:             2,
		NChannelTotalCount:    123,
		NChannelWeightCount:   4567,
		NIntervalSumperminute: 600,
	}

	line := formatHomeBatchStatsLine(stats)

	for _, want := range []string{
		"subsys=2",
		"本批个数(NChannelTotalCount)=123",
		"本批重量=4567",
		"NChannelWeightCount=4567",
		"sum(NExitWeightCount)=0",
		"sum(NWeightGradeCount)=0",
	} {
		if !strings.Contains(line, want) {
			t.Fatalf("formatHomeBatchStatsLine() = %q, want it to contain %q", line, want)
		}
	}
}

func TestShouldLogHomeBatchStatsEveryFiveSeconds(t *testing.T) {
	first := time.Unix(100, 0)
	cTCPHomeBatchStatsLastLogAt = time.Time{}
	if !shouldLogHomeBatchStats(first) {
		t.Fatal("first home batch stats log should be allowed")
	}
	cTCPHomeBatchStatsLastLogAt = first
	if shouldLogHomeBatchStats(first.Add(4 * time.Second)) {
		t.Fatal("home batch stats log should be throttled before five seconds")
	}
	if !shouldLogHomeBatchStats(first.Add(5 * time.Second)) {
		t.Fatal("home batch stats log should be allowed at five seconds")
	}
}
