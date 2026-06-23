package database

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"gorm.io/gorm"
)

const (
	sqlMigrationDir       = "migrations"
	sqlMigrationTableName = "schema_migrations"
)

//go:embed migrations/*.sql
var embeddedSQLMigrations embed.FS

func applySQLMigrations(db *gorm.DB) error {
	return applySQLMigrationsFromFS(db, embeddedSQLMigrations, sqlMigrationDir)
}

func applySQLMigrationsFromFS(db *gorm.DB, fsys fs.FS, dir string) error {
	if err := ensureSQLMigrationTable(db); err != nil {
		return err
	}

	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".sql") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)

	for _, name := range names {
		if err := applySQLMigrationFile(db, fsys, path.Join(dir, name), name); err != nil {
			return err
		}
	}
	return nil
}

func ensureSQLMigrationTable(db *gorm.DB) error {
	return db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
name TEXT PRIMARY KEY,
applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
)`).Error
}

func applySQLMigrationFile(db *gorm.DB, fsys fs.FS, filePath string, name string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Table(sqlMigrationTableName).Where("name = ?", name).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return nil
		}

		data, err := fs.ReadFile(fsys, filePath)
		if err != nil {
			return err
		}

		statements := splitSQLStatements(string(data))
		for index, statement := range statements {
			if strings.TrimSpace(statement) == "" {
				continue
			}
			if err := tx.Exec(statement).Error; err != nil {
				return fmt.Errorf("apply SQL migration %s statement %d: %w", name, index+1, err)
			}
		}

		return tx.Exec(`INSERT INTO schema_migrations(name) VALUES (?)`, name).Error
	})
}

func splitSQLStatements(sqlText string) []string {
	var statements []string
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	inBacktickQuote := false

	for index := 0; index < len(sqlText); index++ {
		ch := sqlText[index]

		if inSingleQuote {
			current.WriteByte(ch)
			if ch == '\\' && index+1 < len(sqlText) {
				index++
				current.WriteByte(sqlText[index])
				continue
			}
			if ch == '\'' {
				if index+1 < len(sqlText) && sqlText[index+1] == '\'' {
					index++
					current.WriteByte(sqlText[index])
					continue
				}
				inSingleQuote = false
			}
			continue
		}

		if inDoubleQuote {
			current.WriteByte(ch)
			if ch == '"' {
				if index+1 < len(sqlText) && sqlText[index+1] == '"' {
					index++
					current.WriteByte(sqlText[index])
					continue
				}
				inDoubleQuote = false
			}
			continue
		}

		if inBacktickQuote {
			current.WriteByte(ch)
			if ch == '`' {
				inBacktickQuote = false
			}
			continue
		}

		if ch == '-' && index+1 < len(sqlText) && sqlText[index+1] == '-' {
			index += 2
			for index < len(sqlText) && sqlText[index] != '\n' {
				index++
			}
			if index < len(sqlText) {
				current.WriteByte('\n')
			}
			continue
		}

		if ch == '#' {
			index++
			for index < len(sqlText) && sqlText[index] != '\n' {
				index++
			}
			if index < len(sqlText) {
				current.WriteByte('\n')
			}
			continue
		}

		if ch == '/' && index+1 < len(sqlText) && sqlText[index+1] == '*' {
			index += 2
			for index+1 < len(sqlText) && !(sqlText[index] == '*' && sqlText[index+1] == '/') {
				index++
			}
			if index+1 < len(sqlText) {
				index++
			}
			continue
		}

		switch ch {
		case '\'':
			inSingleQuote = true
			current.WriteByte(ch)
		case '"':
			inDoubleQuote = true
			current.WriteByte(ch)
		case '`':
			inBacktickQuote = true
			current.WriteByte(ch)
		case ';':
			statement := strings.TrimSpace(current.String())
			if statement != "" {
				statements = append(statements, statement)
			}
			current.Reset()
		default:
			current.WriteByte(ch)
		}
	}

	statement := strings.TrimSpace(current.String())
	if statement != "" {
		statements = append(statements, statement)
	}
	return statements
}
