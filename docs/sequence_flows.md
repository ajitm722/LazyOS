# Sequence Flows

The following sequence diagrams detail the internal data flow and component interactions within the `tui` package during key user operations. Each diagram is annotated with the exact source files and code snippets that make the flow work, enabling a new developer to trace every message from the terminal all the way to the osquery daemon and back.

---

## 1. Application Bootstrap

This flow traces the startup path from the shell through Viper configuration loading, the construction of the AppModel (including semantic key bindings and the InputHandler), and the initialization of the Bubble Tea runtime.

### Source Files

| Step | File | Key Function / Type |
|------|------|-------------------|
| CLI entry | `cmd/lazyos/main.go:15` | `main()` |
| Command registration | `cmd/lazyos/root.go:124` | `Execute(ctx)` / `rootCmd` |
| Config loading | `cmd/lazyos/root.go:131` | `initConfig()` via `PersistentPreRunE` |
| Config schema | `internal/config/config.go:15` | `Config` struct |
| Logger setup | `cmd/lazyos/root.go:58` | `runApp` / `SetupFile` |
| Daemon bootstrap | `cmd/lazyos/root.go:102` | `bootstrapDaemons(cfg)` via `runApp` |
| TUI construction | `internal/tui/app.go:51` | `NewApp(clients, cfg.Keys)` |
| Input handling | `internal/tui/registry.go:22` | `NewInputHandler(cfg)` |
| Program start | `cmd/lazyos/root.go:93` | `startTUI(...)` calls `tea.NewProgram` |

### Diagram

```mermaid
sequenceDiagram
    participant Shell as Shell
    participant Main as cmd/lazyos/root.go (Execute)
    participant Config as cmd/lazyos/root.go (initConfig)
    participant Viper as spf13/viper
    participant Pipeline as cmd/lazyos/root.go (runApp)
    participant Logger as cmd/lazyos/root.go (SetupFile)
    participant Bootstrap as cmd/lazyos/root.go (bootstrapDaemons)
    participant Start as cmd/lazyos/root.go (startTUI)
    participant App as internal/tui/app.go (NewApp)
    participant Runtime as Bubble Tea (tea.NewProgram)

    Shell->>Main: ./lazyos
    Note over Main: Cobra parses PersistentFlags (--config)<br/>and Flags (--osquery-socket, etc.)

    Main->>Main: PersistentPreRunE fires before RunE
    Note over Main: Two-phase config load:<br/>1. initConfig() — locates file & merges env<br/>2. viper.Unmarshal(&cfg) — populates struct

    Main->>Config: initConfig()
    activate Config

    alt --config flag provided
        Config->>Viper: viper.SetConfigFile(cfgFile)
    else no --config flag
        Config->>Viper: viper.AddConfigPath(xdgConfigDir/lazyos)
        Config->>Viper: viper.SetConfigType("yml")
        Config->>Viper: viper.SetConfigName("config")
    end

    Config->>Viper: viper.SetEnvPrefix("LAZYOS")
    Config->>Viper: viper.SetEnvKeyReplacer("-" → "_")
    Config->>Viper: viper.AutomaticEnv()
    Config->>Viper: viper.ReadInConfig()
    Note over Viper: Missing file OK corrupt YAML → error returned

    Config-->>Main: nil or error
    deactivate Config

    alt initConfig returned error
        Main-->>Shell: Error terminated (invalid YAML)
    else initConfig succeeded
        Main->>Viper: viper.Unmarshal(&cfg)
        Note over Viper: Populates config.Config struct

        Main->>Pipeline: RunE calls runApp(cfg)
        activate Pipeline

        Pipeline->>Logger: logger.SetupFile(cfg.LogFile)
        activate Logger
        Note over Logger: Resolves path via DefaultLogPath (respects $XDG_STATE_HOME)<br/>Creates parent dirs with os.MkdirAll, then opens file
        Logger-->>Pipeline: log, logFile, finalLogPath
        deactivate Logger
        Note over Pipeline: defers logFile.Close() with error check<br/>defers os.Remove(finalLogPath) if !cfg.KeepLog

        Pipeline->>Bootstrap: bootstrapDaemons(cfg)
        activate Bootstrap
        Note over Bootstrap: Iterates available[] initializer slice
        Bootstrap-->>Pipeline: clients map[string]daemons.Queryer
        deactivate Bootstrap
        Note over Pipeline: defers clients close with error check

        Pipeline->>Start: startTUI(clients, cfg.Keys)
        activate Start

        Start->>App: NewApp(clients, keys)
        activate App
        Note over App: Initializes InputHandler with BoundActions
        App-->>Start: Initialized AppModel
        deactivate App

        Start->>Runtime: tea.NewProgram(AppModel, tea.WithAltScreen())
        activate Runtime
        Runtime->>App: Init()
        App-->>Runtime: tea.Batch(Child Init Cmds)
        Note over Runtime: Switches to alternate screen buffer<br/>and begins Event Loop
        deactivate Runtime

        Start-->>Pipeline: nil (TUI closed cleanly)
        deactivate Start
        deactivate Pipeline
    end
```

---

## 2. Bubble Tea Runtime Initialization

This flow shows what happens immediately after `tea.NewProgram` is called: the synthetic `WindowSizeMsg`, dimension calculation, and the first render.

### Source Files

| Step | File | Key Function / Type |
|------|------|-------------------|
| Program start | `cmd/lazyos/root.go:52` | `tea.NewProgram(...)` |
| Init command | `internal/tui/app.go:66` | `AppModel.Init()` |
| Window resize | `internal/tui/app.go:106` | `handleWindowSizeMsg(msg)` |
| Layout bounds | `internal/tui/layout.go:77` | `computePaneBounds(width, height)` |
| Layout update | `internal/tui/layout.go:89` | `Layout.Update(msg)` |
| View render | `internal/tui/layout.go:166` | `Layout.View(...)` |

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
    Note over Model: slog.Info(...)<br/>m.layout, cmd = m.layout.Update(msg)

    Model->>Layout: Layout.Update(msg)

    activate Layout
    Layout->>Layout: computePaneBounds(W, H)
    Note over Layout: mainHeight = max(0, H-helpBarHeight)<br/>listW = int(0.3*W)<br/>rightW = W - listW<br/>inputH = int(0.2*mainH)<br/>viewH = mainH - inputH<br/>leftWidth = max(0, listW - 4)<br/>leftHeight = max(0, mainH - 4)<br/>viewWidth = max(0, rightW - 4)<br/>viewHeight = max(0, viewH - 4)<br/>queryHeight = max(0, inputH - 4)

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

    Note over Layout: listStyle.Render(sidebar.View())<br/>inputStyle.Render(querybar.View())<br/>viewStyle.Render(results.ViewStr())<br/>lipgloss.JoinVertical/Horizontal

    Layout-->>Model: Rendered UI string
    deactivate Layout

    Model-->>Runtime: UI string
    Runtime->>Runtime: Draw to alternate screen buffer
```

---

## 3. Command Pattern: Key Event Handling

Keystrokes arrive as `tea.KeyMsg` values and are routed through `handleKeyMsg`, which iterates the `InputHandler.Actions` slice to find a matching `key.Binding`. Each `BoundAction` in the slice couples a binding to its corresponding `AppAction` implementation. On a match, the action's `Apply` method is invoked, mutating `AppModel` and returning an optional `tea.Cmd`. If no global binding matches, the message is forwarded to the currently focused child pane via `routeToFocused`.

### Source Files

| Step | File | Key Function / Type |
|------|------|-------------------|
| Message dispatch | `internal/tui/app.go:72` | `AppModel.Update(msg)` |
| Key routing | `internal/tui/app.go:94` | `handleKeyMsg(msg)` |
| Registry iteration | `internal/tui/registry.go:19` | `InputHandler.Actions` (slice) |
| Action execution | `internal/tui/actions.go:10` | `AppAction.Apply(m)` |
| Action impls | `internal/tui/actions.go:15-100` | `QuitAction`, `ToggleTableAction`, `FocusNextAction`, `FocusPrevAction`, `AutofillAction`, `ExecuteAction` |
| Fallback routing | `internal/tui/app.go:172` | `routeToFocused(msg)` |

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

    User->>Runtime: Keystroke (e.g., 'n')
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
            alt QuitAction
                Note over Action: return m, tea.Quit
            else ToggleTableAction
                Note over Action: m.layout.Results.IsTableMode = !...
            else FocusNextAction / FocusPrevAction
                Note over Action: m.panes = m.panes.Next()/Prev()<br/>Focus/Blur m.layout.Querybar.Input
            else AutofillAction / ExecuteAction
                Note over Action: Returns closure producing<br/>AutofillMsg or RunQueryMsg
            end
            Action-->>Handler: (updated AppModel, tea.Cmd)
            deactivate Action
            Note over Handler: Early return — routeToFocused is NOT called
        end
    end

    Note over Handler: Loop completed with no match found
    Handler->>Handler: Falls through to routeToFocused(msg)

    Handler->>Child: routeToFocused(msg)
    activate Child
    Note over Child: Routes to whichever pane is active:<br/>PaneSidebar -> m.layout.Sidebar.Update(msg)<br/>PaneQuery -> m.layout.Querybar.Update(msg)<br/>PaneResults -> m.layout.Results.Update(msg)
    Child-->>Handler: (updated child Model, tea.Cmd)
    deactivate Child

    Handler-->>Model: (updated AppModel, tea.Cmd)
    deactivate Handler

    Note over Model, Runtime: tea.Cmd is a func()

    alt tea.Cmd is non-nil (EnterAction with query or autofill)
        Model-->>Runtime: (tea.Model, tea.Cmd)
        Note over Runtime: Runtime executes the closure
        Runtime->>Runtime: closure() returns AutofillMsg / RunQueryMsg
        Runtime->>Model: Update(AutofillMsg / RunQueryMsg)
        Note over Model: Type switch matches custom Msg type<br/>Routes to handleAutofillMsg / handleRunQueryMsg
        Model-->>Runtime: (updated AppModel, new tea.Cmd)
    else tea.Cmd is nil (most actions: focus, toggle, quit plus routeToFocused)
        Model-->>Runtime: (tea.Model, nil)
    end

    Runtime->>Model: View()
    Model-->>Runtime: Rendered UI string
    Runtime->>User: Draw terminal screen
    deactivate Model
```

---

## 4. Query Result Generation Pipeline

This flow details the second half of the query lifecycle: once the background goroutine returns a `QueryResultMsg`, the results are formatted into both a line-mode viewport string and a column-based table widget, and focus shifts to the results pane. Both representations are generated side-by-side so the user can toggle between them without re-querying. The `results.Model` acts as the orchestrator — it receives data from `FormatData`, delegates formatting to `format.go`, stores both representations, and dispatches rendering via `ViewStr()` based on the active mode.

### Source Files

| Step | File | Key Function / Type |
|------|------|-------------------|
| Result dispatch | `internal/tui/app.go:149` | `handleQueryResultMsg(msg)` |
| Format orchestrator | `internal/tui/views/results/results.go:18` | `Model` — holds View, Table, IsTableMode, Width, Height |
| Data formatting | `internal/tui/views/results/format.go:23` | `Model.FormatData(rowsData, columns)` — accepts ordered column names from schema |
| Ordered key resolution | `internal/tui/views/results/format.go:26` | `keys = columns` — uses `columns` argument when non-empty; falls back to `extractKeys(rowsData)` only when nil |
| Key extraction (fallback) | `internal/tui/views/results/format.go:44` | `extractKeys(rowsData)` — map-iteration fallback (may jumble order) |
| Line-mode render | `internal/tui/views/results/format.go:58` | `formatLineMode(keys, rowsData)` |
| Column layout | `internal/tui/views/results/format.go:86` | `Model.computeTableLayout(keys)` |
| Column builder | `internal/tui/views/results/format.go:104` | `buildColumns(layout)` |
| Row builder | `internal/tui/views/results/format.go:115` | `buildRows(layout, rowsData)` |
| Table assembly | `internal/tui/views/results/format.go:138` | `Model.buildTableMode(keys, rowsData)` |
| View switching | `internal/tui/views/results/format.go:152` | `Model.FormatError(err)` |
| Focus shift | `internal/tui/app.go:157-159` | `m.panes.Set(PaneResults)`, `Input.Blur()` |
| Mode dispatch | `internal/tui/views/results/results.go:92` | `Model.ViewStr()` — selects View or Table based on IsTableMode |
| Resize handling | `internal/tui/views/results/results.go:77` | `Model.handleWindowResize()` — updates Width/Height/bounds |

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

        Note over ResultsCtrl: The results.Controller stores both a viewport (line mode)<br/>and a table.Model (table mode), plus cached styles

        alt msg.Columns is non-empty
            Note over ResultsCtrl: keys = msg.Columns — uses schema-defined order
        else msg.Columns is nil/empty
            ResultsCtrl->>FormatHelpers: extractKeys(rowsData) — fallback
            Note over FormatHelpers: Collects map keys from rowsData[0]<br/>May jumble order (Go map iteration)
        end

        par Build line-mode viewport
            ResultsCtrl->>FormatHelpers: formatLineMode(keys, rowsData)
            activate FormatHelpers
            Note over FormatHelpers: Computes max key length once<br/>Iterates rows via fmt.Fprintf into strings.Builder<br/>"--- Row 1 ---\n  name = nginx\n  pid = 1234\n"
            FormatHelpers-->>ResultsCtrl: rendered string
            deactivate FormatHelpers
            ResultsCtrl->>ResultsCtrl: m.View.SetContent(string)<br/>m.View.GotoTop()
        and Build table widget
            ResultsCtrl->>FormatHelpers: computeTableLayout(keys)
            activate FormatHelpers
            Note over FormatHelpers: max(m.Width, minViewWidth)<br/>Drops rightmost keys if too many<br/>Returns uniform colWidth + filtered keys
            FormatHelpers-->>ResultsCtrl: tableLayout{keys, colWidth}
            deactivate FormatHelpers

            ResultsCtrl->>FormatHelpers: buildColumns(layout)
            activate FormatHelpers
            Note over FormatHelpers: Turns layout into []table.Column<br/>Each gets Title=key, Width=colWidth
            FormatHelpers-->>ResultsCtrl: []table.Column
            deactivate FormatHelpers

            ResultsCtrl->>FormatHelpers: buildRows(layout, rowsData)
            activate FormatHelpers
            Note over FormatHelpers: Converts raw maps to []table.Row<br/>Truncates overflow with reflow/truncate (UTF-8 safe)
            FormatHelpers-->>ResultsCtrl: []table.Row
            deactivate FormatHelpers

            ResultsCtrl->>ResultsCtrl: table.New(WithColumns, WithRows, ...)<br/>SetStyles(m.tableStyles)
            Note over ResultsCtrl: Assembles table.Model with cached styles<br/>Stored in m.Table field
        end

        ResultsCtrl-->>Model: updated results.Model
        deactivate ResultsCtrl
    end

    Note over Model: Shift focus to results pane
    Model->>Model: m.panes = m.panes.Set(PaneResults)
    Model->>Model: m.layout.Querybar.Input.Blur()
    Model-->>Runtime: (AppModel, nil)
    deactivate Model

    Runtime->>Model: View()
    activate Model
    Model->>Model: Layout.View(panes.Current(), m.input)

    Note over Model, ResultsCtrl: Layout.View calls results.ViewStr()<br/>which checks the IsTableMode flag

    Model->>ResultsCtrl: m.layout.Results.ViewStr()
    activate ResultsCtrl
    alt IsTableMode == true
        ResultsCtrl->>ResultsCtrl: m.Table.View()
        Note over ResultsCtrl: Renders the bubbles/table widget<br/>with scrollable columns and header row
    else IsTableMode == false
        ResultsCtrl->>ResultsCtrl: m.View.View()
        Note over ResultsCtrl: Renders the viewport content<br/>with "--- Row N ---\n key = value" lines
    end
    ResultsCtrl-->>Model: Rendered string
    deactivate ResultsCtrl

    Model-->>Runtime: Rendered UI string
    deactivate Model
    Runtime->>Runtime: Draw to alternate screen buffer
```

---

## 5. Pane Navigation (Focus Next/Prev)

This flow illustrates the specific execution of `FocusNextAction` when a user presses the focus next key (default: l) or `FocusPrevAction` (default: h). The action advances/reverses `activeFocus` through the three panes and manages `Focus`/`Blur` on the text input.

The diagram also shows the two-layer key routing architecture: `handleKeyMsg` first checks a small set of global bindings (including focus navigation). If a binding matches, the corresponding `AppAction` is executed. If **no** global binding matches (e.g. typing a character, pressing Backspace, or arrow keys for scrolling), the message falls through to `routeToFocused`, which forwards the raw `tea.Msg` to whichever child widget currently has focus — letting that widget handle it natively (text input, viewport scrolling, list navigation, etc.) without needing an explicit binding entry.

### Source Files

| Step | File | Key Function / Type |
|------|------|-------------------|
| Cycle action | `internal/tui/actions.go:31` | `FocusNextAction.Apply(m)` / `FocusPrevAction.Apply(m)` |
| Focus state | `internal/tui/pane.go:12` | `PaneManager` struct |
| Input focus | `internal/tui/views/querybar/input.go:20` | `querybar.Model.Input.Focus()` / `.Blur()` |
| Render | `internal/tui/app.go:190` | `AppModel.View()` |

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

    User->>Runtime: Press focus next key (default: l)
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
            Note over Action: Cycles: PaneQuery -> PaneResults -> PaneSidebar -> PaneQuery
            alt new focus is PaneQuery
                Action->>Child: Input.Focus()
            else any other pane
                Action->>Child: Input.Blur()
            end
            Action-->>Handler: (updated AppModel, nil)
            deactivate Action
            Note over Handler: Early return — routeToFocused is NOT called
        end
    end

    Note over Handler: Loop completed with no match found
    Handler->>Handler: Falls through to routeToFocused(msg)

    Handler->>Child: routeToFocused(msg)
    activate Child
    Note over Child: Not a global binding — forwards raw Msg to<br/>the currently focused child widget for native handling<br/>(text input, scrolling, list navigation, etc.)
    Child-->>Handler: (updated child Model, tea.Cmd)
    deactivate Child

    Handler-->>Model: (updated AppModel, tea.Cmd)
    deactivate Handler

    Model-->>Runtime: (AppModel, nil)

    Runtime->>Model: View()
    Model->>Layout: Layout.View(panes.Current(), m.input)
    activate Layout
    Note over Layout: Pane with active focus gets focusedPaneStyle (green border)
    Layout-->>Model: Rendered string
    deactivate Layout

    Model-->>Runtime: UI string
    deactivate Model
    Runtime->>User: Render updated UI with highlighted pane
```
