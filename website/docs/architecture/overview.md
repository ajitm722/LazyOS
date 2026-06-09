---
sidebar_position: 2
---

# High-Level Overview

LazyOS serves as a bridge between a terminal user and the host operating system. The architecture is divided into a frontend interface (the TUI), a caching layer, and a set of backend data source connectors. Multiple backends may be active simultaneously (e.g., kernel and AWS tables, both served by the same osquery daemon), each implementing the `Queryer` interface defined in `internal/daemons/`.

The application employs strict unidirectional data flow. The TUI never communicates directly with backends — all queries pass through the `CachedQueryer` decorator, which routes them to either the local SQLite store or the upstream data source depending on the user's intent. This keeps the UI responsive while isolating it from backend latency.

Backend instantiation is wired in `cmd/lazyos/root.go` via the `bootstrapBackends` function, which maintains a typed registry of backend initializers (`backendInit{key, fn}`). The `--backend` flag (repeatable, default `kernel`) controls which backends are registered. Adding a new backend requires one entry in the `available` slice and a new sub-package under `internal/daemons/osqueryd/`.

Backend cycling is handled by `NextBackendAction` (bound to `B` by default, overridable via `next_backend` in config). It cycles through the ordered `backendOrder` slice, swaps sidebar schema, and fires a resize event.

## System Components

| Component | Description |
|---|---|
| **User** | The operator interacting with the application via terminal keystrokes to navigate, construct queries, and review data. |
| **LazyOS TUI** | The primary application implemented in Go. Uses the Bubble Tea framework to render an interactive three-pane layout. Manages user state, input routing, and communicates with the caching layer. |
| **CachedQueryer** | A `Queryer` decorator that intercepts all query calls. `e` (cached) executes against the local store, lazy-loading missing tables from the upstream backend. `E` (source) fetches from the upstream and refreshes the store. |
| **SQLite Store** | An embedded persistent cache at `~/.cache/lazyos/lazyos.db`. Stores full table snapshots as SQL tables. All values stored as `TEXT`. WAL journal mode is enabled for concurrent read/write performance. |
| **osquery Daemon** | An external background service exposing operating system metrics (processes, network, users) as SQL tables. When extended with the cloudquery extension, additionally exposes cloud resource tables. LazyOS communicates with it via Thrift RPC over a Unix domain socket. |
