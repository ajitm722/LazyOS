---
sidebar_position: 3
---

# Caching Architecture

LazyOS embeds a SQLite database as a local persistent cache. Every table queried through the `e` (execute cached) path is lazy-loaded on first access: the backend is queried for the full table (`SELECT *`), the result is written to SQLite, and the user's original SQL executes against the local store. Subsequent queries against the same table hit the local store directly and return instantly — no backend call occurs.

The `E` (execute source) path always queries the upstream backend directly and refreshes the cached copy of every referenced table, ensuring authoritative data while keeping the cache up to date.

The `CachedQueryer` struct in `internal/cache/` implements the `daemons.Queryer` interface and acts as a decorator around the upstream backend. It holds a direct reference to `*sqlite.SQLiteStore` for both persistence and local query execution. The store has no abstract interface wrapper — it is called directly, and the architecture only adds an abstraction boundary when a second store implementation is needed.

## Query Paths

| Key | Path | Behavior |
|---|---|---|
| `e` | Cached | Optimistic — returns what is in the local cache, lazily loading missing tables from the upstream. Fastest path after the first access. |
| `E` | Source | Authoritative — fetches fresh data from the upstream backend and updates the cache, then runs the query against the refreshed local store. |

## Cache Storage

- **Location**: `~/.cache/lazyos/lazyos.db` (or `cache-db-path` override)
- **Driver**: `mattn/go-sqlite3` (CGo)
- **Journal Mode**: WAL (Write-Ahead Logging) for concurrent read/write
- **Busy Timeout**: 5 seconds
- **Value Type**: All values stored as `TEXT`
- **Sync**: `SyncTable` atomically replaces a table's full contents in a single transaction
