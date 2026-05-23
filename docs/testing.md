# Testing Guide

## Overview

The test suite spans four packages:

* **`internal/daemons`** — The column-helpers test suite, testing `ExtractColumnNames` and `DeriveColumnsFromSchema` — the domain-level column derivation logic that backend implementations rely on for the 0-row fallback.
* **`internal/tui`** — The Bubble Tea-based UI layer. Tests are white-box (`package tui`, same directory as the code) so they can access unexported helpers like `defaultAppModel`, `focusQuery`, and `getDefaultBackend` without reflection or exported test hooks.
* **`internal/tui/views/results`**, **`internal/tui/views/sidebar`**, **`internal/tui/views/querybar`** — Isolated view-component tests (package-scoped).

The backend layer is fully mocked via `internal/daemons/mock.MockQueryer`, which implements the `daemons.Queryer` interface. The `Query` method returns `(rows []map[string]string, columns []string, error)`. Columns are resolved from the first row's map keys when rows > 0 (preserving computed expressions and SELECT aliases) and fall back to `DeriveColumnsFromSchema` when rows == 0. No real osquery socket connection is ever established during testing.

## Test Suites

### `app_test.go` — Core Lifecycle

| Test | What it validates | Mechanism |
|------|-------------------|-----------|
| `TestAppInit` | `AppModel.Init()` returns a non-nil `tea.Cmd`, proving the Layout batch-init fires. | Calls `Init()` on a bare model; checks for nil. |
| `TestQuitAction` | Ctrl+C produces `tea.QuitMsg` through the action pipeline. | Sends `tea.KeyMsg{CtrlC}` through `Update`, executes the returned cmd, type-asserts the result. |
| `TestGetDefaultBackend` | `getDefaultBackend()` prefers "osquery" over other keys, falls back to the first key, and handles an empty map. | Table-driven: three `clients` map inputs mapped to expected `want` strings. |

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
| `TestFocusNavigation` | Tab cycles forward and Shift+Tab cycles backward through the three-pane focus order. | Table-driven: 6 subtests covering all transitions. Each sets `startPane`, sends the key, asserts `Current()` and `Input.Focused()`. |
| `TestUnhandledMsgFallthrough` | Unrecognised message types (e.g. `MouseMsg`) fall through to `routeToFocused` from any pane without producing a cmd. | Sends `tea.MouseMsg{}` from each pane; asserts nil cmd and unchanged pane. |
| `TestToggleTable` | Ctrl+N flips `IsTableMode` from false → true → false. | Two sequential `Update` calls with the toggle key; asserts the boolean after each. |

**Focus cycle** (Tab):
```
PaneSidebar → PaneQuery → PaneResults → PaneSidebar
```

**Reverse cycle** (Shift+Tab):
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
| `TestShortHelp` | `ShortHelp()` returns all five bindings with non-empty help keys. | Asserts non-empty result; checks each binding's `Help().Key`. |
| `TestFullHelp` | `FullHelp()` returns at least one row. | Asserts non-empty slice. |

**Default bindings**:
| Binding | Default key | Action |
|---------|------------|--------|
| `toggle_table` | Ctrl+N | `ToggleTableAction` |
| `focus_next` | Tab | `FocusNextAction` |
| `focus_prev` | Shift+Tab | `FocusPrevAction` |
| `execute/autofill` | Ctrl+E | `EnterAction` |
| `quit` | Ctrl+C | `QuitAction` |

---

### `layout_test.go` — Layout Math & Rendering

| Test | What it validates | Mechanism |
|------|-------------------|-----------|
| `TestComputePaneBounds` | Fractional pane dimension math for 6 terminal sizes. | Table-driven: 30% sidebar, 70% main view, 80/20 vertical split, minus `paneContentInset` and `helpBarHeight`. |
| `TestLayoutTooSmall` | Terminal below 80×24 triggers `tooSmall` flag and warning rendering. | Sends `WindowSizeMsg{50,10}`; asserts `tooSmall` and "Terminal too small" in `View()`. |
| `TestLayoutViewRenders` | View produces non-empty output for all three focus states and help bar is present. | Calls `Layout.View()` with each `PaneID` value; checks non-empty and help-text substrings. |
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
| `TestCustomDelegateRenderSmallWidth` | Delegate renders non-empty output even at 1-unit width (textwidth <= 0). | List with width=1; delegate render produces output. |
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
| `TestComputeTableLayout` | Produces correct key count and positive column width. | Direct call with 3 columns; assertions on `keys` length and `colWidth`. |
| `TestComputeTableLayoutManyCols` | Truncates columns to 1 when width forces `maxCols = 1`. | Width=20 (20/17=1); asserts 1 key. |
| `TestExtractKeysEmpty` | Returns nil for nil input rows. | Direct call; nil assertion. |
| `TestBuildRowsTruncationSmallWidth` | Cell values truncated without ellipsis when column width <= 3. | Width=2; cell length <= colWidth; no ellipsis. |
| `TestBuildRowsTruncationLargeWidth` | Cell values truncated with ellipsis when column width > 3. | Width=5; cell length <= colWidth; ellipsis applied. |

---

### `input_test.go` — Query Bar Input (`internal/tui/views/querybar`)

| Test | What it validates | Mechanism |
|------|-------------------|-----------|
| `TestNewQuerybar` | Correct placeholder text and initial focused state. | Asserts `Placeholder == "SELECT * FROM processes LIMIT 10;"` and `Focused() == true`. |
| `TestInitQuerybar` | `Init()` returns non-nil `tea.Cmd` for cursor blink. | Direct call; non-nil assertion. |
| `TestUpdateQuerybar` | Handles `WindowSizeMsg` (dimension propagation) and key rune input. | Sends `WindowSizeMsg{50, 10}`; asserts `Width() > 0`. Sends key runes; asserts `Value() == "sel"`. |
| `TestViewQuerybar` | Returns non-empty view string. | Sets value; calls `View()`; asserts non-empty. |
| `TestRunQueryMsg` | `RunQueryMsg` stores and exposes the SQL field. | Direct field access assertion. |
| `TestSelectAll` | Ctrl+A selects all text and toggles the `selected` flag. | Sends `KeyCtrlA` on model with content; asserts `selected == true` and nil cmd. |
| `TestSelectAllThenBackspace` | Select‑all followed by Backspace clears the input. | Ctrl+A → Backspace chain; asserts empty value and nil cmd. |
| `TestSelectAllThenDelete` | Select‑all followed by Delete clears the input. | Ctrl+A → Delete chain; asserts empty value. |
| `TestSelectAllThenTypeCharacter` | Select‑all followed by a character replaces the content with that character. | Ctrl+A → `KeyRunes{'X'}`; asserts `Value() == "X"`. |
| `TestSelectAllThenSpace` | Select‑all followed by Space clears the content and inserts a space. | Ctrl+A → `KeySpace`; asserts the original text is no longer present. |
| `TestSelectAllThenPaste` | Select‑all followed by Ctrl+V clears content before the paste cmd runs. | Ctrl+A → `KeyCtrlV`; asserts the original text is gone (clipboard may be empty in CI). |
| `TestSelectAllThenNonMatchingKey` | Select‑all followed by an arrow key deselects without clearing the value. | Ctrl+A → `KeyLeft`; asserts `selected == false` and the original value is preserved. |
| `TestSelectAllToggleOff` | Pressing Ctrl+A when already selected toggles selection off without clearing the value. | Ctrl+A → Ctrl+A; asserts `selected == false`, value preserved, and non-nil cmd returned. |

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

- **`internal/daemons`**: 100.0% statement coverage.
- **`internal/logger`**: 100.0% statement coverage.
- **`internal/tui`**: 100.0% statement coverage.
- **`internal/tui/views/querybar`**: 100.0% statement coverage.
- **`internal/tui/views/results`**: 100.0% statement coverage.
- **`internal/tui/views/sidebar`**: 100.0% statement coverage.

**Omitted (no unit tests):**
- `cmd/lazyos` — entry point, no logic to test.
- `internal/config` — types-only package, no logic.
- `internal/daemons/mock` — test helpers consumed by other tests.
- `internal/daemons/osquery` — requires live osquery socket; integration test candidate (TODO).
