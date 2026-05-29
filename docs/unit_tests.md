# Unit Tests

> For integration tests against a live osquery daemon or SQLite store, see [`osquery_integration_test.md`](./osquery_integration_test.md).

## Overview

The unit test suite spans the following packages:

* **`internal/cache`** — The `CachedQueryer` decorator and lazy-loading logic. Tests cover the `extractTableNames` SQL parser, nil-store fallthrough paths, store-backed query execution with upstream error propagation, and the source-refresh (`QuerySource`) path.
* **`internal/daemons`** — The column-helpers test suite, testing `ExtractColumnNames` and `DeriveColumnsFromSchema` — the domain-level column derivation logic that backend implementations rely on for the 0-row fallback.
* **`internal/logger`** — Fast, isolated integration tests using `t.TempDir()` and `t.Setenv()`.
* **`internal/tui`** — The Bubble Tea-based UI layer. Tests are white-box (`package tui`, same directory as the code) so they can access unexported helpers like `defaultAppModel`, `focusQuery`, `updateApp`, and `getDefaultBackend`.
* **`internal/tui/views/results`**, **`internal/tui/views/sidebar`**, **`internal/tui/views/querybar`** — Isolated view-component tests (package-scoped).

The backend layer is fully mocked via `internal/daemons/mock.MockQueryer`, which implements the `daemons.Queryer` interface. The `Query` method returns `(rows []map[string]string, columns []string, error)`. Columns are resolved from the first row's map keys when rows > 0 (preserving computed expressions and SELECT aliases) and fall back to `DeriveColumnsFromSchema` when rows == 0. No real osquery socket connection is ever established during testing.

## Test Suites

### `app_test.go` — Core Lifecycle + Source Queries

| Test | What it validates | Mechanism |
|------|-------------------|-----------|
| `TestAppInit` | `AppModel.Init()` returns a non-nil `tea.Cmd`, proving the Layout batch-init fires. | Calls `Init()` on a bare model; checks for nil. |
| `TestQuitAction` | Ctrl+C produces `tea.QuitMsg` through the action pipeline. | Sends `tea.KeyMsg{CtrlC}` through `Update`, executes the returned cmd, type-asserts the result. |
| `TestGetDefaultBackend` | `getDefaultBackend()` prefers `"kernel"` over other keys, falls back to the first key, handles missing entries and empty maps. | Table-driven: multiple `clients` map inputs mapped to expected `want` strings. |
| `TestExecuteSourceAction` | `E` key in query bar produces `RunSourceQueryMsg`; `E` key elsewhere produces no cmd. | Sends `KeyRunes{'E'}` from query pane and sidebar; asserts msg type or nil cmd. |
| `TestNextBackendAction` | `B` key cycles through two backends, wraps around, and single-backend mode returns early. | Sends `KeyRunes{'B'}` on two-backend model, checks `activeBackend`; same on single-backend. |
| `TestRunSourceQueryMsgHandler` | `RunSourceQueryMsg` through `Update` produces a `QueryResultMsg` from the mock. | Sends `RunSourceQueryMsg` directly; executes returned cmd, type-asserts result. |
| `TestRunSourceQueryMsgError` | `RunSourceQueryMsg` with a failing mock produces `QueryErrorMsg`. | Mock with `DefaultErr`; asserts error message type. |
| `TestExecuteSourceActionEmptyQuery` | `E` with empty query produces no cmd. | Sends `E` on empty input; nil assertion. |
| `TestExecuteActionEmptyQuery` | `e` with empty query produces no cmd. | Sends `e` on empty input; nil assertion. |
| `TestNextBackendSingleBackend` | Single-backend `B` returns immediately without cycling. | Single-backend model; asserts activeBackend unchanged. |
| `TestAutofillActionWrongPane` | `a` from non-sidebar pane does nothing. | Sends `a` from query pane; nil cmd assertion. |
| `TestHandleKeyMsgInsertModeEsc` | `Esc` in INSERT mode returns to NORMAL mode. | Sets `m.mode = InsertMode`, sends `KeyEsc`; asserts `NormalMode`. |
| `TestHandleKeyMsgInsertModeCtrlL` | `Ctrl+L` in INSERT mode cycles the pane. | Sets INSERT mode, sends `KeyCtrlL`; asserts pane changed. |
| `TestHandleKeyMsgInsertModeRoute` | Non-control keys in INSERT mode preserve the mode. | Sends rune in INSERT mode; asserts mode unchanged. |
| `TestHandleKeyMsgNormalModeI` | `i` in query pane enters INSERT mode. | Sends `KeyRunes{'i'}` from query pane; asserts `InsertMode`. |
| `TestGetDefaultBackendNoKernel` | Fallback when kernel is absent. | Clients map without kernel; asserts first backend chosen. |
| `TestGetDefaultBackendMissingEntry` | Missing client entries skipped during scan. | Clients map has `"real"` but order has `"missing"` first. |

**Helpers defined here** (available to all test files in the package):
- `newAppModel(mock)` — model with 100×50 terminal and mock backend.
- `newAppModelSized(mock, w, h)` — model with custom terminal size and mock backend.
- `defaultAppModel()` — shorthand for tests needing only a plain model.
- `focusQuery(m, sql)` — sets pane to `PaneQuery`, fills input, focuses it.
- `updateApp(m, msg)` — calls `m.Update(msg)` and unwraps the `tea.Model` type assertion.

---

### `app_focus_test.go` — Navigation & Panes

| Test | What it validates | Mechanism |
|------|-------------------|-----------|
| `TestFocusNavigation` | Ctrl+L cycles forward and Ctrl+H cycles backward through the three-pane focus order. | Table-driven: 6 subtests covering all transitions. Each sets `startPane`, sends the key, asserts `Current()` and `Input.Focused()`. |
| `TestUnhandledMsgFallthrough` | Unrecognised message types (e.g. `MouseMsg`) fall through to `routeToFocused` from any pane without producing a cmd. | Sends `tea.MouseMsg{}` from each pane; asserts nil cmd and unchanged pane. |
| `TestToggleTable` | Ctrl+N flips `IsTableMode` from false → true → false. | Two sequential `Update` calls with the toggle key; asserts the boolean after each. |

**Focus cycle** (Ctrl+L):
```
PaneSidebar → PaneQuery → PaneResults → PaneSidebar
```

**Reverse cycle** (Ctrl+H):
```
PaneSidebar → PaneResults → PaneQuery → PaneSidebar
```

---

### `app_query_test.go` — Query Dispatch & Daemon Interaction

| Test | What it validates | Mechanism |
|------|-------------------|-----------|
| `TestQueryDispatchStandard` | Full query pipeline: Enter → `RunQueryMsg` → backend call → `QueryResultMsg` → results displayed in viewport. | Each step of the Bubble Tea message loop is manually unwound: cmd is executed, its return is fed back through `Update`. Asserts message types, row count, pane transition to `PaneResults`, and viewport content. |
| `TestQueryTimeout` | Backend-level timeout enforcement via mock's internal timeout (no real waiting). | Runs inside `synctest.Test`. Mock has `InternalTimeout=3s`, `SlowDuration=5s`. Mock wraps ctx with `context.WithTimeout`, then selects on its Done() vs `time.After(5s)`. Synctest advances both timers; the 3s deadline fires first. |
| `TestQueryErrors` | Error display for recognised sentinels and generic errors. | Table-driven with two cases: `daemons.ErrQueryFailed` (displays "query failed") and a plain error (displays the error text verbatim). |
| `TestEmptyQueryEnter` | Enter on an empty query input produces no cmd. | Pane set to query, no value set; asserts `cmd == nil`. |
| `TestAutofillTrigger` | Enter on a sidebar item produces `AutofillMsg` with the table name. | Mock provides a schema entry so the sidebar has a selectable item; Enter triggers the sidebar branch of `EnterAction`. |
| `TestAutofillHandler` | `AutofillMsg` fed directly through `Update` populates the query bar, shifts focus to `PaneQuery`, and focuses the input. | Direct `AutofillMsg` send; asserts query value, pane, and focus state. |
| `TestQueryZeroRows` | Zero rows with schema produces column headers in table mode. | Mock has `Schema=mock.MockTables` and empty `DefaultResult`; schema-derived columns (`["pid", "name", "state"]`) appear in the table header. |
| `TestQueryZeroRowsDirect` | `QueryResultMsg` fed directly with explicit columns. | Bypasses mock; feeds `Columns: []string{"pid", "name", "state"}` directly; table header rendered. |
| `TestQueryZeroRowsNoColumns` | Zero rows with nil columns still produces a valid (headerless) table. | `QueryResultMsg` with nil columns; table renders without panicking. |
| `TestQueryZeroRowsFallbackNilColumns` | Full mock pipeline returns nil columns (no schema, no rows); results view still renders gracefully without panicking. | Mock without `Schema` and empty `DefaultResult`; line mode shows "0 rows returned."; table mode renders headers. |
| `TestQueryZeroRowsFromSchema` | Zero rows, no row-level columns, but schema provides column derivation. | Mock has `Schema` with `processes` table; columns derived from schema via `DeriveColumnsFromSchema`; table headers appear in schema order (`pid` before `name`). |
| `TestQueryHeaderOrder` | Table column headers appear in schema-defined order, not map-iteration order. | Row data has keys in reverse order (`state`, `name`, `pid`); schema order (`pid`, `name`, `path`, `cmdline`, `state`) is verified in both `columns` slice and rendered table. |

**Pipeline flow** (standard dispatch):
```
KeyMsg(Enter) → EnterAction.Apply → cmd → RunQueryMsg
                                                     ↓
                                              handleRunQueryMsg → cmd → mock.Query
                                                                           ↓
                                              QueryResultMsg ← cmd ←  rows returned
                                                     ↓
                                              handleQueryResultMsg → PaneResults, viewport populated
```

**Pipeline flow** (timeout):
```
mock.InternalTimeout=3s, mock.SlowDuration=5s
                                ↓
                mock.Query wraps ctx with context.WithTimeout(ctx, 3s)
                                ↓
                mock.Query enters select{timeoutCtx.Done(), time.After(5s)}
                                ↓ (synctest advances clock; 3s deadline fires first)
                     timeoutCtx.Done() → returns ErrQueryTimeout
                                ↓
                          QueryErrorMsg{Err: ErrQueryTimeout}
                                ↓
                          handleQueryErrorMsg → "query timed out" in viewport
```

---

### `app_keys_test.go` — Key Bindings & Configuration

| Test | What it validates | Mechanism |
|------|-------------------|-----------|
| `TestKeyBindingOverride` | Custom key overrides rewire the ActionRegistry correctly. | Config remaps ToggleTable to Ctrl+T and FocusNext to Ctrl+N. Table-driven: each key matched via `key.Matches` and action type compared with `reflect.TypeOf`. Separate subtest asserts Ctrl+N does not match ToggleTable. |
| `TestUnmatchedKey` | A plain 'a' rune (no binding) falls through to `routeToFocused` with no cmd. | Sends `KeyRunes{'a'}`; asserts nil cmd and unchanged pane. |
| `TestShortHelp` | `ShortHelp()` returns all registered bindings with non-empty help keys. | Asserts non-empty result; checks each binding's `Help().Key`. |
| `TestFullHelp` | `FullHelp()` returns at least one row. | Asserts non-empty slice. |

**Default bindings**:
| Binding | Default key | Action |
|---------|------------|--------|
| `focus_next` | Ctrl+L | `FocusNextAction` |
| `focus_prev` | Ctrl+H | `FocusPrevAction` |
| `toggle_table` | `t` | `ToggleTableAction` |
| `autofill` | `a` | `AutofillAction` |
| `execute` | `e` | `ExecuteAction` |
| `execute_source` | `E` | `ExecuteSourceAction` |
| `quit` | `q` | `QuitAction` |

---

### `layout_test.go` — Layout Math & Rendering

| Test | What it validates | Mechanism |
|------|-------------------|-----------|
| `TestComputePaneBounds` | Fractional pane dimension math for 6 terminal sizes. | Table-driven: 30% sidebar, 70% main view, 80/20 vertical split, minus `paneContentInset` and `helpBarHeight`. |
| `TestLayoutTooSmall` | Terminal below 80×24 triggers `tooSmall` flag and warning rendering. | Sends `WindowSizeMsg{50,10}`; asserts `tooSmall` and "Terminal too small" in `View()`. |
| `TestLayoutViewRenders` | View produces non-empty output for all three focus states and help bar is present. | Calls `Layout.View()` with each `PaneID` value; checks non-empty and help-text substrings. |
| `TestLayoutViewInsertMode` | View renders the INSERT mode indicator when mode is InsertMode. | Sets `m.mode = InsertMode`; asserts "INSERT" in view string. |
| `TestRenderTooSmallWarning` | Pure `renderTooSmallWarning` function produces the warning string. | Direct function call; substring check. |
| `TestComputePaneSizes` | Outer dimension calculation yields positive widths at 100×50. | Direct function call; sanity checks on `listWidth`, `viewWidth`, `inputWidth`. |

**Pane dimension formulas**:

| Variable | Formula |
|----------|---------|
| `mainHeight` | `max(0, height − helpBarHeight)` |
| `leftWidth` | `max(0, int(width × 0.3) − paneContentInset)` |
| `leftHeight` | `max(0, mainHeight − paneContentInset)` |
| `viewWidth` | `max(0, int(width × 0.7) − paneContentInset)` |
| `viewHeight` | `max(0, int(mainHeight × 0.8) − paneContentInset)` |
| `queryHeight` | `max(0, int(mainHeight × 0.2) − paneContentInset)` |

**Constants**: `helpBarHeight = 1`, `paneContentInset = 4`, `sidebarWidthFraction = 0.3`, `resultsWidthFraction = 0.7`, `resultsHeightFraction = 0.8`, `minRequiredWidth = 80`, `minRequiredHeight = 24`.

---

### `cache_test.go` — Cached Query Decorator (`internal/cache`)

| Test | What it validates | Mechanism |
|------|-------------------|-----------|
| `TestExtractTableNames` | SQL parser correctly extracts table names from FROM, JOIN, and comma-separated clauses while skipping keywords and subqueries. | Table-driven: ~15 SQL inputs mapped to expected table name slices. |
| `TestCachedQueryer_Query` | CachedQueryer with nil store falls through to upstream. | Mock upstream; asserts result matches mock data. |
| `TestCachedQueryer_Query_NilStore` | Query with nil store returns upstream data unchanged. | Nil store; asserts row content from mock. |
| `TestCachedQueryer_QuerySource_NilStore` | QuerySource with nil store returns upstream data unchanged. | Nil store; asserts row content from mock. |
| `TestCachedQueryer_QuerySource_Error` | QuerySource propagates upstream error. | Mock with `DefaultErr`; asserts error returned. |
| `TestCachedQueryer_WithSchema` | GetSchema returns upstream schema. | Mock schema; asserts returned schema matches. |
| `TestCachedQueryer_Close` | Close delegates to upstream. | Mock upstream; asserts no error. |
| `TestCachedQueryer_Query_WithStore` | Query lazy-loads an uncached table from upstream into the store. | Real SQLite store + mock; asserts table is cached after query. |
| `TestCachedQueryer_Query_AlreadyCached` | Second query against same table reads from store, not upstream. | Real store seeded with stale data; asserts stale data returned (proves upstream not called). |
| `TestCachedQueryer_QuerySource_WithStore` | QuerySource refreshes from upstream and overwrites the store. | Real store seeded with stale data; asserts fresh data returned. |
| `TestCachedQueryer_Query_UpstreamError` | Query propagates fetch error when lazy-loading fails. | Real store + failing mock; asserts error from Query. |
| `TestCachedQueryer_QuerySource_UpstreamError` | QuerySource propagates fetch error. | Real store + failing mock; asserts error from QuerySource. |

---

### `columns_test.go` — Column Helper Functions (`internal/daemons`)

| Test | What it validates | Mechanism |
|------|-------------------|-----------|
| `TestExtractColumnNames` | Parses comma-separated column definitions with optional type annotations into a slice of column-name strings. | Table-driven: standard, type-less, single-column, and empty inputs. |
| `TestAutofillColumns` | Returns a comma-separated column string for a matching table, `"*"` when not found, matching case-insensitively, and `"*"` for an empty schema. | Table-driven: table found, not found, case-insensitive, empty schema. |
| `TestDeriveColumnsFromSchema` | Looks up the `FROM` table in the schema catalog and returns ordered column names. | Table-driven: simple SELECT, WHERE clause, lowercase table, no FROM clause, unknown table, empty schema. |

---

### `sidebar_test.go` — Sidebar View (`internal/tui/views/sidebar`)

| Test | What it validates | Mechanism |
|------|-------------------|-----------|
| `TestNewSidebar` | `New()` creates a sidebar with the correct title and populates items from the backend schema. | Asserts `List.Title == "Tables"`; checks `Title()` and `FilterValue()` on the first item. |
| `TestInitSidebar` | `Init()` returns nil (no initial command). | Direct call; nil assertion. |
| `TestUpdateSidebar` | Handles `WindowSizeMsg` and renders a non-empty view without panicking. | Sends `WindowSizeMsg{30,20}`; asserts width and non-empty view. |
| `TestCustomDelegateRender` | Custom delegate produces non-empty output for `TableItem`; `wrappedItem` surfaces description correctly. | Renders via `customDelegate.Render`; checks output and `wrappedItem.Description()`. |
| `TestCustomDelegateRenderSmallWidth` | Delegate renders non-empty output even at 1-unit width. | List with width=1; delegate render produces output. |
| `TestCustomDelegateRenderNonTableItem` | Fallback rendering path for non-`TableItem` list items. | `dummyItem` list item; delegate render produces output. |
| `TestAutofillMsg` | `AutofillMsg` struct stores and exposes `TableName`. | Direct field access assertion. |

---

### `results_test.go` — Results View (`internal/tui/views/results`)

| Test | What it validates | Mechanism |
|------|-------------------|-----------|
| `TestNewResults` | Default initial state: `IsTableMode` false, viewport shows "Awaiting Query". | Direct field checks and viewport substring match. |
| `TestInitResults` | `Init()` returns nil. | Direct call; nil assertion. |
| `TestUpdateResults` | Handles `WindowSizeMsg` dimension update and delegates key events in both line and table modes. | Sends `WindowSizeMsg`; asserts `Width`/`Height`; sends `KeyDown` in line and table modes. |
| `TestViewStr` | `ViewStr()` returns non-empty output in both line and table modes. | Flips `IsTableMode`; asserts non-empty each time. |
| `TestFormatData` | Formats row data with "Row N" labels, "0 rows returned" for empty results, and extracts keys when columns nil. | `FormatData` with data, nil data, and data without columns; substring checks. |
| `TestFormatError` | Produces viewport content with "Query failed:" prefix. | `FormatError(errors.New("db error"))`; substring match. |
| `TestFormatMessage` | Sets viewport content and clears the table widget. | `FormatMessage("loading...")`; substring match. |
| `TestComputeTableLayout` | Produces correct key count and positive column width. | Direct call with 3 columns; assertions on `keys` length and `colWidth`. |
| `TestComputeTableLayoutManyCols` | Truncates columns to 1 when width forces `maxCols = 1`. | Width=20 (20/17=1); asserts 1 key. |
| `TestExtractKeysEmpty` | Returns nil for nil input rows. | Direct call; nil assertion. |
| `TestBuildRowsTruncationSmallWidth` | Cell values truncated without ellipsis when column width <= 3. | Width=2; cell length <= colWidth; no ellipsis. |
| `TestBuildRowsTruncationLargeWidth` | Cell values truncated with ellipsis when column width > 3. | Width=5; cell length <= colWidth; ellipsis applied. |
| `TestUpdateTableMode` | Key events in table mode delegate to the table widget. | Sets `IsTableMode=true`, sends `KeyDown`; asserts mode unchanged. |
| `TestUpdateResultsLineModeJKey` | Pressing `j` in line mode scrolls the viewport down. | Sets content, records YOffset, sends `j`; asserts offset increased. |
| `TestUpdateResultsLineModeKKey` | Pressing `k` in line mode scrolls the viewport up. | Scrolls down first, sends `k`; asserts offset decreased. |

---

### `input_test.go` — Query Bar Input (`internal/tui/views/querybar`)

| Test | What it validates | Mechanism |
|------|-------------------|-----------|
| `TestNewQuerybar` | Correct placeholder text and initial focused state. | Asserts `Placeholder == "SELECT * FROM processes LIMIT 10;"` and `Focused() == true`. |
| `TestInitQuerybar` | `Init()` returns non-nil `tea.Cmd` for cursor blink. | Direct call; non-nil assertion. |
| `TestUpdateQuerybar` | Handles `WindowSizeMsg` (dimension propagation) and key rune input. | Sends `WindowSizeMsg{50, 10}`; asserts `Width() > 0`. Sends key runes; asserts `Value() == "sel"`. |
| `TestViewQuerybar` | Returns non-empty view string. | Sets value; calls `View()`; asserts non-empty. |
| `TestRunQueryMsg` | `RunQueryMsg` stores and exposes the SQL field. | Direct field access assertion. |
| `TestSelectAll` | Ctrl+A toggles the `selected` flag and applies a visual selection style. | Sends `KeyCtrlA` on a focused model with content; asserts `selected == true` and nil cmd. |
| `TestSelectAllThenBackspace` | Select-all followed by Backspace clears the input. | Ctrl+A → Backspace chain; asserts empty value and nil cmd. |
| `TestSelectAllThenDelete` | Select-all followed by Delete clears the input. | Ctrl+A → Delete chain; asserts empty value. |
| `TestSelectAllThenTypeCharacter` | Select-all followed by a character replaces the content with that character. | Ctrl+A → `KeyRunes{'X'}`; asserts `Value() == "X"`. |
| `TestSelectAllThenSpace` | Select-all followed by Space clears the content and inserts a space. | Ctrl+A → `KeySpace`; asserts the original text is no longer present. |
| `TestSelectAllThenPaste` | Select-all followed by Ctrl+V clears content before the paste cmd runs. | Ctrl+A → `KeyCtrlV`; asserts the original text is gone. |
| `TestSelectAllThenNonMatchingKey` | Select-all followed by an arrow key deselects without clearing the value. | Ctrl+A → `KeyLeft`; asserts `selected == false` and the original value is preserved. |
| `TestSelectAllToggleOff` | Pressing Ctrl+A when already selected toggles selection off without clearing the value. | Ctrl+A → Ctrl+A; asserts `selected == false`, value preserved, and non-nil cmd returned. |
| `TestBlur` | Blur deactivates the querybar, clears focus and selection state. | `Focus()` → `Blur()`; asserts `active`, `selected`, and `Focused()` are false. |

---

## Mock Architecture

```
internal/daemons/mock/
  client.go   -- MockQueryer (implements daemons.Queryer)
  schema.go    -- MockTables (canonical mock schema catalog)
```

`MockQueryer` implements `Query`, `Close`, and `GetSchema`. Fields:

| Field | Type | Purpose |
|-------|------|---------|
| `Results` | `map[string][]map[string]string` | Per-SQL injected responses. |
| `DefaultResult` | `[]map[string]string` | Fallback when no SQL match. |
| `DefaultErr` | `error` | If set, returned on every call (overrides results). |
| `SimulateSlowQuery` | `bool` | When true, `Query` enters `select { case <-ctx.Done(): ... ; case <-time.After(SlowDuration): ... }`. |
| `SlowDuration` | `time.Duration` | Duration for the `time.After` branch in slow-query simulation. |
| `InternalTimeout` | `time.Duration` | If set, wraps `ctx` with `context.WithTimeout` before selecting, mirroring the real osquery client's behavior. |
| `Schema` | `[]daemons.TableSchema` | Used by `GetSchema()` and internally by `Query` for column derivation and row filtering. |

**Column resolution**: `MockQueryer.Query` resolves columns from the first row's map keys when rows > 0, preserving computed expressions and SELECT aliases exactly as returned. When rows == 0, it falls back to `daemons.DeriveColumnsFromSchema(sql, m.Schema)` so the TUI can render column headers with no data.

**Schema catalog** (`schema.go`): `MockTables` defines the canonical schemas used in tests (tables `processes`, `users`, `empty`). Tests that need to verify column ordering or zero-row behavior set `Schema: mock.MockTables`.

The mock lives under `internal/daemons/mock/` and is only imported by `_test.go` files, so it is never compiled into the final binary. Verified via `go list -deps ./cmd/lazyos`.

---

## Synctest Timeout Testing

The `testing/synctest` package (available in Go 1.26+) provides an isolated time bubble. Within `synctest.Test`, all `time.Sleep` and `time.After` calls use a fake clock that advances only when every goroutine in the bubble is durably blocked. This enables deterministic timeout tests without wall-clock waiting.

**How `TestQueryTimeout` works**:
1. The mock is configured with `SimulateSlowQuery=true`, `SlowDuration=5s`, and `InternalTimeout=3s`.
2. The query cmd is executed — `handleRunQueryMsg` passes a plain context to `mock.Query(ctx, sql)` (the TUI is agnostic to timeouts).
3. Inside the mock, `Query` wraps the context with `context.WithTimeout(ctx, 3s)` via `InternalTimeout`, then enters a `select { case <-timeoutCtx.Done(): ... ; case <-time.After(5s): ... }`.
4. Synctest's fake clock advances both timers; the 3s internal deadline fires first, `timeoutCtx.Done()` closes, and the mock returns `ErrQueryTimeout`.
5. If the `time.After(5s)` branch were to win instead, the mock would return a different error, causing the test to fail — proving the backend-level timeout is enforced.

The entire test completes in near-zero real time.

---

## Coverage

- **`internal/cache`**: 98.9% statement coverage.
- **`internal/daemons`**: 100.0% statement coverage.
- **`internal/logger`**: 100.0% statement coverage.
- **`internal/tui`**: 97.2% statement coverage.
- **`internal/tui/views/querybar`**: 100.0% statement coverage.
- **`internal/tui/views/results`**: 100.0% statement coverage.
- **`internal/tui/views/sidebar`**: 100.0% statement coverage.

**Omitted from unit-test coverage (no `_test.go` files):**
- `cmd/lazyos` — entry point, no logic to test.
- `internal/config` — types-only, no logic.
- `internal/daemons/mock` — test helpers consumed by other tests.
- `internal/daemons/osqueryd` — integration-only (requires live osquery socket).
- `internal/daemons/osqueryd/aws` — integration-only (requires live osquery socket + cloudquery).
- `internal/daemons/osqueryd/kernel` — integration-only (requires live osquery socket).
- `internal/store/sqlite` — integration-only (requires `-tags=integration`; see [`osquery_integration_test.md`](./osquery_integration_test.md)).

Overall unit-test statement coverage: **98.6%**.
