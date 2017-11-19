package config

import (
	"github.com/kelseyhightower/envconfig"
)

// Config represents the application's configuration file.
type Config struct {
	TillerHost        string
	TillerNamespace   string `default:"kube-system"`
	ReleaseTimeoutMin int64  `default:"5"`
	DebugMode         bool   `envconfig:"DEBUG"`
}

// New returns the application configuration specified by the config file.
func New() (*Config, error) {
	c := &Config{}
	if err := envconfig.Process("chartmgr", c); err != nil {
		return nil, err
	}

	return c, nil
}
