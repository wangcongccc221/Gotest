package database

import (
	"path/filepath"
	"testing"
	"time"
)

func TestQuerySortLogGroupsRunningTimeByDate(t *testing.T) {
	resetORMForTest()
	dbPath := filepath.Join(t.TempDir(), "sort-log.db")
	t.Cleanup(resetORMForTest)
	if err := InitORMWithPath(dbPath); err != nil {
		t.Fatalf("InitORMWithPath: %v", err)
	}
	db, err := getInitializedFileORMDB()
	if err != nil {
		t.Fatalf("getInitializedFileORMDB: %v", err)
	}

	rows := []TbRunningTimeInfo{
		{RunningDate: "2026-06-17", StartTime: "08:00:00", StopTime: "08:30:00"},
		{RunningDate: "2026-06-17", StartTime: "09:00:00", StopTime: "10:15:00"},
		{RunningDate: "2026-06-18", StartTime: "11:00:00", StopTime: ""},
		{RunningDate: "2026-06-19", StartTime: "12:00:00", StopTime: "12:20:00"},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("seed running time rows: %v", err)
	}

	result, err := QuerySortLog(SortLogQueryRequest{
		StartDate: "2026-06-17",
		EndDate:   "2026-06-18",
	})
	if err != nil {
		t.Fatalf("QuerySortLog: %v", err)
	}

	if result.TotalMinutes != 105 {
		t.Fatalf("TotalMinutes = %d, want 105", result.TotalMinutes)
	}
	if len(result.Items) != 2 {
		t.Fatalf("len(Items) = %d, want 2: %#v", len(result.Items), result.Items)
	}
	if result.Items[0].RunningDate != "2026-06-17" || result.Items[0].TotalMinutes != 105 {
		t.Fatalf("first item = %#v, want 2026-06-17 with 105 minutes", result.Items[0])
	}
	if len(result.Items[0].Segments) != 2 {
		t.Fatalf("first item segments = %d, want 2", len(result.Items[0].Segments))
	}
	if result.Items[0].Segments[1].StartMinute != 540 || result.Items[0].Segments[1].StopMinute != 615 {
		t.Fatalf("second segment = %#v, want 09:00-10:15", result.Items[0].Segments[1])
	}
	if result.Items[1].RunningDate != "2026-06-18" || result.Items[1].TotalMinutes != 0 || len(result.Items[1].Segments) != 0 {
		t.Fatalf("second item = %#v, want empty day shell for 2026-06-18", result.Items[1])
	}
}

func TestReportSortRunningStateOpensAndClosesRunningTime(t *testing.T) {
	resetORMForTest()
	dbPath := filepath.Join(t.TempDir(), "sort-running-state.db")
	t.Cleanup(resetORMForTest)
	if err := InitORMWithPath(dbPath); err != nil {
		t.Fatalf("InitORMWithPath: %v", err)
	}
	db, err := getInitializedFileORMDB()
	if err != nil {
		t.Fatalf("getInitializedFileORMDB: %v", err)
	}

	startAt := time.Date(2026, 6, 18, 9, 10, 11, 0, databaseChinaLocation)
	result, err := ReportSortRunningStateAt(SortRunningStateRequest{
		SortSpeedPerMinute: 120,
	}, startAt)
	if err != nil {
		t.Fatalf("ReportSortRunningStateAt start: %v", err)
	}
	if result.Action != "opened" || result.OpenRecordID == 0 {
		t.Fatalf("start result = %#v, want opened with record id", result)
	}

	againAt := startAt.Add(30 * time.Second)
	result, err = ReportSortRunningStateAt(SortRunningStateRequest{
		SortSpeedPerMinute: 88,
	}, againAt)
	if err != nil {
		t.Fatalf("ReportSortRunningStateAt duplicate: %v", err)
	}
	if result.Action != "keptOpen" || result.OpenRecordID == 0 {
		t.Fatalf("duplicate result = %#v, want keptOpen", result)
	}

	stopAt := time.Date(2026, 6, 18, 9, 30, 0, 0, databaseChinaLocation)
	result, err = ReportSortRunningStateAt(SortRunningStateRequest{
		SortSpeedPerMinute: 0,
	}, stopAt)
	if err != nil {
		t.Fatalf("ReportSortRunningStateAt stop: %v", err)
	}
	if result.Action != "closed" || result.ClosedRecordID == 0 {
		t.Fatalf("stop result = %#v, want closed", result)
	}

	var rows []TbRunningTimeInfo
	if err := db.Order("ID asc").Find(&rows).Error; err != nil {
		t.Fatalf("query rows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows len = %d, want 1: %#v", len(rows), rows)
	}
	if rows[0].RunningDate != "2026-06-18" || rows[0].StartTime != "09:10:11" || rows[0].StopTime != "09:30:00" {
		t.Fatalf("row = %#v, want 2026-06-18 09:10:11-09:30:00", rows[0])
	}
}

func TestReportSortRunningStateClosesPreviousDayAndOpensToday(t *testing.T) {
	resetORMForTest()
	dbPath := filepath.Join(t.TempDir(), "sort-running-cross-day.db")
	t.Cleanup(resetORMForTest)
	if err := InitORMWithPath(dbPath); err != nil {
		t.Fatalf("InitORMWithPath: %v", err)
	}
	db, err := getInitializedFileORMDB()
	if err != nil {
		t.Fatalf("getInitializedFileORMDB: %v", err)
	}
	openRow := TbRunningTimeInfo{RunningDate: "2026-06-17", StartTime: "23:50:00", StopTime: ""}
	if err := db.Create(&openRow).Error; err != nil {
		t.Fatalf("seed open row: %v", err)
	}

	result, err := ReportSortRunningStateAt(SortRunningStateRequest{
		SortSpeedPerMinute: 60,
	}, time.Date(2026, 6, 18, 0, 0, 10, 0, databaseChinaLocation))
	if err != nil {
		t.Fatalf("ReportSortRunningStateAt cross day: %v", err)
	}
	if result.Action != "closedAndOpened" || result.ClosedRecordID == 0 || result.OpenRecordID == 0 {
		t.Fatalf("cross day result = %#v, want closedAndOpened", result)
	}

	var rows []TbRunningTimeInfo
	if err := db.Order("ID asc").Find(&rows).Error; err != nil {
		t.Fatalf("query rows: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows len = %d, want 2: %#v", len(rows), rows)
	}
	if rows[0].StopTime != "23:59:59" {
		t.Fatalf("previous day StopTime = %q, want 23:59:59", rows[0].StopTime)
	}
	if rows[1].RunningDate != "2026-06-18" || rows[1].StartTime != "00:00:10" || rows[1].StopTime != "" {
		t.Fatalf("today row = %#v, want open row at 00:00:10", rows[1])
	}
}
