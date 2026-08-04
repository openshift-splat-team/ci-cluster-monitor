package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type NutanixConfig struct {
	PrismCentral PrismCentralConfig `yaml:"prism_central"`
	VMFilters    VMFilters          `yaml:"vm_filters"`
	Thresholds   VMThresholds       `yaml:"thresholds"`
}

type PrismCentralConfig struct {
	Endpoint string `yaml:"endpoint"`
	Port     string `yaml:"port"`
	Insecure bool   `yaml:"insecure"`
}

type VMFilters struct {
	NamePrefixes []string `yaml:"name_prefixes"`
}

type VMThresholds struct {
	OrphanTTLHours int `yaml:"orphan_ttl_hours"`
}

func LoadNutanix(path string) (*NutanixConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg NutanixConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	cfg.applyDefaults()

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return &cfg, nil
}

func (c *NutanixConfig) applyDefaults() {
	if c.PrismCentral.Port == "" {
		c.PrismCentral.Port = "9440"
	}
	if len(c.VMFilters.NamePrefixes) == 0 {
		c.VMFilters.NamePrefixes = []string{"ci-op-"}
	}
	if c.Thresholds.OrphanTTLHours == 0 {
		c.Thresholds.OrphanTTLHours = 8
	}
}

func (c *NutanixConfig) validate() error {
	if c.PrismCentral.Endpoint == "" {
		return fmt.Errorf("prism_central.endpoint is required")
	}
	if c.Thresholds.OrphanTTLHours <= 0 {
		return fmt.Errorf("orphan_ttl_hours must be positive")
	}
	if len(c.VMFilters.NamePrefixes) == 0 {
		return fmt.Errorf("at least one name prefix is required")
	}
	return nil
}
