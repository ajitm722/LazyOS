// Package bootstrap provides shared backend initialization logic used by
// both the TUI and MCP server binaries. It lives in its own package to
// avoid the import cycle that would occur if placed in internal/config
// (config → aws/kernel → config).
package bootstrap

import (
	"fmt"
	"strings"

	"github.com/ajitm722/LazyOS/internal/config"
	"github.com/ajitm722/LazyOS/internal/daemons"
	"github.com/ajitm722/LazyOS/internal/daemons/osqueryd/aws"
	"github.com/ajitm722/LazyOS/internal/daemons/osqueryd/kernel"
)

// BackendInit pairs a flag key with its initializer function.
type BackendInit struct {
	Key string
	Fn  func(config.Config) (string, daemons.Queryer, error)
}

// AvailableBackends returns the default ordered list of backend initializers.
// Adding a new backend is a single entry here.
func AvailableBackends() []BackendInit {
	return []BackendInit{
		{Key: "kernel", Fn: kernel.InitFromConfig},
		{Key: "aws", Fn: aws.InitFromConfig},
	}
}

// BootstrapBackends iterates the available initializers, filtered by
// cfg.Backends, and builds a name-to-Queryer map along with an ordered
// list of backend names. Returns an error if any backend fails to initialize.
func BootstrapBackends(cfg config.Config) (map[string]daemons.Queryer, []string, error) {
	backends := cfg.Backends
	if len(backends) == 0 {
		backends = []string{"kernel"}
	}

	enabled := make(map[string]bool, len(backends))
	for _, b := range backends {
		enabled[strings.ToLower(strings.TrimSpace(b))] = true
	}

	clients := make(map[string]daemons.Queryer)
	var order []string

	for _, entry := range AvailableBackends() {
		if !enabled[entry.Key] {
			continue
		}
		name, client, err := entry.Fn(cfg)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to initialize %s: %w", name, err)
		}
		if client != nil && name != "" {
			clients[name] = client
			order = append(order, name)
		}
	}

	return clients, order, nil
}
