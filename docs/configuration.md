# Configuration

LazyOS follows the [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html) to keep your home directory clean. Three directories matter:

| Directory | Role | Purpose |
|---|---|---|
| `~/.config/lazyos/` | **The Control Room** | Settings that tell the app how to behave. `config.yml` lives here. |
| `~/.local/state/lazyos/` | **The Diary** | Runtime records that capture what the app did. `lazyos.log` lives here. |
| `~/.cache/lazyos/` | **The Scratchpad** | Temporary data that speeds things up but can be safely deleted at any time without data loss. |

LazyOS uses `spf13/viper` for configuration. A custom config file path can be specified via the `--config` flag.

### The Viper Hierarchy

When lazyos starts, Viper looks for configuration in a specific order of precedence. The highest priority wins:

1. **CLI Flags** (`--osquery-socket=/tmp/cli.em`) — Highest priority.
2. **Environment Variables** (`LAZYOS_OSQUERY_SOCKET=/tmp/env.em`)
3. **Config File** (`osquery-socket: /tmp/yaml.em`)
4. **Defaults** (`/tmp/default.em`) — Lowest priority.

## Example configuration

```yaml
# ~/.config/lazyos/config.yml
# LazyOS reads this file automatically at startup.
# All fields are optional; defaults are compiled into the binary.

# Path to the osquery extension socket.
# Default: /tmp/osquery.em
osquery-socket: "/tmp/osquery.em"

# Maximum duration to wait for the initial osquery Thrift connection.
# Accepts Go duration strings (e.g. "2s", "5s", "500ms").
# Default: 2s
osquery-startup-timeout: "2s"

# Maximum duration to wait for an osquery query response.
# Accepts Go duration strings (e.g. "5s", "1m", "500ms").
# Default: 10s
osquery-query-timeout: "10s"

# Override the default log file path.
# Default: "" (resolved via DefaultLogPath — respects $XDG_STATE_HOME,
#          falls back to ~/.local/state/lazyos/lazyos.log)
log-file: "/tmp/lazyos.log"

# Keep the log file after the application exits.
# Default: false (log file is automatically deleted on exit)
keep-log: true

keys:
  # toggle_table — switches the results pane between line mode (column=value)
  # and table mode (scrollable columns). Equivalent to pressing Ctrl+N by default.
  toggle_table: "c"

  # focus_next — moves keyboard focus to the next pane in the cycle:
  #   table list → query input → results pane → table list
  # Tab is the default — it never conflicts with query/table-name input.
  focus_next: "n"

  # focus_prev — moves keyboard focus to the previous pane in the cycle:
  #   table list ← query input ← results pane ← table list
  # Shift+Tab is the default — it never conflicts with query/table-name input.
  focus_prev: "m"

  # enter — custom key to trigger query execution or table autofill.
  # "ctrl+e" is the default.
  enter: "e"

  # quit — cleanly terminates the application.
  # Equivalent to pressing Ctrl+C by default.
  quit: "q"
```

The config overrides integrate seamlessly into the `InputHandler` ensuring help menus and actions sync perfectly with the customized shortcuts.

## Configuration Fields

* **`osquery-socket`**: The Unix domain socket path that the osquery extension process listens on. This is the communication endpoint — LazyOS opens a Thrift RPC connection to this path to send SQL queries and receive result sets. The default `/tmp/osquery.em` is the standard path used by the osquery `--extension` flag, but can point to any osquery running with a custom socket.
* **`osquery-startup-timeout`**: The maximum duration the application waits for the initial Thrift connection to the osquery daemon. Accepts Go duration strings (`"2s"`, `"5s"`, `"500ms"`). Defaults to `2s`.
* **`osquery-query-timeout`**: The maximum duration allowed for an individual osquery query to complete before aborting. Accepts Go duration strings (`"10s"`, `"30s"`, `"1m"`). This prevents the TUI from hanging indefinitely if osquery is slow or unresponsive (e.g., during heavy system introspection). Defaults to `10s`.
* **`log-file`**: Override the default log file path. If empty, the logger resolves the path via `DefaultLogPath` — it respects `$XDG_STATE_HOME` first, then falls back to `~/.local/state/lazyos/lazyos.log`. Defaults to `""`.
* **`keep-log`**: Preserve the log file on disk after the application exits. When `false` (the default), the log file is automatically deleted during shutdown. Set to `true` to retain logs for debugging across sessions.

Viper looks for a config file at `~/.config/lazyos/config.yml` at startup (or the path given by `--config`). If the file does not exist the application uses compiled-in defaults. Environment variables with the `LAZYOS_` prefix (e.g. `LAZYOS_KEEP_LOG=true`) are also read automatically.
