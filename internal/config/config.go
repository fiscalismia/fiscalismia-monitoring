package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"log/slog"
	"os"
	"time"
)

type TargetGroups struct {
	Internal []Target `yaml:"internal"`
	External []Target `yaml:"external"`
}

type Target struct {
	Name       string        `yaml:"name"`
	URL        string        `yaml:"url"`
	Host       string        `yaml:"host"`
	Type       string        `yaml:"type"`
	X509Verify bool          `yaml:"x509_verify"`
	Timeout    time.Duration `yaml:"timeout,omitempty"`
}

type Config struct {
	GlobalTimeout time.Duration `yaml:"global_timeout"`
	RootDomain    string        `yaml:"root_domain"`
	Targets       TargetGroups  `yaml:"targets"`
}

func Load(path string) (*Config, error) {
	var config Config

	data, err := os.ReadFile(path)
	if err != nil {
		slog.Error("Error reading config", "path", path)
		return nil, fmt.Errorf("Parse config %w", err)
	}

	// Unmarshal (parse & map yaml file) and allocate to go struct in memory
	if err := yaml.Unmarshal(data, &config); err != nil {
		slog.Error("Yaml unmarshaling failed.")
		return nil, fmt.Errorf("Cannot unmarshal yaml data: %v", err)
	}

	// Set global timeout for targets /wo an explicit one
	for i := range config.Targets.Internal {
		if config.Targets.Internal[i].Timeout == 0 {
			config.Targets.Internal[i].Timeout = config.GlobalTimeout
		}
	}
	for i := range config.Targets.External {
		if config.Targets.External[i].Timeout == 0 {
			config.Targets.External[i].Timeout = config.GlobalTimeout
		}
	}

	return &config, nil
}
