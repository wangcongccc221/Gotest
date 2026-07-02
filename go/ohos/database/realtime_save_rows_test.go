package database

import (
	"path/filepath"
	"testing"
	"time"
)

func TestCurrentFruitProcessStartTimeUsesActiveUnfinishedBatch(t *testing.T) {
	resetORMForTest()
	dbPath := filepath.Join(t.TempDir(), "current-fruit-start.db")
	t.Cleanup(resetORMForTest)
	if err := InitORMWithPath(dbPath); err != nil {
		t.Fatalf("InitORMWithPath: %v", err)
	}
	db, err := getInitializedFileORMDB()
	if err != nil {
		t.Fatalf("getInitializedFileORMDB: %v", err)
	}

	rows := []TbFruitInfo{
		{
			CustomerName:   "finished",
			StartTime:      "2026-06-18 08:00:00",
			StartedState:   "1",
			CompletedState: "1",
		},
		{
			CustomerName:   "active",
			StartTime:      "2026-06-18 09:10:11",
			StartedState:   "1",
			CompletedState: "0",
		},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("seed fruit rows: %v", err)
	}

	got, ok, err := CurrentFruitProcessStartTime()
	if err != nil {
		t.Fatalf("CurrentFruitProcessStartTime: %v", err)
	}
	if !ok {
		t.Fatalf("CurrentFruitProcessStartTime ok = false, want true")
	}

	want := time.Date(2026, 6, 18, 9, 10, 11, 0, databaseChinaLocation)
	if !got.Equal(want) {
		t.Fatalf("start time = %s, want %s", got.Format(time.RFC3339), want.Format(time.RFC3339))
	}
}

func TestCurrentFruitProcessStartTimeIgnoresMissingActiveBatch(t *testing.T) {
	resetORMForTest()
	dbPath := filepath.Join(t.TempDir(), "no-current-fruit-start.db")
	t.Cleanup(resetORMForTest)
	if err := InitORMWithPath(dbPath); err != nil {
		t.Fatalf("InitORMWithPath: %v", err)
	}
	db, err := getInitializedFileORMDB()
	if err != nil {
		t.Fatalf("getInitializedFileORMDB: %v", err)
	}

	if err := db.Create(&TbFruitInfo{
		CustomerName:   "finished",
		StartTime:      "2026-06-18 08:00:00",
		StartedState:   "1",
		CompletedState: "1",
	}).Error; err != nil {
		t.Fatalf("seed finished fruit row: %v", err)
	}

	_, ok, err := CurrentFruitProcessStartTime()
	if err != nil {
		t.Fatalf("CurrentFruitProcessStartTime: %v", err)
	}
	if ok {
		t.Fatalf("CurrentFruitProcessStartTime ok = true, want false")
	}
}
