package database

import "testing"

func TestVersionUpdateComparisonMatches48Rule(t *testing.T) {
	if !checkUpdateVersion("2.1.0.0", "2.0.9") {
		t.Fatalf("expected higher middle version to require update")
	}
	if checkUpdateVersion("1.0.0", "2.0.0") {
		t.Fatalf("expected lower cloud version to avoid rollback update")
	}
	if !checkUpdateVersion("2.0.1", "2.0.0") {
		t.Fatalf("expected higher patch version to require update")
	}
}

func TestParseCloudVersionScansVersionToken(t *testing.T) {
	label, version := parseCloudVersion(versionUpdateFSM, "FSM_200_V3.0.1.7_Release")
	if label != "V3.0.1.7" || version != "3.0.1.7" {
		t.Fatalf("unexpected FSM version label=%q version=%q", label, version)
	}

	label, version = parseCloudVersion(versionUpdateRSS, "Linux_FruitSort200_V2.0.0.4_20260620")
	if label != "V2.0.0.4" || version != "2.0.0.4" {
		t.Fatalf("unexpected RSS version label=%q version=%q", label, version)
	}
}

func TestBuildControllerUpdateRequestVersionUsesRawVersion(t *testing.T) {
	got := buildControllerUpdateRequestVersion("RM200_L4_FSM_V01.00.01", "FSM_200", "FSM")
	if got != "RM200_L4_FSM" {
		t.Fatalf("unexpected FSM request version: %q", got)
	}
}
