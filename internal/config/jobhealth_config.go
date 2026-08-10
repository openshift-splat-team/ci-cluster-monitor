package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type JobHealthConfig struct {
	Prow       ProwConfig          `yaml:"prow"`
	Jobs       []JobEntry          `yaml:"jobs"`
	Thresholds JobHealthThresholds `yaml:"thresholds"`
}

type ProwConfig struct {
	URL string `yaml:"url"`
}

type JobEntry struct {
	Name      string `yaml:"name"`
	Component string `yaml:"component"`
}

type JobHealthThresholds struct {
	PassRateWarningPercent  int `yaml:"pass_rate_warning_percent"`
	PassRateCriticalPercent int `yaml:"pass_rate_critical_percent"`
	LookbackDays            int `yaml:"lookback_days"`
	MinRuns                 int `yaml:"min_runs"`
}

func LoadJobHealth(path string) (*JobHealthConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg JobHealthConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	cfg.applyJobHealthDefaults()

	if err := cfg.validateJobHealth(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return &cfg, nil
}

func (c *JobHealthConfig) applyJobHealthDefaults() {
	if c.Prow.URL == "" {
		c.Prow.URL = "https://prow.ci.openshift.org"
	}
	if c.Thresholds.PassRateWarningPercent == 0 {
		c.Thresholds.PassRateWarningPercent = 80
	}
	if c.Thresholds.PassRateCriticalPercent == 0 {
		c.Thresholds.PassRateCriticalPercent = 60
	}
	if c.Thresholds.LookbackDays == 0 {
		c.Thresholds.LookbackDays = 2
	}
	if c.Thresholds.MinRuns == 0 {
		c.Thresholds.MinRuns = 3
	}
}

func (c *JobHealthConfig) validateJobHealth() error {
	if len(c.Jobs) == 0 {
		return fmt.Errorf("no jobs configured")
	}
	for i, job := range c.Jobs {
		if job.Name == "" {
			return fmt.Errorf("job at index %d has empty name", i)
		}
	}
	if c.Thresholds.PassRateCriticalPercent >= c.Thresholds.PassRateWarningPercent {
		return fmt.Errorf("pass_rate_critical_percent (%d) must be less than pass_rate_warning_percent (%d)",
			c.Thresholds.PassRateCriticalPercent, c.Thresholds.PassRateWarningPercent)
	}
	if c.Thresholds.LookbackDays <= 0 {
		return fmt.Errorf("lookback_days must be positive")
	}
	if c.Thresholds.MinRuns <= 0 {
		return fmt.Errorf("min_runs must be positive")
	}
	return nil
}
