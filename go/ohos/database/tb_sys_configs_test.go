package database

import (
	"path/filepath"
	"testing"
)

func TestConfigValueAccessIgnoresStringCreateDate(t *testing.T) {
	resetORMForTest()

	tempDir := t.TempDir()
	t.Cleanup(resetORMForTest)

	if err := InitORMWithPath(filepath.Join(tempDir, "config.db")); err != nil {
		t.Fatalf("InitORMWithPath() error = %v", err)
	}
	db, err := getORMDB()
	if err != nil {
		t.Fatalf("getORMDB() error = %v", err)
	}

	if err := db.Exec(
		`INSERT INTO tb_sys_configs (FType, FValue, FModuleName, FVisible, FCreateDate) VALUES (?, ?, ?, ?, ?)`,
		"已选水果种类",
		"1.1-新鲜脐橙;",
		"RSS",
		0,
		"2026-06-15 17:23:59",
	).Error; err != nil {
		t.Fatalf("insert config with string FCreateDate error = %v", err)
	}

	value, err := GetConfigValue("已选水果种类")
	if err != nil {
		t.Fatalf("GetConfigValue() error = %v", err)
	}
	if value != "1.1-新鲜脐橙;" {
		t.Fatalf("GetConfigValue() = %q, want %q", value, "1.1-新鲜脐橙;")
	}

	if err := SaveConfigValue("已选水果种类", "3.1-安岳柠檬;"); err != nil {
		t.Fatalf("SaveConfigValue() error = %v", err)
	}

	value, err = GetConfigValue("已选水果种类")
	if err != nil {
		t.Fatalf("GetConfigValue() after save error = %v", err)
	}
	if value != "3.1-安岳柠檬;" {
		t.Fatalf("GetConfigValue() after save = %q, want %q", value, "3.1-安岳柠檬;")
	}
}
