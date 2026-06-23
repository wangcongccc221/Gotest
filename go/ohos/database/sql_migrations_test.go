package database

import (
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestSplitSQLStatementsKeepsSemicolonsInsideStrings(t *testing.T) {
	sqlText := `
CREATE TABLE sample (id INTEGER PRIMARY KEY, value TEXT);
INSERT INTO sample(id, value) VALUES (1, 'g;斤;Kg;lb;oz;');
-- comment with ; should not create an empty statement
INSERT INTO sample(id, value) VALUES (2, 'done');
`

	statements := splitSQLStatements(sqlText)
	if len(statements) != 3 {
		t.Fatalf("len(splitSQLStatements) = %d, want 3: %#v", len(statements), statements)
	}
	if !strings.Contains(statements[1], "'g;斤;Kg;lb;oz;'") {
		t.Fatalf("second statement lost quoted semicolons: %q", statements[1])
	}
}

func TestApplySQLMigrationsFromFSRunsEachMigrationOnce(t *testing.T) {
	db := openMigrationTestDB(t)
	fsys := fstest.MapFS{
		"migrations/001_init.sql": {
			Data: []byte(`
CREATE TABLE IF NOT EXISTS sample (id INTEGER PRIMARY KEY, value TEXT);
INSERT INTO sample(id, value) VALUES (1, 'one');
`),
		},
	}

	if err := applySQLMigrationsFromFS(db, fsys, "migrations"); err != nil {
		t.Fatalf("first applySQLMigrationsFromFS: %v", err)
	}
	if err := applySQLMigrationsFromFS(db, fsys, "migrations"); err != nil {
		t.Fatalf("second applySQLMigrationsFromFS: %v", err)
	}

	var rowCount int64
	if err := db.Table("sample").Count(&rowCount).Error; err != nil {
		t.Fatalf("count sample rows: %v", err)
	}
	if rowCount != 1 {
		t.Fatalf("sample row count = %d, want 1", rowCount)
	}

	var migrationCount int64
	if err := db.Table(sqlMigrationTableName).Where("name = ?", "001_init.sql").Count(&migrationCount).Error; err != nil {
		t.Fatalf("count migration rows: %v", err)
	}
	if migrationCount != 1 {
		t.Fatalf("migration row count = %d, want 1", migrationCount)
	}
}

func TestApplySQLMigrationsFromFSDoesNotRecordFailedMigration(t *testing.T) {
	db := openMigrationTestDB(t)
	fsys := fstest.MapFS{
		"migrations/001_fail.sql": {
			Data: []byte(`
CREATE TABLE IF NOT EXISTS sample (id INTEGER PRIMARY KEY, value TEXT);
INSERT INTO missing_table(id, value) VALUES (1, 'one');
`),
		},
	}

	if err := applySQLMigrationsFromFS(db, fsys, "migrations"); err == nil {
		t.Fatal("applySQLMigrationsFromFS succeeded, want error")
	}

	var migrationCount int64
	if err := db.Table(sqlMigrationTableName).Where("name = ?", "001_fail.sql").Count(&migrationCount).Error; err != nil {
		t.Fatalf("count migration rows: %v", err)
	}
	if migrationCount != 0 {
		t.Fatalf("failed migration row count = %d, want 0", migrationCount)
	}
}

func openMigrationTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "migrations.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite test db: %v", err)
	}
	return db
}
