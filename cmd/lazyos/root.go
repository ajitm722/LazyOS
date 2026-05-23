// Package main implements the Cobra command that bootstraps the entire
// application: reading config, connecting to osquery, and launching the TUI.
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

	"github.com/ajitm722/lazyos/internal/config"
	"github.com/ajitm722/lazyos/internal/daemons"
	"github.com/ajitm722/lazyos/internal/daemons/osquery"
	"github.com/ajitm722/lazyos/internal/logger"
	"github.com/ajitm722/lazyos/internal/tui"
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

	// 2. Initialize Daemons
	clients, err := bootstrapDaemons(cfg)
	if err != nil {
		return fmt.Errorf("failed to bootstrap daemons: %w", err)
	}
	defer func() {
		for _, c := range clients {
			if err := c.Close(); err != nil {
				log.Error("Failed to close daemon client", "error", err)
			}
		}
	}()

	// 3. Start Application
	return startTUI(ctx, clients, cfg.Keys)
}

// startTUI initializes and runs the Bubble Tea application loop.
func startTUI(ctx context.Context, clients map[string]daemons.Queryer, keys config.Keys) error {
	p := tea.NewProgram(tui.NewApp(clients, keys), tea.WithAltScreen(), tea.WithContext(ctx))
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("tui execution failed: %w", err)
	}
	return nil
}

// bootstrapDaemons iterates the available initializers and builds a name-to-Queryer map.
func bootstrapDaemons(cfg config.Config) (map[string]daemons.Queryer, error) {
	// available contains the initialization functions for all supported backends.
	available := []func(cfg config.Config) (name string, client daemons.Queryer, err error){
		osquery.InitFromConfig,
	}

	clients := make(map[string]daemons.Queryer)

	for _, initFn := range available {
		name, client, err := initFn(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize %s: %w", name, err)
		}
		if client != nil && name != "" {
			clients[name] = client
		}
	}

	return clients, nil
}

// Execute builds the root Cobra command and starts the application.
func Execute(ctx context.Context) error {
	var cfg config.Config

	// rootCmd defines the top-level CLI command broken down into lifecycle hooks.
	rootCmd := &cobra.Command{
		Use:   "lazyos",
		Short: "A TUI for real-time osquery system visualization",
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
	rootCmd.Flags().Duration("osquery-startup-timeout", 2*time.Second, "Timeout for the initial osquery Thrift connection")
	rootCmd.Flags().Duration("osquery-query-timeout", 10*time.Second, "Timeout for individual osquery queries")
	rootCmd.Flags().String("log-file", "", "Override default log file path")
	rootCmd.Flags().Bool("keep-log", false, "Keep the log file after exit")

	if err := viper.BindPFlags(rootCmd.Flags()); err != nil {
		return err
	}

	return rootCmd.ExecuteContext(ctx)
}
