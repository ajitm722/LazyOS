// Package config provides the configuration types that map to the user's config
// file (~/.config/lazyos/config.yml) and CLI flags.
package config

import "time"

// Keys holds the user-configurable keybinding overrides. Each field maps to a
// semantic action name in the YAML and is applied by NewKeyMap at startup.
type Keys struct {
	ToggleTable string `mapstructure:"toggle_table"`
	FocusNext   string `mapstructure:"focus_next"`
	FocusPrev   string `mapstructure:"focus_prev"`
	Enter       string `mapstructure:"enter"`
	Quit        string `mapstructure:"quit"`
}

// Config is the top-level application configuration. It is deserialized from
// the config file or CLI flags and passed to the TUI constructor.
type Config struct {
	OsquerySocket         string        `mapstructure:"osquery-socket"`
	OsqueryStartupTimeout time.Duration `mapstructure:"osquery-startup-timeout"`
	OsqueryQueryTimeout   time.Duration `mapstructure:"osquery-query-timeout"`
	LogFile               string        `mapstructure:"log-file"`
	KeepLog               bool          `mapstructure:"keep-log"`
	Keys                  Keys          `mapstructure:"keys"`
}
