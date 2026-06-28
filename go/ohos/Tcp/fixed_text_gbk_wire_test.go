package tcp

import (
	"encoding/json"
	"testing"
	"unsafe"
)

func TestEncodeGradeInfoPayloadConvertsFixedTextFieldsToGBK(t *testing.T) {
	var grade StGradeInfo
	copy(grade.StrFruitName[:], []byte("苹果"))
	copy(grade.StrSizeGradeName[:], []byte("大果"))
	copy(grade.StrQualityGradeName[:], []byte("优质"))

	payload, err := encodeGradeInfoPayload(grade)
	if err != nil {
		t.Fatalf("encodeGradeInfoPayload() error = %v", err)
	}

	var encoded StGradeInfo
	encodedBytes := unsafe.Slice((*byte)(unsafe.Pointer(&encoded)), int(unsafe.Sizeof(encoded)))
	copy(encodedBytes, payload)

	assertBytesPrefix(t, encoded.StrFruitName[:], []byte{0xC6, 0xBB, 0xB9, 0xFB})
	assertBytesPrefix(t, encoded.StrSizeGradeName[:], []byte{0xB4, 0xF3, 0xB9, 0xFB})
	assertBytesPrefix(t, encoded.StrQualityGradeName[:], []byte{0xD3, 0xC5, 0xD6, 0xCA})
	assertBytesPrefix(t, grade.StrFruitName[:], []byte("苹果"))
}

func TestGradeNameTextOverridesUseGBKByteBudget(t *testing.T) {
	var grade StGradeInfo
	grade.NSizeGradeNum = 2
	grade.NQualityGradeNum = 1

	wire := gradeInfoForGBKWireWithNameTexts(grade, &webSocketGradeNameTexts{
		SizeGradeNames:    []string{"一二三四五六七", "A1一二三四五六"},
		QualityGradeNames: []string{"优良一级"},
	})

	assertBytesPrefix(
		t,
		wire.StrSizeGradeName[:cTCPServerMaxTextLength],
		[]byte{0xD2, 0xBB, 0xB6, 0xFE, 0xC8, 0xFD, 0xCB, 0xC4, 0xCE, 0xE5, 0xC1, 0xF9},
	)
	assertBytesPrefix(
		t,
		wire.StrSizeGradeName[cTCPServerMaxTextLength:cTCPServerMaxTextLength*2],
		[]byte{0x41, 0x31, 0xD2, 0xBB, 0xB6, 0xFE, 0xC8, 0xFD, 0xCB, 0xC4, 0xCE, 0xE5},
	)
	assertBytesPrefix(
		t,
		wire.StrQualityGradeName[:cTCPServerMaxTextLength],
		[]byte{0xD3, 0xC5, 0xC1, 0xBC, 0xD2, 0xBB, 0xBC, 0xB6},
	)
}

func TestSaveCTCPStGlobalFullJSONIncludesFullGBKGradeNameTexts(t *testing.T) {
	var global StGlobal
	global.Grade.NSizeGradeNum = 1
	global.Grade.NQualityGradeNum = 1
	writeGBKFixedTextSlot(global.Grade.StrSizeGradeName[:cTCPServerMaxTextLength], "一二三四五六")
	writeGBKFixedTextSlot(global.Grade.StrQualityGradeName[:cTCPServerMaxTextLength], "优良一级")

	jsonText := saveCTCPStGlobalFullJSON(global)
	if jsonText == "" {
		t.Fatal("saveCTCPStGlobalFullJSON() returned empty JSON")
	}

	var decoded struct {
		GradeNameTexts struct {
			SizeGradeNames    []string
			QualityGradeNames []string
		}
	}
	if err := json.Unmarshal([]byte(jsonText), &decoded); err != nil {
		t.Fatalf("unmarshal StGlobal JSON: %v", err)
	}

	if got := decoded.GradeNameTexts.SizeGradeNames; len(got) != 1 || got[0] != "一二三四五六" {
		t.Fatalf("SizeGradeNames = %#v, want [一二三四五六]", got)
	}
	if got := decoded.GradeNameTexts.QualityGradeNames; len(got) != 1 || got[0] != "优良一级" {
		t.Fatalf("QualityGradeNames = %#v, want [优良一级]", got)
	}
}

func TestBuildExitDisplayBroadcastPacketConvertsNamesToGBK(t *testing.T) {
	info := defaultExitDisplayInfo()
	info.DisplayType = 1
	info.DisplayNames[0] = "自定义出口"

	packet := buildExitDisplayBroadcastPacket(info, StGradeInfo{}, false)

	nameStart := 1 + 4
	assertBytesPrefix(t, packet[nameStart:nameStart+cTCPExitGradeNameWireSize], []byte{0xD7, 0xD4, 0xB6, 0xA8, 0xD2, 0xE5, 0xB3, 0xF6, 0xBF, 0xDA})
}

func TestApplyExitAdditionalTextInfoToBroadcastDataConvertsTextToGBK(t *testing.T) {
	info := defaultExitAdditionalTextInfo()
	info.AdditionalTexts[0] = "一号附加"

	var data StExitAdditionalTextData
	applyExitAdditionalTextInfoToBroadcastData(&data, info)

	assertBytesPrefix(t, data.Additionaldata[:cTCPExitAdditionalTextWireSize], []byte{0xD2, 0xBB, 0xBA, 0xC5, 0xB8, 0xBD, 0xBC, 0xD3})
}

func TestSaveCTCPStGlobalFullJSONConvertsGBKGradeNamesToUTF8Bytes(t *testing.T) {
	var global StGlobal
	copy(global.Grade.StrFruitName[:], []byte{0xC6, 0xBB, 0xB9, 0xFB})
	copy(global.Grade.StrSizeGradeName[:], []byte{0xD2, 0xBB})
	copy(global.Grade.StrQualityGradeName[:], []byte{0xD3, 0xC5, 0xD6, 0xCA})

	jsonText := saveCTCPStGlobalFullJSON(global)
	if jsonText == "" {
		t.Fatal("saveCTCPStGlobalFullJSON() returned empty JSON")
	}

	var decoded struct {
		Grade struct {
			StrFruitName        []uint8
			StrSizeGradeName    []uint8
			StrQualityGradeName []uint8
		}
	}
	if err := json.Unmarshal([]byte(jsonText), &decoded); err != nil {
		t.Fatalf("unmarshal StGlobal JSON: %v", err)
	}

	assertBytesPrefix(t, decoded.Grade.StrFruitName, []byte("苹果"))
	assertBytesPrefix(t, decoded.Grade.StrSizeGradeName, []byte("一"))
	assertBytesPrefix(t, decoded.Grade.StrQualityGradeName, []byte("优质"))
}

func assertBytesPrefix(t *testing.T, got []byte, want []byte) {
	t.Helper()
	if len(got) < len(want) {
		t.Fatalf("got len %d, want at least %d", len(got), len(want))
	}
	for i, value := range want {
		if got[i] != value {
			t.Fatalf("byte[%d] = 0x%02X, want 0x%02X; got prefix=% X want=% X", i, got[i], value, got[:len(want)], want)
		}
	}
}
