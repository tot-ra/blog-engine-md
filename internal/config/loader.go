package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Load reads and parses the configuration file
func Load(path string) (*SiteConfig, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := Validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
