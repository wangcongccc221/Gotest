package tcp

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"testing"
)

func TestParseDataStWaveInfoReadsWaveSamples(t *testing.T) {
	source := StWaveInfo{
		NChannelId:  0x0111,
		FruitWeight: 123.5,
	}
	source.Waveform0[0] = 10
	source.Waveform0[255] = 4095
	source.Waveform1[0] = 20
	source.Waveform1[255] = 2048

	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, source); err != nil {
		t.Fatalf("build StWaveInfo payload: %v", err)
	}

	got, err := ParseData[StWaveInfo](buf.Bytes())
	if err != nil {
		t.Fatalf("ParseData[StWaveInfo] failed: %v", err)
	}

	if got.NChannelId != source.NChannelId || got.FruitWeight != source.FruitWeight {
		t.Fatalf("unexpected wave header: channel=%#x weight=%.1f", got.NChannelId, got.FruitWeight)
	}
	if got.Waveform0[0] != 10 || got.Waveform0[255] != 4095 {
		t.Fatalf("unexpected waveform0 samples: first=%d last=%d", got.Waveform0[0], got.Waveform0[255])
	}
	if got.Waveform1[0] != 20 || got.Waveform1[255] != 2048 {
		t.Fatalf("unexpected waveform1 samples: first=%d last=%d", got.Waveform1[0], got.Waveform1[255])
	}
}

func TestNewCTCPWaveInfoRealtimeFramePreservesHeaderAndWaveSamples(t *testing.T) {
	head := cTCPServerCommandHead{
		NSrcId: 0x0101,
		NDstId: 0x0202,
		NCmdId: cmdWAMWaveInfo,
	}
	source := StWaveInfo{
		NChannelId:  0x0112,
		FruitWeight: 88.25,
	}
	source.Waveform0[3] = 333
	source.Waveform1[4] = 444

	frame := newCTCPWaveInfoRealtimeFrame(head, source)
	if frame.SrcID != head.NSrcId || frame.DstID != head.NDstId || frame.CmdID != head.NCmdId {
		t.Fatalf("frame header mismatch: %+v", frame)
	}
	if frame.WaveInfo.NChannelId != source.NChannelId {
		t.Fatalf("frame channel mismatch: got %#x want %#x", frame.WaveInfo.NChannelId, source.NChannelId)
	}

	raw, err := json.Marshal(frame)
	if err != nil {
		t.Fatalf("marshal frame: %v", err)
	}

	var decoded struct {
		SrcID    int32      `json:"srcId"`
		DstID    int32      `json:"dstId"`
		CmdID    int32      `json:"cmdId"`
		WaveInfo StWaveInfo `json:"waveInfo"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal frame: %v", err)
	}
	if decoded.SrcID != head.NSrcId || decoded.WaveInfo.Waveform0[3] != 333 || decoded.WaveInfo.Waveform1[4] != 444 {
		t.Fatalf("decoded frame lost data: %+v", decoded)
	}
}
