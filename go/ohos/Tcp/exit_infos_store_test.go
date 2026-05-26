package tcp

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestParseStExitInfosConfigValueReads48Format(t *testing.T) {
	var nameBytes [MAX_EXIT_NUM * MAX_TEXT_LENGTH]byte
	nameBytes[0] = 'A'
	var controlBytes [MAX_EXIT_NUM]byte
	for i := range controlBytes {
		controlBytes[i] = 2
	}
	controlBytes[3] = 1

	value, err := json.Marshal(map[string]any{
		"ExitInfos":       base64.StdEncoding.EncodeToString(nameBytes[:]),
		"ExitControlMode": base64.StdEncoding.EncodeToString(controlBytes[:]),
		"ExitBoxType":     []int{0, 1, 0, 1},
	})
	if err != nil {
		t.Fatalf("marshal 48 value: %v", err)
	}

	exitInfos, err := parseStExitInfosConfigValue(string(value))
	if err != nil {
		t.Fatalf("parseStExitInfosConfigValue() error = %v", err)
	}
	if exitInfos.ExitName[0] != 'A' {
		t.Fatalf("ExitName[0] = %q, want A", exitInfos.ExitName[0])
	}
	if exitInfos.ExitControlMode[3] != 1 {
		t.Fatalf("ExitControlMode[3] = %d, want 1", exitInfos.ExitControlMode[3])
	}
	if exitInfos.ExitBoxType[1] != 1 || exitInfos.ExitBoxType[3] != 1 {
		t.Fatalf("ExitBoxType = %v, want indexes 1 and 3 set", exitInfos.ExitBoxType[:4])
	}
}

func TestFormatStExitInfosConfigValueWrites48Format(t *testing.T) {
	exitInfos := defaultStExitInfos()
	exitInfos.ExitName[0] = 'B'
	exitInfos.ExitControlMode[2] = 1
	exitInfos.ExitBoxType[4] = 1

	value, err := formatStExitInfosConfigValue(exitInfos)
	if err != nil {
		t.Fatalf("formatStExitInfosConfigValue() error = %v", err)
	}

	var decoded map[string]json.RawMessage
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		t.Fatalf("saved value is not JSON: %v", err)
	}
	if _, ok := decoded["fsmId"]; ok {
		t.Fatalf("saved value contains Go wrapper field fsmId: %s", value)
	}
	if _, ok := decoded["exitInfos"]; ok {
		t.Fatalf("saved value contains Go wrapper field exitInfos: %s", value)
	}
	var exitBoxType []int
	if err := json.Unmarshal(decoded["ExitBoxType"], &exitBoxType); err != nil {
		t.Fatalf("ExitBoxType is not a JSON array: %v, value=%s", err, string(decoded["ExitBoxType"]))
	}
	if len(exitBoxType) != MAX_EXIT_NUM || exitBoxType[4] != 1 {
		t.Fatalf("ExitBoxType = %v, want 48 item array with index 4 set", exitBoxType)
	}

	parsed, err := parseStExitInfosConfigValue(value)
	if err != nil {
		t.Fatalf("parse saved value: %v", err)
	}
	if parsed.ExitName[0] != 'B' || parsed.ExitControlMode[2] != 1 || parsed.ExitBoxType[4] != 1 {
		t.Fatalf("parsed saved value lost fields: %+v", parsed)
	}
}

func TestApplyStExitInfosUpdatePreservesOmittedFields(t *testing.T) {
	base := defaultStExitInfos()
	base.ExitName[0] = 'C'
	base.ExitControlMode[5] = 1
	update := webSocketExitInfosControl{
		ExitBoxType: make([]uint8, MAX_EXIT_NUM),
	}
	update.ExitBoxType[5] = 1

	merged, err := applyStExitInfosUpdate(base, update)
	if err != nil {
		t.Fatalf("applyStExitInfosUpdate() error = %v", err)
	}
	if merged.ExitName[0] != 'C' {
		t.Fatalf("ExitName[0] = %q, want preserved C", merged.ExitName[0])
	}
	if merged.ExitControlMode[5] != 1 {
		t.Fatalf("ExitControlMode[5] = %d, want preserved 1", merged.ExitControlMode[5])
	}
	if merged.ExitBoxType[5] != 1 {
		t.Fatalf("ExitBoxType[5] = %d, want updated 1", merged.ExitBoxType[5])
	}
}

func TestApplyStExitInfosUpdateWritesAndClearsExitNameWhenFieldPresent(t *testing.T) {
	base := defaultStExitInfos()
	nameBytes := make([]uint8, MAX_EXIT_NUM*MAX_TEXT_LENGTH)
	nameBytes[0] = 'N'

	merged, err := applyStExitInfosUpdate(base, webSocketExitInfosControl{ExitName: nameBytes})
	if err != nil {
		t.Fatalf("applyStExitInfosUpdate() write error = %v", err)
	}
	if merged.ExitName[0] != 'N' {
		t.Fatalf("ExitName[0] = %q, want N", merged.ExitName[0])
	}

	cleared, err := applyStExitInfosUpdate(merged, webSocketExitInfosControl{ExitName: make([]uint8, MAX_EXIT_NUM*MAX_TEXT_LENGTH)})
	if err != nil {
		t.Fatalf("applyStExitInfosUpdate() clear error = %v", err)
	}
	if cleared.ExitName[0] != 0 {
		t.Fatalf("ExitName[0] = %q, want cleared zero", cleared.ExitName[0])
	}
}
