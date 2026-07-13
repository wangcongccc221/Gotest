package tcp

import "testing"

// 对齐 48 协议(interface.h): 等级数量字段 0x7F 表示"未使用",不是第 127 个等级。
// 回归背景: 重启瞬间 FSM 首帧可能带未初始化的 0x7F/0xFF,旧逻辑钳制成 16
// 会凭空落库 16 组不存在的品质(tb_gradeinfo 出现大量幽灵品质行)。
func TestRealtimeSaveStoredGradeCountSentinel(t *testing.T) {
	cases := []struct {
		in   uint8
		want int
	}{
		{0, 0},
		{1, 1},
		{8, 8},
		{16, 16},
		{17, 16},  // 越界但非哨兵: 按上限钳制
		{126, 16}, // 越界但非哨兵: 按上限钳制
		{0x7F, 0}, // 哨兵: 未使用
		{0x80, 0},
		{0xFF, 0},
	}
	for _, c := range cases {
		if got := realtimeSaveStoredGradeCount(c.in); got != c.want {
			t.Fatalf("realtimeSaveStoredGradeCount(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestBuildRealtimeSaveGradesSentinelQualityCountAsUnused(t *testing.T) {
	var grade StGradeInfo
	grade.NQualityGradeNum = 0x7F // FSM 未初始化/未使用品质分级
	grade.NSizeGradeNum = 4
	var aggregate realtimeSaveAggregate

	items := buildRealtimeSaveGrades(grade, aggregate)
	if len(items) != 4 {
		t.Fatalf("rows = %d, want 4 (品质哨兵应视为未使用,仅按尺寸展开)", len(items))
	}
	for _, item := range items {
		if item.GradeID < 0 || item.GradeID > 3 {
			t.Fatalf("GradeID = %d, want 0..3", item.GradeID)
		}
		if item.QualityName != "" {
			t.Fatalf("GradeID %d QualityName = %q, want empty", item.GradeID, item.QualityName)
		}
		if item.TraitDensity != "" || item.TraitFlaw != "" {
			t.Fatalf("GradeID %d traits = density %q flaw %q, want empty (品质未启用不写特征)", item.GradeID, item.TraitDensity, item.TraitFlaw)
		}
	}
}

func TestBuildRealtimeSaveGradesValidQualityCountUnchanged(t *testing.T) {
	var grade StGradeInfo
	grade.NQualityGradeNum = 2
	grade.NSizeGradeNum = 4
	// 品质启用时特征字段按 48 语义原样落库: 0x7F 存为 "127"(密度未参与分类)
	for i := range grade.Grades {
		grade.Grades[i].SbDensity = 0x7F
	}
	var aggregate realtimeSaveAggregate

	items := buildRealtimeSaveGrades(grade, aggregate)
	if len(items) != 8 {
		t.Fatalf("rows = %d, want 8 (2品质×4尺寸)", len(items))
	}
	maxGradeID := 0
	for _, item := range items {
		if item.GradeID > maxGradeID {
			maxGradeID = item.GradeID
		}
		if item.TraitDensity != "127" {
			t.Fatalf("GradeID %d TraitDensity = %q, want \"127\" (对齐48存原值)", item.GradeID, item.TraitDensity)
		}
	}
	if maxGradeID != cTCPServerMaxSizeGradeNum+3 {
		t.Fatalf("max GradeID = %d, want %d (第二品质组从16起)", maxGradeID, cTCPServerMaxSizeGradeNum+3)
	}
}
