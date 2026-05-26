# UI and Key Bindings

## User Interface and Interaction Model

The interface consists of three interactive panes:

```text
┌─────────────────────────────────────────────────┐
│ ┌────────────┐ ┌──────────────────────────────┐ │
│ │ osquery    │ │ SELECT * FROM processes ...  │ │
│ │ tables     │ │ (query input)                │ │
│ │            │ │                              │ │
│ │ processes  │ ├──────────────────────────────┤ │
│ │ routes     │ │ --- Row 1 ---                │ │
│ │ listening… │ │ pid   = 1234                 │ │
│ │ users      │ │ name  = nginx                │ │
│ │            │ │ state = LISTEN               │ │
│ └────────────┘ └──────────────────────────────┘ │
│ -- NORMAL -- j/k: nav t: toggle a: autofill e: execute ^l: next ^h: prev ^c: quit │
└─────────────────────────────────────────────────┘
```

### Modal Editing

LazyOS uses a vim-inspired modal interaction model with two modes:

* **NORMAL mode** (default): Keys trigger navigation and commands (`j`/`k` to move, `a` to autofill, `e` to execute, etc.).
* **INSERT mode** (query bar only): Keys type into the SQL editor. Press `i` in the query bar to enter INSERT mode; press `Esc` to return to NORMAL.

A mode indicator (`-- NORMAL --` or `-- INSERT --`) is displayed in the bottom-left corner.

### Global Key Bindings

Global keybindings dictate layout manipulation, lifecycle events, and window management. These keys are defined semantically via `charmbracelet/bubbles/key` and can be remapped utilizing the Viper configuration file.

* **Ctrl+L (`focus_next`)**: Shifts the active `focus` forward through the panes: Table List → Query Input → Results → Table List.
* **Ctrl+H (`focus_prev`)**: Shifts the active `focus` backward through the panes.
* **`t` (`toggle_table`)**: Toggles the `isTableMode` boolean. Switches the results display between a vertical key-value line mode and a horizontal, scrollable column table mode.
* **`a` (`autofill`)**: In the Table List pane, retrieves the selected table name, constructs a `SELECT <columns> FROM <table> LIMIT 10;` query, populates the Query Input pane, and shifts focus (remaining in NORMAL mode).
* **`e` (`execute`)**: In the Query Input pane, executes the currently populated SQL query and shifts focus to the Results pane.
* **Ctrl+C (`quit`)**: Terminates the application cleanly via `tea.Quit`.

### Navigation in NORMAL Mode

In NORMAL mode, each pane responds to `j` and `k` for vertical movement:

* **Table List (sidebar)**: `j`/`k` scroll through the table list. Press `/` to enter filter mode, which lets you type to narrow the list; `Esc` exits filter mode.
* **Query Input**: `j`/`k` are not applicable; press `i` to enter INSERT mode for editing.
* **Results (line mode)**: `j`/`k` scroll the viewport up and down.
* **Results (table mode)**: `j`/`k` move the selection cursor up and down through rows.

### Contextual Key Bindings

Contextual keys defer based on which pane holds the active `focus`.

* **`a` (Table List Focused)**: Autofills the query bar with a `SELECT <columns> FROM <table> LIMIT 10;` query for the selected table. Focus shifts to the query bar in NORMAL mode.
* **`e` (Query Input Focused)**: Executes the currently populated SQL query. The response is formatted into the Results pane and focus shifts there.
* **`i` (Query Input Focused, NORMAL mode)**: Enters INSERT mode, allowing text to be typed into the SQL editor. The cursor appears and the mode indicator changes to `-- INSERT --`.
* **`Esc` (Query Input Focused, INSERT mode)**: Returns to NORMAL mode, hiding the cursor and restoring command-key behaviour.
* **Ctrl+A (Query Input Focused, INSERT mode)**: Toggles a "select all" state on the input. When activated, the next key press — a character, space, paste (`Ctrl+V`), Backspace, or Delete — replaces the entire query. Pressing `Ctrl+A` again deactivates the state without altering the text.

## Data Rendering Architecture

The application handles data returned from osquery by concurrently constructing two distinct views to manage arbitrary schema sizes gracefully:

1. **Line-Mode View (Default)**:
    * Iterates through each row, formatting output as `column = value`.
    * Provides robust rendering for tables with an unpredictable or excessive number of columns, preventing terminal overflow.
    * Rendered via a `viewport.Model`.

2. **Table-Mode View (Toggled)**:
    * Constructs a traditional column-and-row table using `table.Model`.
    * Implements dynamic column truncation: removes rightmost columns if the total column count exceeds the calculated maximum based on terminal width.
    * Implements value truncation: appends `...` to cell values that exceed their calculated column width.
    * Rebuilt entirely upon each successful query execution to guarantee schema alignment.

## Layout Sizing Model

All width and height values in `internal/tui/layout.go` are expressed in **characters** (width) and **lines** (height). A width of `80` means 80 monospace characters fit horizontally; a height of `24` means 24 lines of text fit vertically.

### Unit breakdown

| Term | Unit | Example |
|---|---|---|
| `tea.WindowSizeMsg.Width` | characters | `100` → 100 monospace characters fit on each line |
| `tea.WindowSizeMsg.Height` | lines | `50` → 50 text lines can be displayed |
| Fraction constant | multiplier | `sidebarWidthFraction = 0.3` → sidebar gets `0.3 × 100 = 30` characters of width |
| Integer constant | absolute amount | `helpBarHeight = 1` → 1 line reserved for the help menu |
| Inset | absolute amount | `paneContentInset = 4` → 4 characters/lines subtracted for borders and gutter |

### Worked example: 100×50 terminal

Given `tea.WindowSizeMsg{Width: 100, Height: 50}` (100 characters wide, 50 lines tall):

1. **Main height** = `50 − 1` (help bar) = 49 lines.
2. **Sidebar** (30 % width, full left height):
   * `leftWidth` = `0.3 × 100 − 4` = 26 characters wide
   * `leftHeight` = `49 − 4` = 45 lines tall
3. **Results pane** (70 % width, 80 % main height):
   * `viewWidth` = `0.7 × 100 − 4` = 66 characters wide
   * `viewHeight` = `0.8 × 49 − 4` = 35 lines tall
4. **Query bar** (70 % width, remaining 20 % main height):
   * `viewWidth` = 66 characters wide (same as results)
   * `queryHeight` = `0.2 × 49 − 4` = 5 lines tall
