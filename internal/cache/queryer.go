// Package cache provides a transparent Queryer decorator that lazily fetches
// tables from an upstream data source on first access and persists them in a
// local SQLite store. Subsequent queries against cached tables are served from
// the local store without touching the upstream. Use QuerySource to bypass the
// cache and refresh tables from the upstream.
package cache

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ajitm722/LazyOS/internal/daemons"
	"github.com/ajitm722/LazyOS/internal/store/sqlite"
)

// CachedQueryer wraps an upstream daemons.Queryer and a local SQLiteStore to
// provide lazy-loaded, cached query execution. On the first Query call that
// references a table, it fetches the full table from the upstream and syncs
// it into the store. Subsequent queries against that table are served from
// the store without touching the upstream.
type CachedQueryer struct {
	upstream daemons.Queryer
	store    *sqlite.SQLiteStore
}

// compile-time interface check
var _ daemons.Queryer = (*CachedQueryer)(nil)

// NewCachedQueryer creates a CachedQueryer that delegates to upstream for
// live data and persists results in st. If st is nil, Query and QuerySource
// fall through to the upstream directly (no caching).
func NewCachedQueryer(upstream daemons.Queryer, st *sqlite.SQLiteStore) *CachedQueryer {
	return &CachedQueryer{upstream: upstream, store: st}
}

// Query executes sql against the local store. Any table referenced in the
// query that is not yet present in the store is fetched first from the
// upstream via fetchTable and synced in. Subsequent queries against the same
// table bypass the upstream entirely.
func (c *CachedQueryer) Query(ctx context.Context, sql string) ([]map[string]string, []string, error) {
	slog.Debug("cached query", "sql", sql)
	if c.store == nil {
		return c.upstream.Query(ctx, sql)
	}

	tables := extractTableNames(sql)
	for _, t := range tables {
		if !c.store.HasTable(t) {
			slog.Info("lazy-loading uncached table", "table", t)
			if err := c.fetchTable(ctx, t); err != nil {
				slog.Error("failed to lazy-load table", "table", t, "error", err)
				return nil, nil, fmt.Errorf("lazy-load table %s: %w", t, err)
			}
		}
	}

	return c.store.Query(ctx, sql)
}

// QuerySource fetches fresh data for every table referenced in sql from the
// upstream, syncs the results into the store, then runs the query locally.
// Use this when the caller needs authoritative data from the live source.
func (c *CachedQueryer) QuerySource(ctx context.Context, sql string) ([]map[string]string, []string, error) {
	slog.Info("source query", "sql", sql)
	if c.store == nil {
		return c.upstream.Query(ctx, sql)
	}

	tables := extractTableNames(sql)
	for _, t := range tables {
		if err := c.fetchTable(ctx, t); err != nil {
			slog.Error("failed to refresh table", "table", t, "error", err)
			return nil, nil, fmt.Errorf("refresh table %s: %w", t, err)
		}
	}

	return c.store.Query(ctx, sql)
}

// fetchTable queries the upstream for every row in tableName and syncs the
// result set into the local store, replacing any existing cached copy.
func (c *CachedQueryer) fetchTable(ctx context.Context, tableName string) error {
	sql := fmt.Sprintf("SELECT * FROM %s", tableName)
	rows, cols, err := c.upstream.Query(ctx, sql)
	if err != nil {
		return err
	}
	return c.store.SyncTable(tableName, cols, rows)
}

// Close releases the upstream connection.
func (c *CachedQueryer) Close() error {
	return c.upstream.Close()
}

// GetSchema returns the table schema catalog from the upstream.
func (c *CachedQueryer) GetSchema() []daemons.TableSchema {
	return c.upstream.GetSchema()
}
