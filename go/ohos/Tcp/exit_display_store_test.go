package tcp

import "testing"

func TestExitDisplayInfoConfigValuesMatch48Keys(t *testing.T) {
	info := defaultExitDisplayInfo()
	info.DisplayType = 3
	info.DisplayNames[0] = "出口1"
	info.DisplayNames[47] = "出口48"

	values := exitDisplayInfoConfigValues(info)

	if values[cTCPExitDisplayTypeConfigName] != "3" {
		t.Fatalf("%s = %q, want 3", cTCPExitDisplayTypeConfigName, values[cTCPExitDisplayTypeConfigName])
	}
	if values["出口1显示名称"] != "出口1" {
		t.Fatalf("出口1显示名称 = %q, want 出口1", values["出口1显示名称"])
	}
	if values["出口48显示名称"] != "出口48" {
		t.Fatalf("出口48显示名称 = %q, want 出口48", values["出口48显示名称"])
	}
	if _, ok := values["出口49显示名称"]; ok {
		t.Fatal("unexpected 出口49显示名称 key")
	}
}

func TestParseExitDisplayInfoConfigValuesReadsDisplayTypeAndNames(t *testing.T) {
	values := map[string]string{
		cTCPExitDisplayTypeConfigName: "5",
		"出口1显示名称":                     "一号屏",
		"出口3显示名称":                     "三号屏",
	}

	info, err := parseExitDisplayInfoConfigValues(values)
	if err != nil {
		t.Fatalf("parseExitDisplayInfoConfigValues() error = %v", err)
	}
	if info.DisplayType != 5 {
		t.Fatalf("DisplayType = %d, want 5", info.DisplayType)
	}
	if info.DisplayNames[0] != "一号屏" || info.DisplayNames[2] != "三号屏" {
		t.Fatalf("DisplayNames[0,2] = %q, %q", info.DisplayNames[0], info.DisplayNames[2])
	}
	if info.DisplayNames[1] != "" {
		t.Fatalf("DisplayNames[1] = %q, want empty", info.DisplayNames[1])
	}
}

func TestSetExitDisplayNameEnabledMatchesCppBitSemantics(t *testing.T) {
	var displayType int64

	displayType = setExitDisplayNameEnabled(displayType, 0, true)
	displayType = setExitDisplayNameEnabled(displayType, 47, true)
	if displayType != (int64(1) | (int64(1) << 47)) {
		t.Fatalf("DisplayType = %d, want bits 0 and 47 set", displayType)
	}
	if !exitDisplayNameEnabled(displayType, 0) || !exitDisplayNameEnabled(displayType, 47) {
		t.Fatalf("display name bits not detected: %d", displayType)
	}

	displayType = setExitDisplayNameEnabled(displayType, 0, false)
	if exitDisplayNameEnabled(displayType, 0) {
		t.Fatalf("exit 0 display name bit still enabled: %d", displayType)
	}
	if !exitDisplayNameEnabled(displayType, 47) {
		t.Fatalf("exit 47 display name bit was cleared unexpectedly: %d", displayType)
	}
}

func TestApplyExitDisplayInfoToBroadcastSysConfigCopies20ByteSlots(t *testing.T) {
	info := defaultExitDisplayInfo()
	info.DisplayType = 1
	info.DisplayNames[0] = "ABCDEFGHIJKLMNOPQRSTUV"
	info.DisplayNames[1] = "B2"

	var config StBroadcastSysConfig
	for i := range config.StrDisplayName {
		config.StrDisplayName[i] = 'x'
	}

	applyExitDisplayInfoToBroadcastSysConfig(&config, info)

	if config.ExitDisplayType != 1 {
		t.Fatalf("ExitDisplayType = %d, want 1", config.ExitDisplayType)
	}
	first := string(config.StrDisplayName[:20])
	if first != "ABCDEFGHIJKLMNOPQRST" {
		t.Fatalf("first display name slot = %q, want first 20 bytes", first)
	}
	second := config.StrDisplayName[20:40]
	if second[0] != 'B' || second[1] != '2' {
		t.Fatalf("second display name prefix = %q, want B2", string(second[:2]))
	}
	for i, value := range second[2:] {
		if value != 0 {
			t.Fatalf("second display name byte %d = %d, want zero fill", i+2, value)
		}
	}
}

func TestParseExitDisplayEnabledConfigValueMatches48ContainsTrue(t *testing.T) {
	if !parseExitDisplayEnabledConfigValue("true") {
		t.Fatal("true should enable exit display broadcast")
	}
	if !parseExitDisplayEnabledConfigValue("true-from-db") {
		t.Fatal("48 uses contains(\"true\"), so true-from-db should enable")
	}
	if parseExitDisplayEnabledConfigValue("false") {
		t.Fatal("false should disable exit display broadcast")
	}
}

func TestBuildExitDisplayBroadcastPacketUsesCustomNameWhenBitEnabled(t *testing.T) {
	info := defaultExitDisplayInfo()
	info.DisplayType = 1
	info.DisplayNames[0] = "自定义出口"

	packet := buildExitDisplayBroadcastPacket(info, StGradeInfo{}, false)

	if len(packet) != 1+cTCP48MaxExitNum*cTCPExitGradeItemWireSize+1 {
		t.Fatalf("packet len = %d, want %d", len(packet), 1+cTCP48MaxExitNum*cTCPExitGradeItemWireSize+1)
	}
	if packet[0] != 0xAA || packet[len(packet)-1] != 0x55 {
		t.Fatalf("packet frame = 0x%02X ... 0x%02X, want 0xAA ... 0x55", packet[0], packet[len(packet)-1])
	}
	if got := int(packet[1]) | int(packet[2])<<8 | int(packet[3])<<16 | int(packet[4])<<24; got != 1 {
		t.Fatalf("first ExitIndex = %d, want 1", got)
	}
	name := string(packet[5 : 5+len("自定义出口")])
	if name != "自定义出口" {
		t.Fatalf("first GradeName prefix = %q, want 自定义出口", name)
	}
}

func TestBuildExitDisplayBroadcastPacketFallsBackToGradeNames(t *testing.T) {
	grade := StGradeInfo{}
	grade.NSizeGradeNum = 2
	grade.NQualityGradeNum = 1
	copy(grade.StrSizeGradeName[:], []byte("大果"))
	copy(grade.StrSizeGradeName[cTCPServerMaxTextLength:], []byte("中果"))
	copy(grade.StrQualityGradeName[:], []byte("A级"))
	grade.Grades[1].ExitLow = 1

	packet := buildExitDisplayBroadcastPacket(defaultExitDisplayInfo(), grade, true)

	nameStart := 1 + 4
	name := string(packet[nameStart : nameStart+len("中果.A级")])
	if name != "中果.A级" {
		t.Fatalf("first GradeName prefix = %q, want 中果.A级", name)
	}
}
