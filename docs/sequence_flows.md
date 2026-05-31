# Sequence Flows

The following sequence diagrams detail the internal data flow and component interactions within the `tui` package during key user operations. Each diagram is annotated with the exact source files that make the flow work, enabling a developer to trace every message from the terminal to the caching layer and the upstream daemon.

---

## 1. Application Bootstrap

This flow traces the startup path from the shell through Viper configuration loading, backend bootstrap, SQLite store initialisation, `CachedQueryer` wrapping, TUI construction, and Bubble Tea runtime start.

### Source Files

| Step | File | Key Function / Type |
|------|------|-------------------|
| CLI entry | `cmd/lazyos/main.go:15` | `main()` |
| Command registration | `cmd/lazyos/root.go` | `Execute(ctx)` / `rootCmd` |
| Config loading | `cmd/lazyos/root.go` | `initConfig()` via `PersistentPreRunE` |
| Config schema | `internal/config/config.go` | `Config` struct |
| Logger setup | `cmd/lazyos/root.go` | `runApp` / `SetupFile` |
| Backend bootstrap | `cmd/lazyos/root.go` | `bootstrapBackends(cfg)` |
| SQLite store open | `cmd/lazyos/root.go` | `sqlite.Open(dbPath)` |
| Cache wrapping | `cmd/lazyos/root.go` | `cache.NewCachedQueryer(client, st)` per backend |
| TUI construction | `internal/tui/app.go` | `NewApp(clients, backendOrder, cfg.Keys)` |
| Input handling | `internal/tui/registry.go` | `NewInputHandler(cfg)` |
| Program start | `cmd/lazyos/root.go` | `startTUI(...)` calls `tea.NewProgram` |

### Diagram

```mermaid
sequenceDiagram
    participant Shell as Shell
    participant Main as cmd/lazyos/root.go (Execute)
    participant Config as cmd/lazyos/root.go (initConfig)
    participant Viper as spf13/viper
    participant Pipeline as cmd/lazyos/root.go (runApp)
    participant Logger as cmd/lazyos/root.go (SetupFile)
    participant Bootstrap as cmd/lazyos/root.go (bootstrapBackends)
    participant Store as internal/store/sqlite (SQLiteStore)
    participant Cache as internal/cache/queryer.go (NewCachedQueryer)
    participant TUI as internal/tui/app.go (NewApp)
    participant Runtime as Bubble Tea (tea.NewProgram)

    Shell->>Main: ./lazyos
    Note over Main: Cobra parses PersistentFlags (--config)<br/>and Flags (--osquery-socket, etc.)

    Main->>Main: PersistentPreRunE fires before RunE

    Main->>Config: initConfig()
    activate Config
    Config->>Viper: viper.AddConfigPath / viper.SetConfigFile
    Config->>Viper: viper.AutomaticEnv()
    Config->>Viper: viper.ReadInConfig()
    Config-->>Main: nil
    deactivate Config

    Main->>Viper: viper.Unmarshal(&cfg)
    Main->>Pipeline: RunE calls runApp(cfg)
    activate Pipeline

    Pipeline->>Logger: logger.SetupFile(cfg.LogFile)
    activate Logger
    Logger-->>Pipeline: log, logFile, finalLogPath
    deactivate Logger

    Pipeline->>Bootstrap: bootstrapBackends(cfg)
    activate Bootstrap
    Note over Bootstrap: Iterates available[] initializer slice
    Bootstrap-->>Pipeline: clients map[string]daemons.Queryer
    deactivate Bootstrap

    Pipeline->>Store: sqlite.Open(dbPath)
    activate Store
    Note over Store: Resolves to ~/.cache/lazyos/lazyos.db<br/>or cfg.CacheDBPath
    Store-->>Pipeline: *sqlite.SQLiteStore
    deactivate Store

    Note over Pipeline: defers store.Close()
    Pipeline->>Cache: for name, client := range clients
    Pipeline->>Pipeline: clients[name] = cache.NewCachedQueryer(client, st)
    Note over Pipeline: Each backend gets wrapped in a CachedQueryer decorator

    Pipeline->>TUI: NewApp(clients, backendOrder, keys)
    activate TUI
    Note over TUI: Initializes InputHandler with BoundActions
    TUI-->>Pipeline: Initialized AppModel
    deactivate TUI

    Pipeline->>Runtime: tea.NewProgram(AppModel, tea.WithAltScreen())
    activate Runtime
    Runtime->>TUI: Init()
    TUI-->>Runtime: tea.Batch(Child Init Cmds)
    deactivate Runtime
    deactivate Pipeline
```

---

## 2. Bubble Tea Runtime Initialization

This flow shows what happens immediately after `tea.NewProgram` is called: the synthetic `WindowSizeMsg`, dimension calculation, and the first render.

### Source Files

| Step | File | Key Function / Type |
|------|------|-------------------|
| Program start | `cmd/lazyos/root.go` | `tea.NewProgram(...)` |
| Init command | `internal/tui/app.go` | `AppModel.Init()` |
| Window resize | `internal/tui/app.go` | `handleWindowSizeMsg(msg)` |
| Layout bounds | `internal/tui/layout.go` | `computePaneBounds(width, height)` |
| Layout update | `internal/tui/layout.go` | `Layout.Update(msg)` |
| View render | `internal/tui/layout.go` | `Layout.View(...)` |

### Diagram

```mermaid
sequenceDiagram
    participant Runtime as Bubble Tea
    participant Model as internal/tui/app.go (AppModel)
    participant Layout as internal/tui/layout.go (Layout)
    participant Sidebar as internal/tui/views/sidebar
    participant Querybar as internal/tui/views/querybar
    participant Results as internal/tui/views/results
    participant InputHandler as internal/tui/registry.go

    Runtime->>Model: Init()
    Model->>Layout: Layout.Init()
    Layout->>Sidebar: sidebar.Init()
    Layout->>Querybar: querybar.Init()
    Layout->>Results: results.Init()
    Sidebar-->>Layout: nil
    Querybar-->>Layout: textinput.Blink
    Results-->>Layout: nil
    Layout-->>Model: tea.Batch(textinput.Blink)
    Model-->>Runtime: tea.Batch(...)

    Note over Runtime: Runtime measures terminal size
    Runtime->>Model: Update(tea.WindowSizeMsg{W, H})

    activate Model
    Model->>Model: handleWindowSizeMsg(msg)

    Model->>Layout: Layout.Update(msg)

    activate Layout
    Layout->>Layout: computePaneBounds(W, H)
    Note over Layout: mainHeight = max(0, H - helpBarHeight)<br/>listW = int(0.3 * W)<br/>rightW = W - listW<br/>inputH = int(0.2 * mainH)<br/>viewH = mainH - inputH

    Layout->>Sidebar: Update(WindowSizeMsg{leftWidth, leftHeight})
    Sidebar-->>Layout: (sidebar.Model, cmd)
    Layout->>Querybar: Update(WindowSizeMsg{viewWidth, queryHeight})
    Querybar-->>Layout: (querybar.Model, cmd)
    Layout->>Results: Update(WindowSizeMsg{viewWidth, viewHeight})
    Results-->>Layout: (results.Model, cmd)

    Layout-->>Model: (updated Layout, tea.Batch)
    deactivate Layout

    Model-->>Runtime: (AppModel, tea.Batch)
    deactivate Model

    Note over Runtime: Runtime automatically calls View()

    Runtime->>Model: View()
    Model->>Layout: Layout.View(panes.Current(), m.input)

    activate Layout
    Note over Layout: paneStylesForFocus(activeFocus)<br/>computePaneSizes(width, height)

    Layout->>InputHandler: l.Help.View(keys)
    InputHandler-->>Layout: Rendered help string

    Note over Layout: listStyle.Render(sidebar.View())<br/>inputStyle.Render(querybar.View())<br/>viewStyle.Render(results.ViewStr())

    Layout-->>Model: Rendered UI string
    deactivate Layout

    Model-->>Runtime: UI string
    Runtime->>Runtime: Draw to alternate screen buffer
```

---

## 3. Command Pattern: Key Event Handling

### Source Files

| Step | File | Key Function / Type |
|------|------|-------------------|
| Message dispatch | `internal/tui/app.go` | `AppModel.Update(msg)` |
| Key routing | `internal/tui/app.go` | `handleKeyMsg(msg)` |
| Registry iteration | `internal/tui/registry.go` | `InputHandler.Actions` (slice) |
| Action execution | `internal/tui/actions.go` | `AppAction.Apply(m)` |
| Action impls | `internal/tui/actions.go` | `QuitAction`, `ToggleTableAction`, `FocusNextAction`, `FocusPrevAction`, `AutofillAction`, `ExecuteAction`, `ExecuteSourceAction` |
| Fallback routing | `internal/tui/app.go` | `routeToFocused(msg)` |

### Diagram

```mermaid
sequenceDiagram
    participant User as User
    participant Runtime as Bubble Tea
    participant Model as internal/tui/app.go (Update)
    participant Handler as internal/tui/app.go (handleKeyMsg)
    participant Reg as internal/tui/registry.go (InputHandler)
    participant Action as internal/tui/actions.go (AppAction)
    participant Child as internal/tui/views (Active Pane)

    User->>Runtime: Keystroke ('e', 'E', 'a', etc.)
    Runtime->>Model: Update(tea.KeyMsg)

    activate Model
    Model->>Handler: handleKeyMsg(msg)

    activate Handler
    Handler->>Reg: for _, mapping := range m.input.Actions

    loop Over each binding entry
        Reg->>Reg: key.Matches(msg, mapping.Binding)
        opt Binding matches
            Reg-->>Handler: Match found
            Handler->>Action: mapping.Action.Apply(m)
            activate Action
            alt ExecuteAction
                Note over Action: return m, cmd → RunQueryMsg{SQL}
            else ExecuteSourceAction
                Note over Action: return m, cmd → RunSourceQueryMsg{SQL}
            else QuitAction / ToggleTableAction / FocusNextAction / etc.
                Note over Action: State mutation only, nil cmd
            end
            Action-->>Handler: (updated AppModel, tea.Cmd)
            deactivate Action
            Note over Handler: Early return — routeToFocused is NOT called
        end
    end

    Note over Handler: Loop completed with no match
    Handler->>Child: routeToFocused(msg)
    activate Child
    Child-->>Handler: (updated child Model, tea.Cmd)
    deactivate Child

    Handler-->>Model: (updated AppModel, tea.Cmd)
    deactivate Handler

    Model-->>Runtime: (AppModel, tea.Cmd)
    deactivate Model
    Runtime->>Model: View()
    Model-->>Runtime: Rendered UI string
    Runtime->>User: Draw terminal screen
```

---

## 4. Query Result Generation Pipeline

This flow details the second half of the query lifecycle: once the background goroutine returns a `QueryResultMsg`, the results are formatted into both a line-mode viewport string and a column-based table widget, and focus shifts to the results pane. Both representations are generated side-by-side so the user can toggle between them without re-querying. The `results.Model` orchestrates formatting, stores both representations, and dispatches rendering via `ViewStr()` based on the active mode.

This flow is the same for both cached (`e`) and source (`E`) queries — both produce `QueryResultMsg` that is handled identically by `handleQueryResultMsg`.

### Source Files

| Step | File | Key Function / Type |
|------|------|-------------------|
| Result dispatch | `internal/tui/app.go` | `handleQueryResultMsg(msg)` |
| Format orchestrator | `internal/tui/views/results/results.go` | `Model` — holds View, Table, IsTableMode |
| Data formatting | `internal/tui/views/results/format.go` | `Model.FormatData(rowsData, columns)` |
| Key extraction (fallback) | `internal/tui/views/results/format.go` | `extractKeys(rowsData)` |
| Line-mode render | `internal/tui/views/results/format.go` | `formatLineMode(keys, rowsData)` |
| Column layout | `internal/tui/views/results/format.go` | `Model.computeTableLayout(keys)` |
| Row builder | `internal/tui/views/results/format.go` | `buildRows(layout, rowsData)` |
| Focus shift | `internal/tui/app.go` | `m.panes.Set(PaneResults)`, `Querybar.Blur()` |
| Mode dispatch | `internal/tui/views/results/results.go` | `Model.ViewStr()` — selects View or Table |

### Diagram

```mermaid
sequenceDiagram
    participant Runtime as Bubble Tea
    participant Model as internal/tui/app.go (AppModel)
    participant ResultsCtrl as internal/tui/views/results (Model)
    participant FormatHelpers as internal/tui/views/results/format.go

    Note over Runtime: Background goroutine completed
    Runtime->>Model: Update(QueryResultMsg{Rows, Columns})

    activate Model
    Model->>Model: handleQueryResultMsg(msg)

    alt len(msg.Rows) == 0
        Model->>ResultsCtrl: View.SetContent("0 rows returned.")
    else rows available
        Model->>ResultsCtrl: Results.FormatData(msg.Rows, msg.Columns)
        activate ResultsCtrl

        alt msg.Columns is non-empty
            Note over ResultsCtrl: keys = msg.Columns — uses schema-defined order
        else msg.Columns is nil/empty
            ResultsCtrl->>FormatHelpers: extractKeys(rowsData) — fallback
        end

        par Build line-mode viewport
            ResultsCtrl->>FormatHelpers: formatLineMode(keys, rowsData)
            activate FormatHelpers
            Note over FormatHelpers: Builds "--- Row N ---\n  key = value\n" into strings.Builder
            FormatHelpers-->>ResultsCtrl: rendered string
            deactivate FormatHelpers
            ResultsCtrl->>ResultsCtrl: m.View.SetContent(string)<br/>m.View.GotoTop()
        and Build table widget
            ResultsCtrl->>FormatHelpers: computeTableLayout(keys)
            activate FormatHelpers
            Note over FormatHelpers: Drops rightmost keys if too many<br/>Returns uniform colWidth + filtered keys
            FormatHelpers-->>ResultsCtrl: tableLayout{keys, colWidth}
            deactivate FormatHelpers

            ResultsCtrl->>FormatHelpers: buildRows(layout, rowsData)
            activate FormatHelpers
            Note over FormatHelpers: Converts raw maps to []table.Row<br/>Truncates overflow UTF-8 safe
            FormatHelpers-->>ResultsCtrl: []table.Row
            deactivate FormatHelpers

            ResultsCtrl->>ResultsCtrl: table.New(WithColumns, WithRows, ...)<br/>SetStyles(m.tableStyles)
        end

        ResultsCtrl-->>Model: updated results.Model
        deactivate ResultsCtrl
    end

    Note over Model: Shift focus to results pane
    Model->>Model: m.panes = m.panes.Set(PaneResults)
    Model->>Model: m.layout.Querybar.Blur()
    Model-->>Runtime: (AppModel, nil)
    deactivate Model

    Runtime->>Model: View()
    activate Model
    Model->>ResultsCtrl: m.layout.Results.ViewStr()
    activate ResultsCtrl
    alt IsTableMode == true
        ResultsCtrl->>ResultsCtrl: m.Table.View()
    else IsTableMode == false
        ResultsCtrl->>ResultsCtrl: m.View.View()
    end
    ResultsCtrl-->>Model: Rendered string
    deactivate ResultsCtrl

    Model-->>Runtime: Rendered UI string
    deactivate Model
    Runtime->>Runtime: Draw to alternate screen buffer
```

---

---

## 5. Pane Navigation (Focus Next/Prev)

This flow illustrates the specific execution of `FocusNextAction` when a user presses the focus next key (default: Ctrl+L) or `FocusPrevAction` (default: Ctrl+H). The action advances or reverses `activeFocus` through the three panes and manages `Focus`/`Blur` on the text input.

The diagram also shows the two-layer key routing architecture: `handleKeyMsg` first checks global bindings. If a binding matches, the corresponding `AppAction` is executed. If no global binding matches, the message falls through to `routeToFocused`, which forwards the raw `tea.Msg` to whichever child widget currently has focus — letting that widget handle it natively without an explicit binding entry.

### Source Files

| Step | File | Key Function / Type |
|------|------|-------------------|
| Cycle action | `internal/tui/actions.go` | `FocusNextAction.Apply(m)` / `FocusPrevAction.Apply(m)` |
| Focus state | `internal/tui/pane.go` | `PaneManager` struct |
| Input focus | `internal/tui/views/querybar/input.go` | `querybar.Model.Focus()` / `.Blur()` / `.EnterNormal()` |
| Render | `internal/tui/app.go` | `AppModel.View()` |

### Diagram

```mermaid
sequenceDiagram
    participant User as User
    participant Runtime as Bubble Tea
    participant Model as internal/tui/app.go (Update)
    participant Handler as internal/tui/app.go (handleKeyMsg)
    participant Reg as internal/tui/registry.go (InputHandler)
    participant Action as internal/tui/actions.go (FocusNextAction)
    participant Child as internal/tui/views (Active Pane)
    participant Layout as layout.go (Layout.View)

    User->>Runtime: Press focus next key (Ctrl+L)
    Runtime->>Model: Update(tea.KeyMsg)

    activate Model
    Model->>Handler: handleKeyMsg(msg)
    activate Handler

    Handler->>Reg: for _, mapping := range m.input.Actions

    loop Over each binding entry
        Reg->>Reg: key.Matches(msg, mapping.Binding)
        opt FocusNextAction matches
            Reg-->>Handler: Match found
            Handler->>Action: FocusNextAction.Apply(m)
            activate Action
            Action->>Action: m.panes = m.panes.Next()
            Note over Action: Cycles: PaneQuery → PaneResults → PaneSidebar → PaneQuery
            alt new focus is PaneQuery and INSERT mode
                Action->>Child: Focus()
            else new focus is PaneQuery and NORMAL mode
                Action->>Child: EnterNormal()
            else any other pane
                Action->>Child: Blur()
            end
            Action-->>Handler: (updated AppModel, nil)
            deactivate Action
        end
    end

    Note over Handler: Loop completed with no match
    Handler->>Child: routeToFocused(msg)
    activate Child
    Note over Child: Forwards raw Msg to focused widget<br/>for native handling (scrolling, list nav, etc.)
    Child-->>Handler: (updated child Model, tea.Cmd)
    deactivate Child

    Handler-->>Model: (updated AppModel, tea.Cmd)
    deactivate Handler

    Model-->>Runtime: (AppModel, nil)

    Runtime->>Model: View()
    Model->>Layout: Layout.View(panes.Current(), m.input)
    activate Layout
    Note over Layout: Focused pane gets green border (focusedPaneStyle)
    Layout-->>Model: Rendered string
    deactivate Layout

    Model-->>Runtime: UI string
    deactivate Model
    Runtime->>User: Render updated UI with highlighted pane
```

---

## 6. Mouse Event Handling

### Source Files

| Step | File | Key Function / Type |
|------|------|-------------------|
| Mouse dispatch | `internal/tui/app.go` | `handleMouseMsg(msg)` |
| Pane detection | `internal/tui/layout.go` | `Layout.PaneAt(x, y)` |
| Coordinate offset | `internal/tui/layout.go` | `Layout.MouseOffset(pane)` |
| Focus switch | `internal/tui/pane.go` | `PaneManager.Set(id)` |
| Child forward | `internal/tui/app.go` | `routeToFocused(adjustedMsg)` |
| Sidebar click/wheel | `internal/tui/views/sidebar/sidebar.go` | `itemIndexAt(y)`, `scrollBy(delta)`, double-click → `AutofillMsg` |
| Querybar click | `internal/tui/views/querybar/input.go` | `handleMouseClick(msg)` |
| Results table click | `internal/tui/views/results/results.go` | `handleTableClick(msg)` |

### Diagram

```mermaid
sequenceDiagram
    participant User as User
    participant Runtime as Bubble Tea
    participant Model as internal/tui/app.go (Update)
    participant Handler as internal/tui/app.go (handleMouseMsg)
    participant Layout as internal/tui/layout.go (Layout)
    participant Child as internal/tui/views (Active Pane)

    User->>Runtime: Left-click at (X, Y)
    Runtime->>Model: Update(tea.MouseMsg{X, Y, Left, Press})

    activate Model
    Model->>Handler: handleMouseMsg(msg)
    activate Handler

    Handler->>Layout: Layout.PaneAt(msg.X, msg.Y)
    Layout-->>Handler: PaneID (sidebar / query / results / "")

    alt pane differs from current focus
        Handler->>Handler: m.panes = m.panes.Set(pane)
        alt pane is PaneQuery and INSERT mode
            Handler->>Child: Querybar.Focus()
        else pane is PaneQuery and NORMAL mode
            Handler->>Child: Querybar.EnterNormal()
        else other pane
            Handler->>Child: Querybar.Blur()
        end
    end

    Handler->>Layout: Layout.MouseOffset(m.panes.Current())
    Layout-->>Handler: (offX, offY)

    Note over Handler: adjusted.Msg{X - offX, Y - offY}

    Handler->>Child: routeToFocused(adjusted.Msg)
    activate Child
    Note over Child: Widget handles mouse natively<br/>(list selection, text cursor, scroll)
    Child-->>Handler: (updated child, tea.Cmd)
    deactivate Child

    Handler-->>Model: (updated AppModel, tea.Cmd)
    deactivate Handler

    Model-->>Runtime: (AppModel, tea.Cmd)
    deactivate Model

    Runtime->>Model: View()
    Model-->>Runtime: Rendered UI string
    Runtime->>User: Updated UI with focused pane highlight
```
