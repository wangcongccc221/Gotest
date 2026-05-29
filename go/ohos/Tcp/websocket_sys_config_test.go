package tcp

import (
	"encoding/binary"
	"testing"
)

func TestSysConfigCommandIDMatches48Enum(t *testing.T) {
	if cTCPHCSysConfig != 0x0050 {
		t.Fatalf("cTCPHCSysConfig = 0x%04X, want 0x0050", uint32(cTCPHCSysConfig))
	}
}

func TestEncodeSysConfigPayloadMatches48Layout(t *testing.T) {
	sys := StSysConfig{
		Width:                  1920,
		Height:                 1080,
		PacketSize:             1500,
		NSystemInfo:            0x1234,
		NSubsysNum:             2,
		NExitNum:               48,
		NClassificationInfo:    0x1F,
		MultiFreq:              1,
		NCameraType:            1,
		CIRClassifyType:        0x13,
		UVClassifyType:         0x03,
		WeightClassifyTpye:     0x01,
		InternalClassifyType:   0x3F,
		UltrasonicClassifyType: 0x03,
		IfWIFIEnable:           0,
		CheckExit:              1,
		CheckNum:               1,
		NIQSEnable:             1,
	}
	sys.ExitState[0] = 7
	sys.NChannelInfo[0] = 2
	sys.NImageUV[0] = 1
	sys.NDataRegistration[0] = 2
	sys.NImageSugar[0] = 3
	sys.NImageUltrasonic[0] = 4
	sys.NCameraDelay[0] = 123456

	payload, err := encodeSysConfigPayload(sys)
	if err != nil {
		t.Fatalf("encodeSysConfigPayload() error = %v", err)
	}
	if len(payload) != cTCP48StSysConfigWireSize {
		t.Fatalf("payload length = %d, want %d", len(payload), cTCP48StSysConfigWireSize)
	}
	if payload[0] != 7 || payload[384] != 2 || payload[388] != 1 || payload[392] != 2 || payload[396] != 3 || payload[400] != 4 {
		t.Fatalf("array header bytes not in 48 layout: got exit=%d channel=%d uv=%d reg=%d sugar=%d ultrasonic=%d",
			payload[0], payload[384], payload[388], payload[392], payload[396], payload[400])
	}
	if got := int32(binary.LittleEndian.Uint32(payload[404:408])); got != 123456 {
		t.Fatalf("NCameraDelay[0] = %d, want 123456", got)
	}
	if got := int32(binary.LittleEndian.Uint32(payload[476:480])); got != 1920 {
		t.Fatalf("Width = %d, want 1920", got)
	}
	if got := binary.LittleEndian.Uint16(payload[488:490]); got != 0x1234 {
		t.Fatalf("NSystemInfo = 0x%04X, want 0x1234", got)
	}
	if payload[490] != 2 || payload[491] != 48 || payload[492] != 0x1F || payload[493] != 1 || payload[503] != 1 {
		t.Fatalf("tail bytes not in 48 layout: got subsys=%d exits=%d classify=0x%02X freq=%d iqs=%d",
			payload[490], payload[491], payload[492], payload[493], payload[503])
	}
}
