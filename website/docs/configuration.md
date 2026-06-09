---
sidebar_position: 3
---

# Configuration

LazyOS follows the [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html) to keep your home directory clean. Three directories matter:

| Directory | Role | Purpose |
|---|---|---|
| `~/.config/lazyos/` | **The Control Room** | Settings that tell the app how to behave. `config.yml` lives here. |
| `~/.local/state/lazyos/` | **The Diary** | Runtime records that capture what the app did. `lazyos.log` lives here. |
| `~/.cache/lazyos/` | **The Scratchpad** | Persistent cache data. `lazyos.db` (SQLite) lives here |

LazyOS uses `spf13/viper` for configuration. A custom config file path can be specified via the `--config` flag.

## The Viper Hierarchy

When lazyos starts, Viper looks for configuration in a specific order of precedence. The highest priority wins:

1. **CLI Flags** (`--osquery-socket=/tmp/cli.em`) — Highest priority.
2. **Environment Variables** (`LAZYOS_OSQUERY_SOCKET=/tmp/env.em`)
3. **Config File** (`osquery-socket: /tmp/yaml.em`)
4. **Defaults** (`/tmp/default.em`) — Lowest priority.

## Example Configuration

```yaml
# ~/.config/lazyos/config.yml
# LazyOS reads this file automatically at startup.
# All fields are optional; defaults are compiled into the binary.

# Path to the osquery extension socket.
# Default: /tmp/osquery.em
osquery-socket: "/tmp/osquery.em"

# Maximum duration to wait for the initial osquery Thrift connection.
# Default: 2s
osquery-startup-timeout: "2s"

# Maximum duration to wait for an osquery query response. This timeout
# also sets the osquery-go socket locker's MaxWaitTime, which controls
# how long a goroutine waits for the shared Thrift socket before timing
# out. With multiple concurrent queries this may need to be longer than
# a single query's execution time.
# Default: 10s
osquery-query-timeout: "60s"

# Override the default log file path.
# Default: "" (resolved via DefaultLogPath — respects $XDG_STATE_HOME,
#          falls back to ~/.local/state/lazyos/lazyos.log)
log-file: "/tmp/lazyos.log"

# Keep the log file after the application exits.
# Default: false (log file is automatically deleted on exit)
keep-log: true

# Path to the persistent SQLite cache database. Created automatically
# on first launch.
# Default: $XDG_CACHE_HOME/lazyos/lazyos.db
cache-db-path: "/home/user/.cache/lazyos/lazyos.db"

keys:
  # execute — cached query (uses local SQLite store).
  # Equivalent to pressing e by default.
  execute: "e"

  # execute_source — source query (fetches from upstream, refreshes cache).
  # Equivalent to pressing E by default.
  execute_source: "E"

  # toggle_table — switches the results pane between line mode (column=value)
  # and table mode (scrollable columns).
  toggle_table: "t"

  # focus_next — moves keyboard focus to the next pane in the cycle:
  #   table list → query input → results pane → table list
  focus_next: "ctrl+l"

  # focus_prev — moves keyboard focus to the previous pane in the cycle:
  #   table list ← query input ← results pane ← table list
  focus_prev: "ctrl+h"

  # quit — cleanly terminates the application.
  quit: "q"
```

The config overrides integrate seamlessly into the `InputHandler` ensuring help menus and actions sync perfectly with the customized shortcuts.

## Configuration Fields

- **`osquery-socket`**: The Unix domain socket path that the osquery extension process listens on. This is the communication endpoint — LazyOS opens a Thrift RPC connection to this path to send SQL queries and receive result sets. The default `/tmp/osquery.em` is the standard path used by the osquery `--extension` flag, but can point to any osquery running with a custom socket.
- **`osquery-startup-timeout`**: The maximum duration the application waits for the initial Thrift connection to the osquery daemon. Accepts Go duration strings (`"2s"`, `"5s"`, `"500ms"`). Defaults to `2s`.
- **`osquery-query-timeout`**: The maximum duration allowed for an individual osquery query to complete before aborting. Accepts Go duration strings (`"10s"`, `"30s"`, `"1m"`). This timeout is also applied as the `MaxWaitTime` for the osquery-go socket locker, so with N concurrent goroutines waiting for the shared socket this may need to be longer than a single query's execution time. Defaults to `10s`.
- **`log-file`**: Override the default log file path. If empty, the logger resolves the path via `DefaultLogPath` — it respects `$XDG_STATE_HOME` first, then falls back to `~/.local/state/lazyos/lazyos.log`. Defaults to `""`.
- **`keep-log`**: Preserve the log file on disk after the application exits. When `false` (the default), the log file is automatically deleted during shutdown. Set to `true` to retain logs for debugging across sessions.
- **`cache-db-path`**: Override the path to the persistent SQLite cache database. If empty, the path resolves to `$XDG_CACHE_HOME/lazyos/lazyos.db` (typically `~/.cache/lazyos/lazyos.db`). Defaults to `""`.

Viper looks for a config file at `~/.config/lazyos/config.yml` at startup (or the path given by `--config`). If the file does not exist the application uses compiled-in defaults. Environment variables with the `LAZYOS_` prefix (e.g. `LAZYOS_KEEP_LOG=true`) are also read automatically.
