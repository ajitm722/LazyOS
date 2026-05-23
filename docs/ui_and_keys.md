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
│ tab: focus   ^e: exec   ^t: toggle  q: quit     │
└─────────────────────────────────────────────────┘
```

### Global Key Bindings

Global keybindings dictate layout manipulation, lifecycle events, and window management. These keys are defined semantically via `charmbracelet/bubbles/key` and can be remapped utilizing the Viper configuration file.

* **Tab/Shift+Tab (`focus_next`/`focus_prev`)**: Shifts the active `focus` state between the Table List, Query Input, and Results panes. The focused pane is highlighted with a distinct border color. Tab-based focus avoids interfering with letters typed in queries or table-name filters.
* **Ctrl+N (`toggle_table`)**: Toggles the `isTableMode` boolean. Switches the results display between a vertical key-value line mode and a horizontal, scrollable column table mode.
* **Ctrl+C (`quit`)**: Terminates the application cleanly via `tea.Quit`.

### Contextual Key Bindings

Contextual keys defer based on which pane holds the active `focus`.

* **Ctrl+E (Table List Focused)**: Retrieves the selected table name, constructs a `SELECT * FROM <table> LIMIT 10;` query, populates the Query Input pane, and shifts focus.
* **Ctrl+E (Query Input Focused)**: Executes the currently populated SQL query via the `osquery` client. Handles errors gracefully and formats the response data into the Results pane.
* **Ctrl+A (Query Input Focused)**: Toggles a "select all" state on the input. When activated, the next key press — a character, space, paste (`Ctrl+V`), Backspace, or Delete — replaces the entire query. Pressing `Ctrl+A` again deactivates the state without altering the text. This shortcut is useful after autofilling a table query (via `Ctrl+E` on the sidebar), allowing an immediate overwrite of the autofilled query.

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
