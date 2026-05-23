package daemons

import (
	"context"
	"errors"
)

// Standardized errors that all Queryer implementations should return.
// These allow the TUI to handle specific failure states consistently.
var (
	ErrQueryTimeout = errors.New("query timed out")
	ErrQueryFailed  = errors.New("query failed")
)

// TableSchema holds the generic metadata for any backend's table.
type TableSchema struct {
	Name        string
	Description string
	Columns     string
}

// Queryer defines the interface for all backend data sources.
// Query returns the result rows, the ordered column names, and any error.
// Columns must be populated even when rows is empty so the UI can render
// column headers in table mode.
type Queryer interface {
	Query(ctx context.Context, sql string) (rows []map[string]string, columns []string, err error)
	Close() error
	GetSchema() []TableSchema // Enables dynamic sidebar rendering
}
