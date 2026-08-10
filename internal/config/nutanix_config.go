package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const DefaultNamePrefix = "ci-op-"

type NutanixConfig struct {
	PrismCentral       PrismCentralConfig `yaml:"prism_central"`
	VMFilters          VMFilters          `yaml:"vm_filters"`
	Thresholds         VMThresholds       `yaml:"thresholds"`
	CapacityMonitoring CapacityMonitoring `yaml:"capacity_monitoring"`
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

type CapacityMonitoring struct {
	ClusterUUIDs       []string           `yaml:"cluster_uuids"`
	CapacityThresholds CapacityThresholds `yaml:"thresholds"`
}

type CapacityThresholds struct {
	CPUWarningPercent     int `yaml:"cpu_warning_percent"`
	CPUCriticalPercent    int `yaml:"cpu_critical_percent"`
	MemoryWarningPercent  int `yaml:"memory_warning_percent"`
	MemoryCriticalPercent int `yaml:"memory_critical_percent"`
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
		c.VMFilters.NamePrefixes = []string{DefaultNamePrefix}
	}
	if c.Thresholds.OrphanTTLHours == 0 {
		c.Thresholds.OrphanTTLHours = 8
	}
	if c.CapacityMonitoring.CapacityThresholds.CPUWarningPercent == 0 {
		c.CapacityMonitoring.CapacityThresholds.CPUWarningPercent = 70
	}
	if c.CapacityMonitoring.CapacityThresholds.CPUCriticalPercent == 0 {
		c.CapacityMonitoring.CapacityThresholds.CPUCriticalPercent = 85
	}
	if c.CapacityMonitoring.CapacityThresholds.MemoryWarningPercent == 0 {
		c.CapacityMonitoring.CapacityThresholds.MemoryWarningPercent = 70
	}
	if c.CapacityMonitoring.CapacityThresholds.MemoryCriticalPercent == 0 {
		c.CapacityMonitoring.CapacityThresholds.MemoryCriticalPercent = 85
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
	if err := c.CapacityMonitoring.CapacityThresholds.validate(); err != nil {
		return fmt.Errorf("capacity_monitoring.thresholds: %w", err)
	}
	return nil
}

func (t *CapacityThresholds) validate() error {
	if t.CPUWarningPercent <= 0 || t.CPUWarningPercent > 100 {
		return fmt.Errorf("cpu_warning_percent must be between 1 and 100")
	}
	if t.CPUCriticalPercent <= 0 || t.CPUCriticalPercent > 100 {
		return fmt.Errorf("cpu_critical_percent must be between 1 and 100")
	}
	if t.CPUWarningPercent >= t.CPUCriticalPercent {
		return fmt.Errorf("cpu_warning_percent (%d) must be less than cpu_critical_percent (%d)",
			t.CPUWarningPercent, t.CPUCriticalPercent)
	}
	if t.MemoryWarningPercent <= 0 || t.MemoryWarningPercent > 100 {
		return fmt.Errorf("memory_warning_percent must be between 1 and 100")
	}
	if t.MemoryCriticalPercent <= 0 || t.MemoryCriticalPercent > 100 {
		return fmt.Errorf("memory_critical_percent must be between 1 and 100")
	}
	if t.MemoryWarningPercent >= t.MemoryCriticalPercent {
		return fmt.Errorf("memory_warning_percent (%d) must be less than memory_critical_percent (%d)",
			t.MemoryWarningPercent, t.MemoryCriticalPercent)
	}
	return nil
}
