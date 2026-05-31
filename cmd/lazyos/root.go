// Package main implements the Cobra command that bootstraps the entire
// application: reading config, connecting to backend data sources, and
// launching the TUI.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ajitm722/LazyOS/internal/cache"
	"github.com/ajitm722/LazyOS/internal/config"
	"github.com/ajitm722/LazyOS/internal/daemons"
	"github.com/ajitm722/LazyOS/internal/daemons/osqueryd/aws"
	"github.com/ajitm722/LazyOS/internal/daemons/osqueryd/kernel"
	"github.com/ajitm722/LazyOS/internal/logger"
	"github.com/ajitm722/LazyOS/internal/store/sqlite"
	"github.com/ajitm722/LazyOS/internal/tui"
)

var cfgFile string

// initConfig locates and parses the config file. It returns an error when the
// file exists but cannot be parsed (e.g., invalid YAML). A missing file is not
// an error — the caller may rely on flags and environment variables alone.
func initConfig() error {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		// It is not critical if UserConfigDir fails; we just skip loading the
		// file and rely on environment variables and flags.
		if configDir, err := os.UserConfigDir(); err == nil {
			viper.AddConfigPath(filepath.Join(configDir, "lazyos"))
		}
		viper.SetConfigType("yml")
		viper.SetConfigName("config")
	}

	viper.SetEnvPrefix("LAZYOS")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	return nil
}

// runApp is the core execution pipeline. It manages the lifecycle of resources,
// ensuring everything is cleanly initialized before the TUI starts, and properly
// torn down when the TUI exits.
func runApp(ctx context.Context, cfg config.Config) error {
	// 1. Initialize Logger
	log, logFile, finalLogPath, err := logger.SetupFile(cfg.LogFile)
	if err != nil {
		return err
	}
	defer func() {
		if err := logFile.Close(); err != nil {
			log.Error("Failed to close log file", "error", err)
		}
		if !cfg.KeepLog {
			if err := os.Remove(finalLogPath); err != nil && !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "Failed to cleanup log file: %v\n", err)
			}
		}
	}()

	// 2. Initialize Backends
	clients, backendOrder, err := bootstrapBackends(cfg)
	if err != nil {
		return fmt.Errorf("failed to bootstrap backends: %w", err)
	}
	defer func() {
		for _, c := range clients {
			if err := c.Close(); err != nil {
				log.Error("Failed to close backend client", "error", err)
			}
		}
	}()

	// 3. Initialize persistent cache
	dbPath := cfg.CacheDBPath
	if dbPath == "" {
		cacheDir, err := os.UserCacheDir()
		if err != nil {
			cacheDir = filepath.Join(os.Getenv("HOME"), ".cache")
		}
		dbPath = filepath.Join(cacheDir, "lazyos", "lazyos.db")
	}

	st, err := sqlite.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open sqlite store: %w", err)
	}
	defer st.Close()

	for name, client := range clients {
		clients[name] = cache.NewCachedQueryer(client, st)
	}
	log.Info("persistent store opened", "path", dbPath)

	// 4. Start Application
	return startTUI(ctx, clients, backendOrder, cfg.Keys)
}

// startTUI initializes and runs the Bubble Tea application loop.
func startTUI(ctx context.Context, clients map[string]daemons.Queryer, backendOrder []string, keys config.Keys) error {
	p := tea.NewProgram(tui.NewApp(clients, backendOrder, keys), tea.WithAltScreen(), tea.WithContext(ctx), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("tui execution failed: %w", err)
	}
	return nil
}

// backendInit pairs a flag key with its initializer.
type backendInit struct {
	key string
	fn  func(config.Config) (string, daemons.Queryer, error)
}

// bootstrapBackends iterates the available initializers, filtered by the
// --backend flag, and builds a name-to-Queryer map along with an ordered
// list of backend names for UI cycling.
func bootstrapBackends(cfg config.Config) (map[string]daemons.Queryer, []string, error) {
	backends := cfg.Backends
	if len(backends) == 0 {
		backends = []string{"kernel"}
	}

	enabled := make(map[string]bool, len(backends))
	for _, b := range backends {
		enabled[strings.ToLower(strings.TrimSpace(b))] = true
	}

	// available defines the registration order and the mapping from flag
	// keys to init functions. Adding a new backend is a single entry here.
	available := []backendInit{
		{key: "kernel", fn: kernel.InitFromConfig},
		{key: "aws", fn: aws.InitFromConfig},
	}

	clients := make(map[string]daemons.Queryer)
	var order []string

	for _, entry := range available {
		if !enabled[entry.key] {
			continue
		}
		name, client, err := entry.fn(cfg)
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

// Execute builds the root Cobra command and starts the application.
func Execute(ctx context.Context) error {
	var cfg config.Config

	// rootCmd defines the top-level CLI command broken down into lifecycle hooks.
	rootCmd := &cobra.Command{
		Use:   "lazyos",
		Short: "A TUI for exploring live system data across multiple backends",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := initConfig(); err != nil {
				return err
			}
			return viper.Unmarshal(&cfg)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runApp(cmd.Context(), cfg)
		},
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/lazyos/config.yml)")

	rootCmd.Flags().String("osquery-socket", "/tmp/osquery.em", "Path to the osquery extension socket")
	rootCmd.Flags().Duration("osquery-startup-timeout", 10*time.Second, "Timeout for the initial osquery Thrift connection")
	rootCmd.Flags().Duration("osquery-query-timeout", 100*time.Second, "Timeout for individual osquery queries")
	rootCmd.Flags().String("log-file", "", "Override default log file path")
	rootCmd.Flags().Bool("keep-log", false, "Keep the log file after exit")
	rootCmd.Flags().StringSlice("backend", []string{"kernel"}, "Backends to enable: kernel, aws")
	rootCmd.Flags().String("cache-db-path", "", "Path to the cache SQLite database")

	if err := viper.BindPFlags(rootCmd.Flags()); err != nil {
		return err
	}

	return rootCmd.ExecuteContext(ctx)
}
