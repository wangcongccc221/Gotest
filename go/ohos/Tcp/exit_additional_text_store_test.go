package tcp

import (
	"strings"
	"testing"
)

func TestExitAdditionalTextConfigValuesMatch48Keys(t *testing.T) {
	info := defaultExitAdditionalTextInfo()
	info.AdditionalTexts[0] = "一号附加"
	info.AdditionalTexts[47] = "四十八号附加"

	values := exitAdditionalTextInfoConfigValues(info)

	if values["出口1附加信息"] != "一号附加" {
		t.Fatalf("出口1附加信息 = %q, want 一号附加", values["出口1附加信息"])
	}
	if values["出口48附加信息"] != "四十八号附加" {
		t.Fatalf("出口48附加信息 = %q, want 四十八号附加", values["出口48附加信息"])
	}
	if _, ok := values["出口49附加信息"]; ok {
		t.Fatal("unexpected 出口49附加信息 key")
	}
}

func TestParseExitAdditionalTextInfoConfigValuesReadsTexts(t *testing.T) {
	values := map[string]string{
		"出口1附加信息": "一号备注",
		"出口3附加信息": "三号备注",
	}

	info := parseExitAdditionalTextInfoConfigValues(values)

	if info.AdditionalTexts[0] != "一号备注" || info.AdditionalTexts[2] != "三号备注" {
		t.Fatalf("AdditionalTexts[0,2] = %q, %q", info.AdditionalTexts[0], info.AdditionalTexts[2])
	}
	if info.AdditionalTexts[1] != "" {
		t.Fatalf("AdditionalTexts[1] = %q, want empty", info.AdditionalTexts[1])
	}
}

func TestApplyExitAdditionalTextInfoToBroadcastDataCopies100ByteSlots(t *testing.T) {
	info := defaultExitAdditionalTextInfo()
	info.AdditionalTexts[0] = strings.Repeat("A", 105)
	info.AdditionalTexts[1] = "B2"

	var data StExitAdditionalTextData
	for i := range data.Additionaldata {
		data.Additionaldata[i] = 'x'
	}

	applyExitAdditionalTextInfoToBroadcastData(&data, info)

	first := string(data.Additionaldata[:100])
	if first != strings.Repeat("A", 100) {
		t.Fatalf("first additional text slot = %q, want first 100 bytes", first)
	}
	second := data.Additionaldata[100:200]
	if second[0] != 'B' || second[1] != '2' {
		t.Fatalf("second additional text prefix = %q, want B2", string(second[:2]))
	}
	for i, value := range second[2:] {
		if value != 0 {
			t.Fatalf("second additional text byte %d = %d, want zero fill", i+2, value)
		}
	}
}
