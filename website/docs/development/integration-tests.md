---
sidebar_position: 2
---

# Integration Tests

For unit tests (mocked backend, no external dependencies), see [Unit Tests](./unit-tests).

## Overview

Integration tests in `internal/daemons/osquery/osquery_integration_test.go` validate the real Thrift RPC client against a live `osqueryd` daemon. Unlike the unit tests (which use `internal/daemons/mock.MockQueryer`), these tests require an actual osquery binary and a Unix domain socket.

The file uses the build tag `//go:build integration` so it is never compiled during `make test` or `go test ./...`. It is only compiled and executed when the `-tags=integration` flag is passed.

## Execution

### Prerequisite

A minimal osquery binary is expected at `./build/osquery/osqueryd`. The `make setup-sandbox` target downloads it automatically if absent.

### Running

| Command | Output format |
|---------|---------------|
| `make test-integration` | `gotestsum --format pkgname` — one-line summary per package. |
| `make test-integration-verbose` | `gotestsum --format standard-verbose` — full `=== RUN` output. |

Both targets:
1. Depend on `setup-sandbox` (downloads osqueryd if missing).
2. Invoke `WITH_OSQUERY_DAEMON`, a Makefile macro that starts an ephemeral `osqueryd`, waits for the socket, runs the test binary, then cleans up.

### Environment

The osquery daemon is launched with:

```
osqueryd --ephemeral --disable_database --disable_events \
  --logger_path=/tmp/lazyos_sandbox/logs \
  --extensions_socket=/tmp/lazyos_sandbox/osquery.em
```

The macro sets `LAZYOS_TEST_SOCKET=/tmp/lazyos_sandbox/osquery.em` in the test environment. Tests read this variable via `os.Getenv("LAZYOS_TEST_SOCKET")`.

## Test Catalog

### Client Connectivity

| Test | Purpose |
|------|---------|
| `TestIntegrationNewClient_ValidSocket` | Verifies `NewClient` connects to the live socket and `Close()` returns without error. |
| `TestIntegrationNewClient_InvalidSocket` | Verifies `NewClient` fails immediately (zero startup timeout) against a non-existent path. |

### Schema Verification

| Test | Purpose |
|------|---------|
| `TestIntegrationGetSchema` | Asserts `GetSchema()` returns every entry in `CoreTables` with matching `Name` and `Columns`. |
| `TestIntegrationTableHasDescription` | Asserts no `CoreTables` entry has an empty `Description`. |
| `TestIntegrationTableHasName` | Asserts no `CoreTables` entry has an empty `Name`. |
| `TestIntegrationDeriveColumnsConsistency` | For each table, verifies `DeriveColumnsFromSchema` round-trips correctly. |

### Query Execution

| Test | Purpose |
|------|---------|
| `TestIntegrationQuery_Basic` | Executes `SELECT pid, name FROM processes LIMIT 1` and asserts non-empty rows and expected columns. |
| `TestIntegrationQuery_Timeout` | Passes a zero-deadline context and asserts a timeout error is returned. |
| `TestIntegrationQuery_InvalidSQL` | Sends malformed SQL (`SELECT invalid`) and asserts an error is returned. |
| `TestIntegrationQuery_EmptyResult` | Sends `WHERE pid = -1` (zero rows) and verifies columns are still populated (schema fallback path). |

### Full Schema Cross-Reference

**`TestIntegrationAllTableSchemas`**

For every table in `CoreTables`:
1. **PRAGMA validation**: Runs `PRAGMA table_info(<name>)` via Thrift and builds a set of actual column names. Asserts every declared column exists in that set.
2. **SELECT validation**: Runs `SELECT * FROM <name> LIMIT 1`. Asserts every declared column appears in the result's column list. If rows are returned, asserts they contain the declared column keys.

**`TestIntegrationCoreTablesAreQueryable`**

Runs `SELECT COUNT(*) AS cnt FROM <name>` on every table and logs the row count. Tables that return zero rows (e.g., `docker_containers` on a host without Docker, `sudoers` without sudo rules) are still considered queryable — the test only checks that the query succeeds and returns at least one row (the COUNT result).

## Adding New Integration Tests

1. Add new test functions in `osquery_integration_test.go` prefixed with `TestIntegration`.
2. The test file already opens and closes a single `Client` per test function — use `socketPath(t)` to obtain the socket path.
3. Available helpers:
   - `socketPath(t)` — returns `LAZYOS_TEST_SOCKET` or fatally fails.
   - `actualColumnsForTable(t, c, tableName)` — returns `map[string]struct{}` of real column names via `PRAGMA table_info`.
   - Constants `startupTimeout` (5s) and `queryTimeout` (10s) for `NewClient`.
4. Run `make test-integration-verbose` to verify.

## Adding New Tables to CoreTables

When adding a new table to `CoreTables` in `internal/daemons/osquery/schema.go`:

1. Verify the table exists in osquery: `./build/osquery/osqueryd --S "PRAGMA table_info(<newname>)"`.
2. Verify every column name in the declaration matches the output of `PRAGMA table_info`.
3. Run `make test-integration` — `TestIntegrationAllTableSchemas` and `TestIntegrationCoreTablesAreQueryable` will validate the new entry automatically.
