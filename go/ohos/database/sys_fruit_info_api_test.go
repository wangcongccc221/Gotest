package database

import (
	"path/filepath"
	"testing"
)

func TestQuerySysFruitInfoFiltersAndOrdersRows(t *testing.T) {
	resetORMForTest()
	dbPath := filepath.Join(t.TempDir(), "sys-fruit-info.db")
	t.Cleanup(resetORMForTest)
	if err := InitORMWithPath(dbPath); err != nil {
		t.Fatalf("InitORMWithPath: %v", err)
	}
	db, err := getInitializedFileORMDB()
	if err != nil {
		t.Fatalf("getInitializedFileORMDB: %v", err)
	}

	rows := []TbSysFruitInfo{
		{CustomerID: 1, SystemID: 1, BatchNumber: 10, BatchWeight: 100},
		{CustomerID: 2, SystemID: 1, BatchNumber: 20, BatchWeight: 200},
		{CustomerID: 2, SystemID: 2, BatchNumber: 30, BatchWeight: 300},
		{CustomerID: 3, SystemID: 1, BatchNumber: 40, BatchWeight: 400},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("seed sys fruit info rows: %v", err)
	}

	systemID := 1
	pageSize := 2
	result, err := QuerySysFruitInfo(SysFruitInfoQueryRequest{
		SystemID:  &systemID,
		PageSize:  &pageSize,
		SortOrder: stringPtrForTest("desc"),
	})
	if err != nil {
		t.Fatalf("QuerySysFruitInfo: %v", err)
	}
	if result.TotalCount != 3 {
		t.Fatalf("TotalCount = %d, want 3", result.TotalCount)
	}
	if len(result.Items) != 2 {
		t.Fatalf("len(Items) = %d, want 2", len(result.Items))
	}
	if result.Items[0].CustomerID != 3 || result.Items[0].BatchNumber != 40 {
		t.Fatalf("first row = %#v, want latest customer for system 1", result.Items[0])
	}
	if result.Items[1].CustomerID != 2 || result.Items[1].BatchNumber != 20 {
		t.Fatalf("second row = %#v, want second latest customer for system 1", result.Items[1])
	}
}

func stringPtrForTest(value string) *string {
	return &value
}
