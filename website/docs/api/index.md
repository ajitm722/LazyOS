---
sidebar_position: 1
---

# API Reference

The LazyOS API documentation is auto-generated from Go source code comments and package documentation. This section provides comprehensive reference material for all public packages and types.

## Packages

| Package | Description |
|---|---|
| `cmd/lazyos` | Application entry point and CLI wiring |
| `internal/cache` | Lazy-loading query cache decorator |
| `internal/config` | Configuration types and Viper integration |
| `internal/daemons` | Domain interfaces (`Queryer`), `TableSchema`, and column helpers |
| `internal/daemons/osqueryd` | Thrift RPC client for osqueryd communication |
| `internal/daemons/osqueryd/kernel` | Kernel table schema catalog and `Queryer` implementation |
| `internal/daemons/osqueryd/aws` | AWS table schema catalog and `Queryer` implementation |
| `internal/logger` | Structured JSON logging via `slog` |
| `internal/mcp` | Model Context Protocol server implementation |
| `internal/store/sqlite` | SQLite cache backend with WAL journal mode |
| `internal/tui` | Bubble Tea TUI application core |
| `internal/tui/views/querybar` | SQL query input component |
| `internal/tui/views/results` | Results display (line mode + table mode) |
| `internal/tui/views/sidebar` | Table browser sidebar component |

## Key Types

### `daemons.Queryer`

```go
type Queryer interface {
    Query(ctx context.Context, sql string) (rows []map[string]string, columns []string, err error)
    Close() error
    GetSchema() []TableSchema
}
```

The core domain interface. All backends (kernel, AWS) and the cache decorator implement this contract.

### `daemons.TableSchema`

```go
type TableSchema struct {
    Name        string
    Description string
    Columns     string
}
```

Schema catalog entry for a single osquery table.

### `cache.CachedQueryer`

```go
type CachedQueryer struct {
    // implements daemons.Queryer
}

func NewCachedQueryer(upstream daemons.Queryer, store *sqlite.SQLiteStore) *CachedQueryer
func (c *CachedQueryer) Query(ctx context.Context, sql string) ([]map[string]string, []string, error)
func (c *CachedQueryer) QuerySource(ctx context.Context, sql string) ([]map[string]string, []string, error)
func (c *CachedQueryer) GetSchema() []daemons.TableSchema
func (c *CachedQueryer) Close() error
```

Decorator that transparently adds SQLite persistence to any upstream `Queryer`.

### `sqlite.SQLiteStore`

```go
type SQLiteStore struct{}

func Open(dbPath string) (*SQLiteStore, error)
func (s *SQLiteStore) Query(ctx context.Context, sql string) ([]map[string]string, []string, error)
func (s *SQLiteStore) SyncTable(name string, columns []string, rows []map[string]string) error
func (s *SQLiteStore) HasTable(name string) bool
func (s *SQLiteStore) Health(ctx context.Context) error
func (s *SQLiteStore) Close() error
```

Embedded cache backend. WAL journal mode, `TEXT`-only values, atomic `SyncTable` transactions.

---

For detailed package-level documentation, visit [pkg.go.dev](https://pkg.go.dev/github.com/ajitm722/LazyOS) or generate docs locally with `go doc`.

```bash
# View docs for a specific package
go doc github.com/ajitm722/LazyOS/internal/cache

# View docs for a specific type
go doc github.com/ajitm722/LazyOS/internal/cache.CachedQueryer

# Start a local doc server
godoc -http=:6060
```
