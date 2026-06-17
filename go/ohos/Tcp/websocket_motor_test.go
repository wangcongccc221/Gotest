package tcp

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestMotorInfoCommandIDMatches48Enum(t *testing.T) {
	if cTCPHCMotorInfo != 0x005C {
		t.Fatalf("cTCPHCMotorInfo = 0x%04X, want 0x005C", uint32(cTCPHCMotorInfo))
	}
}

func TestMotorPowerCommandIDsMatch48Enum(t *testing.T) {
	if cTCPHCMotorEnable != 0x0016 {
		t.Fatalf("cTCPHCMotorEnable = 0x%04X, want 0x0016", uint32(cTCPHCMotorEnable))
	}
	if cTCPHCExitClear != 0x001F {
		t.Fatalf("cTCPHCExitClear = 0x%04X, want 0x001F", uint32(cTCPHCExitClear))
	}
	if cTCPHCGlobalExitInfo != 0x0058 {
		t.Fatalf("cTCPHCGlobalExitInfo = 0x%04X, want 0x0058", uint32(cTCPHCGlobalExitInfo))
	}
}

func TestEncodeMotorInfoPayloadMatches48Layout(t *testing.T) {
	motor := StMotorInfo{
		BExitId:                  2,
		BMotorSwitch:             2,
		NMotorEnableSwitchNum:    500,
		NMotorEnableSwitchWeight: 0,
		FDelayTime:               1.5,
		FHoldTime:                2.5,
	}

	payload, err := encodeMotorInfoPayload(motor)
	if err != nil {
		t.Fatalf("encodeMotorInfoPayload() error = %v", err)
	}
	if len(payload) != cTCP48StMotorInfoWireSize {
		t.Fatalf("payload length = %d, want %d", len(payload), cTCP48StMotorInfoWireSize)
	}
	if payload[0] != 2 || payload[1] != 2 {
		t.Fatalf("header bytes = [%d %d], want [2 2]", payload[0], payload[1])
	}
	if got := int32(binary.LittleEndian.Uint32(payload[4:8])); got != 500 {
		t.Fatalf("NMotorEnableSwitchNum = %d, want 500", got)
	}
	if got := math.Float32frombits(binary.LittleEndian.Uint32(payload[12:16])); got != 1.5 {
		t.Fatalf("FDelayTime = %v, want 1.5", got)
	}
	if got := math.Float32frombits(binary.LittleEndian.Uint32(payload[16:20])); got != 2.5 {
		t.Fatalf("FHoldTime = %v, want 2.5", got)
	}
}
