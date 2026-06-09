---
sidebar_position: 4
---

# Internal Package Structure

The project follows standard Go layout conventions with strict separation of concerns.

## `cmd/lazyos/`

The execution entry point. Uses Cobra for CLI and Viper for config. `runApp()` manages the full lifecycle: logger initialization, backend bootstrap, opening the SQLite store, wrapping each backend in a `CachedQueryer`, and launching the TUI. No sync or prefetch commands exist — the cache is lazily populated on first query.

## `internal/config/`

Configuration types serialized from YAML or CLI flags. Contains `Config`, `Keys` (with `execute_source` for the `E` key binding), and `CacheDBPath` for the SQLite database path override.

## `internal/cache/`

Implements the lazy-loading query cache.

- `queryer.go`: Defines the `CachedQueryer` struct, its constructor, and the core `Query`, `QuerySource`, `fetchTable`, `Close`, and `GetSchema` methods. This file contains only the decorator logic — SQL parsing lives in the separate `sql_parser.go`.
- `sql_parser.go`: Contains all SQL text-parsing utilities: `extractTableNames` (extracts table names from `FROM`/`JOIN` clauses), `parseFromClause`, `isJoinWord`, `isQueryKeyword`, `isWordChar`, and `keywordNormalize`. These helpers have no dependencies on the cache or store types.

## `internal/store/sqlite/`

Concrete SQLite implementation of the cache backend. Uses `mattn/go-sqlite3` (CGo driver) with WAL journal mode and a 5-second busy timeout. Exports `SQLiteStore` with methods `Query`, `SyncTable`, `HasTable`, `Health`, and `Close`. `SyncTable` atomically replaces a table's full contents in a single transaction. Tests require the `integration` build tag (`go test -tags=integration`).

## `internal/tui/`

The core Bubble Tea application.

- `app.go`: Owns `AppModel` — the single source of truth for TUI state. Routes `RunQueryMsg` (cached) and `RunSourceQueryMsg` (source refresh) to the appropriate path. Type-asserts the active client to `*cache.CachedQueryer` to call `QuerySource` when the user presses `E`.
- `actions.go`: Defines all `AppAction` implementations. `ExecuteAction` emits `RunQueryMsg` for the cached path. `ExecuteSourceAction` emits `RunSourceQueryMsg` for the source-refresh path.
- `registry.go`: Maps key bindings to actions. `e` → `ExecuteAction`, `E` → `ExecuteSourceAction`.
- `layout.go`: Owns `Layout`, lipgloss rendering, pane dimension math, and the terminal-too-small guard.
- `views/`: Isolated UI components (sidebar, querybar, results).

## `internal/daemons/`

Defines the `Queryer` interface, `TableSchema` domain model, and sentinel errors (`ErrQueryTimeout`, `ErrQueryFailed`). Contains `columns.go` with three helpers: `ExtractColumnNames` (parses `"pid, name"` into `["pid", "name"]`), `DeriveColumnsFromSchema` (extracts column names from the schema for a given SQL `FROM` table), and `AutofillColumns` (returns a comma-separated column string or `"*"` for sidebar autofill). This package acts purely as the domain contract — no backend or cache implementations live here.

## `internal/daemons/mock/`

Provides `MockQueryer` — a fully instrumented `Queryer` stub for tests. Supports simulated slow queries, configurable error injection, and custom result sets per SQL string.

## `internal/daemons/osqueryd/`

Low-level Thrift RPC communication with the osquery daemon. The `Client` struct wraps `goosquery.ExtensionManagerClient` and exposes `queryTimeout`-bounded `Query` calls. The `executeThriftQuery` method uses `QueryRowsContext` instead of `Query`, passing the deadline-bearing context through to the osquery-go locker. This prevents the locker from falling back to its very short internal default timeout when multiple goroutines contend for the shared Unix socket.

- `internal/daemons/osqueryd/kernel/`: Contains `KernelTables` (27 kernel/system observability tables) and a `Queryer` struct embedding `*osqueryd.Client`, with `GetSchema()` returning `KernelTables`. Registered as `"kernel"`.
- `internal/daemons/osqueryd/aws/`: Contains `AWSTables` (20 critical AWS resource tables) and a `Queryer` struct embedding `*osqueryd.Client`, with `GetSchema()` returning `AWSTables`. Registered as `"aws"`.

## `internal/logger/`

Structured JSON logging via `slog`. Writes to an XDG-compliant log file.

---

## Architectural Principles

### Lazy-Loading over Prefetching

The cache is populated on demand rather than through a separate sync phase. This eliminates startup delay, removes the need for a separate sync binary or Makefile target, and ensures the TUI is always immediately responsive. The cost is that the first query against any uncached table hits the upstream backend — subsequent queries are instant.

### Composition over Inheritance

`CachedQueryer` implements `daemons.Queryer` and embeds both an upstream `Queryer` and a `*sqlite.SQLiteStore`. It is itself a valid `Queryer` that can be passed directly to the TUI — no changes to the TUI's interface contract were needed. The decorator pattern allows the caching layer to be transparently inserted between the application and its backends.

### Single Shared Socket

All queries against a single backend share one Thrift socket connection. The osquery-go library serializes access through an internal locker, so spawning additional connections does not improve parallelism. Accordingly, lazy-loading and source-refresh logic use a single upstream client per backend rather than a connection pool. The locker's `MaxWaitTime` is set to match the configured query timeout, not the library's default.

### Concrete Store over Abstract Interface

The cache backend is `*sqlite.SQLiteStore` directly — no interface layer. This avoids premature abstraction and keeps the code straightforward. A store abstraction should be introduced only when a second implementation (e.g., Postgres, DuckDB) is required, at which point the desired interface can be extracted from the concrete usage patterns.

### Two Query Modes

The `e` and `E` keys map to two semantically distinct operations:

- **`e` (cached)**: Optimistic — returns what is in the local cache, lazily loading missing tables from the upstream. Fastest path after the first access.
- **`E` (source)**: Authoritative — fetches fresh data from the upstream backend and updates the cache, then runs the query against the refreshed local store.
