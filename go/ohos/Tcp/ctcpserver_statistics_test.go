package tcp

import (
	"testing"
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
