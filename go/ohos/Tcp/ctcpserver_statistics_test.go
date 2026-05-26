package tcp

import "testing"

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
