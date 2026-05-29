package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStore implements a local persistent table cache backed by a SQLite
// database file. Tables are stored as SQL tables with TEXT columns matching
// the column names returned by the upstream data source.
type SQLiteStore struct {
	db *sql.DB
}

// Open creates or opens the SQLite database at dbPath, enabling WAL journal
// mode and a 5-second busy timeout to handle concurrent read/write contention.
func Open(dbPath string) (*SQLiteStore, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("create store directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Query(ctx context.Context, sqlQuery string) ([]map[string]string, []string, error) {
	rows, err := s.db.QueryContext(ctx, sqlQuery)
	if err != nil {
		return nil, nil, fmt.Errorf("query sqlite: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, nil, fmt.Errorf("get columns: %w", err)
	}

	var result []map[string]string
	for rows.Next() {
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, nil, fmt.Errorf("scan row: %w", err)
		}

		row := make(map[string]string, len(cols))
		for i, col := range cols {
			val := values[i]
			if val == nil {
				row[col] = ""
			} else {
				row[col] = fmt.Sprintf("%v", val)
			}
		}
		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("rows iteration: %w", err)
	}

	if result == nil {
		result = []map[string]string{}
	}

	return result, cols, nil
}

func (s *SQLiteStore) SyncTable(name string, columns []string, rows []map[string]string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	quoted := quoteIdent(name)
	if _, err := tx.Exec("DROP TABLE IF EXISTS " + quoted); err != nil {
		return fmt.Errorf("drop table %s: %w", name, err)
	}

	var colDefs []string
	for _, col := range columns {
		colDefs = append(colDefs, quoteIdent(col)+" TEXT")
	}
	createSQL := "CREATE TABLE " + quoted + " (" + strings.Join(colDefs, ", ") + ")"
	if _, err := tx.Exec(createSQL); err != nil {
		return fmt.Errorf("create table %s: %w", name, err)
	}

	if len(rows) == 0 {
		return tx.Commit()
	}

	var placeholders []string
	var quotedCols []string
	for _, col := range columns {
		placeholders = append(placeholders, "?")
		quotedCols = append(quotedCols, quoteIdent(col))
	}
	insertSQL := "INSERT INTO " + quoted + " (" + strings.Join(quotedCols, ", ") + ") VALUES (" + strings.Join(placeholders, ", ") + ")"

	stmt, err := tx.Prepare(insertSQL)
	if err != nil {
		return fmt.Errorf("prepare insert: %w", err)
	}
	defer stmt.Close()

	for _, row := range rows {
		args := make([]interface{}, len(columns))
		for i, col := range columns {
			args[i] = row[col]
		}
		if _, err := stmt.Exec(args...); err != nil {
			return fmt.Errorf("insert row: %w", err)
		}
	}

	return tx.Commit()
}

func (s *SQLiteStore) HasTable(name string) bool {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", name).Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}

func (s *SQLiteStore) Health(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// quoteIdent wraps an SQL identifier in double-quotes, escaping any internal
// double-quote characters.
func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}
