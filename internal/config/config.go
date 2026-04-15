package config

import (
	"fmt"
	"os"
	"time"
	"gopkg.in/yaml.v3"
)

type Target struct {
	Name 		string 		`yaml:"name"`
	URL 		string 		`yaml:"url"`
	Host 		string 		`yaml:"host,omitempty"`
	Type 		string 		`yaml:"type"`
	Timeout string 		`yaml:"timeout,omitempty"`
}

type Config struct {
	GlobalTimeout	 time.Duration	`yaml:"global_timeout"`
	Targets 			[]Target				`yaml:"targets"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("parse config %w", err)
	}

	// Set global timeout for targets /wo an explicit one
	for i := range config.Targets {
		if config.Targets[i].Timeout == 0 {
			config.Targets[i].Timeout = config.GlobalTimeout
		}
	}
}