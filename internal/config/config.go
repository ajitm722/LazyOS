package config

import "time"

type Keys struct {
	ToggleTable   string `mapstructure:"toggle_table"`
	FocusNext     string `mapstructure:"focus_next"`
	FocusPrev     string `mapstructure:"focus_prev"`
	Autofill      string `mapstructure:"autofill"`
	Execute       string `mapstructure:"execute"`
	ExecuteSource string `mapstructure:"execute_source"`
	Quit          string `mapstructure:"quit"`
	NextBackend   string `mapstructure:"next_backend"`
}

type Config struct {
	OsquerySocket         string        `mapstructure:"osquery-socket"`
	OsqueryStartupTimeout time.Duration `mapstructure:"osquery-startup-timeout"`
	OsqueryQueryTimeout   time.Duration `mapstructure:"osquery-query-timeout"`
	LogFile               string        `mapstructure:"log-file"`
	KeepLog               bool          `mapstructure:"keep-log"`
	Keys                  Keys          `mapstructure:"keys"`
	Backends              []string      `mapstructure:"backend"`
	CacheDBPath           string        `mapstructure:"cache-db-path"`
}
