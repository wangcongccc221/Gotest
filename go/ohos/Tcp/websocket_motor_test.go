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

func TestIpmPowerCommandIDsMatch48Enum(t *testing.T) {
	if cTCPHCIpmShutdown != 0x2008 {
		t.Fatalf("cTCPHCIpmShutdown = 0x%04X, want 0x2008", uint32(cTCPHCIpmShutdown))
	}
}

func TestParseARPOutputExtractsRemoteMAC(t *testing.T) {
	output := `? (192.168.0.17) at aa:bb:cc:dd:ee:ff [ether] on wlan0`

	got, ok := parseARPOutputMAC(output, "192.168.0.17")
	if !ok {
		t.Fatal("parseARPOutputMAC() did not find MAC")
	}
	if got != "AA-BB-CC-DD-EE-FF" {
		t.Fatalf("mac = %q, want AA-BB-CC-DD-EE-FF", got)
	}
}

func TestParseARPOutputRejectsDifferentIP(t *testing.T) {
	output := `? (192.168.0.18) at aa:bb:cc:dd:ee:ff [ether] on wlan0`

	if mac, ok := parseARPOutputMAC(output, "192.168.0.17"); ok {
		t.Fatalf("parseARPOutputMAC() = %q, want not found", mac)
	}
}

func TestBuildWakeOnLANPacketRepeatsMAC(t *testing.T) {
	packet, err := buildWakeOnLANPacket("AA-BB-CC-DD-EE-FF")
	if err != nil {
		t.Fatalf("buildWakeOnLANPacket() error = %v", err)
	}
	if len(packet) != 102 {
		t.Fatalf("packet length = %d, want 102", len(packet))
	}
	for i := 0; i < 6; i++ {
		if packet[i] != 0xFF {
			t.Fatalf("packet[%d] = 0x%02X, want 0xFF", i, packet[i])
		}
	}
	wantMAC := []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	for offset := 6; offset < len(packet); offset += len(wantMAC) {
		for i := range wantMAC {
			if packet[offset+i] != wantMAC[i] {
				t.Fatalf("packet[%d] = 0x%02X, want 0x%02X", offset+i, packet[offset+i], wantMAC[i])
			}
		}
	}
}

func TestBuildWakeOnLANPacketRejectsInvalidMAC(t *testing.T) {
	if _, err := buildWakeOnLANPacket("AA-BB-CC-DD-EE-100"); err == nil {
		t.Fatal("buildWakeOnLANPacket() error = nil, want invalid MAC error")
	}
}

func TestResetCupCommandIDsMatch48Enum(t *testing.T) {
	if cTCPHCWAMWeightReset != 0x0111 {
		t.Fatalf("cTCPHCWAMWeightReset = 0x%04X, want 0x0111", uint32(cTCPHCWAMWeightReset))
	}
	if cTCPHCWAMCupStateReset != 0x0112 {
		t.Fatalf("cTCPHCWAMCupStateReset = 0x%04X, want 0x0112", uint32(cTCPHCWAMCupStateReset))
	}
}

func TestBuildWAMDestIDsForResetCupMatches48GetWAMID(t *testing.T) {
	tests := []struct {
		name  string
		fsmID int32
		want  []int32
	}{
		{name: "first subsystem", fsmID: 0x0100, want: []int32{0x01D0}},
		{name: "second subsystem", fsmID: 0x0200, want: []int32{0x02D0}},
		{name: "broadcast fallback", fsmID: -1, want: []int32{0x01D0}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildWAMDestIDsForResetCup(tt.fsmID)
			if len(got) != len(tt.want) {
				t.Fatalf("dest count = %d, want %d (%v)", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("dest[%d] = 0x%04X, want 0x%04X", i, uint32(got[i]), uint32(tt.want[i]))
				}
			}
		})
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
