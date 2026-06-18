package database

import (
	"path/filepath"
	"testing"
)

func TestDeleteFruitInfoByCustomerIDsSoftDeletesPositiveIDsOnly(t *testing.T) {
	resetORMForTest()
	dbPath := filepath.Join(t.TempDir(), "history-delete.db")
	t.Cleanup(resetORMForTest)
	if err := InitORMWithPath(dbPath); err != nil {
		t.Fatalf("InitORMWithPath: %v", err)
	}
	db, err := getInitializedFileORMDB()
	if err != nil {
		t.Fatalf("getInitializedFileORMDB: %v", err)
	}

	visible := 1
	rows := []TbFruitInfo{
		{CustomerID: 101, CustomerName: "keep", FVisible: &visible},
		{CustomerID: 102, CustomerName: "delete-a", FVisible: &visible},
		{CustomerID: 103, CustomerName: "delete-b", FVisible: &visible},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("seed fruit rows: %v", err)
	}

	deletedRows, err := DeleteFruitInfoByCustomerIDs([]int{102, 0, -7, 103})
	if err != nil {
		t.Fatalf("DeleteFruitInfoByCustomerIDs: %v", err)
	}
	if deletedRows != 2 {
		t.Fatalf("deletedRows = %d, want 2", deletedRows)
	}

	var keep TbFruitInfo
	if err := db.Where("CustomerID = ?", 101).First(&keep).Error; err != nil {
		t.Fatalf("read keep row: %v", err)
	}
	if keep.FVisible == nil || *keep.FVisible != 1 {
		t.Fatalf("keep FVisible = %#v, want 1", keep.FVisible)
	}

	for _, customerID := range []int{102, 103} {
		var row TbFruitInfo
		if err := db.Where("CustomerID = ?", customerID).First(&row).Error; err != nil {
			t.Fatalf("read deleted row %d: %v", customerID, err)
		}
		if row.FVisible == nil || *row.FVisible != 0 {
			t.Fatalf("CustomerID %d FVisible = %#v, want 0", customerID, row.FVisible)
		}
	}
}

func TestDeleteFruitInfoByCustomerIDsIgnoresEmptyInput(t *testing.T) {
	deletedRows, err := DeleteFruitInfoByCustomerIDs([]int{0, -1})
	if err != nil {
		t.Fatalf("DeleteFruitInfoByCustomerIDs empty: %v", err)
	}
	if deletedRows != 0 {
		t.Fatalf("deletedRows = %d, want 0", deletedRows)
	}
}
