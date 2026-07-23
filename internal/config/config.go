// Copyright 2025.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Repositories []Repository `yaml:"repositories"`
	Thresholds   Thresholds   `yaml:"thresholds"`
	Filters      Filters      `yaml:"filters"`
}

type Repository struct {
	Owner     string   `yaml:"owner"`
	Name      string   `yaml:"name"`
	Workflows []string `yaml:"workflows"`
}

type Thresholds struct {
	WarningAgeDays  int `yaml:"warning_age_days"`
	CriticalAgeDays int `yaml:"critical_age_days"`
	StaleUpdateDays int `yaml:"stale_update_days"`
}

type Filters struct {
	ExcludeDrafts  bool     `yaml:"exclude_drafts"`
	ExcludeLabels  []string `yaml:"exclude_labels"`
	ExcludeAuthors []string `yaml:"exclude_authors"`
}

func (r Repository) FullName() string {
	return r.Owner + "/" + r.Name
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	cfg.applyDefaults()

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Thresholds.WarningAgeDays == 0 {
		c.Thresholds.WarningAgeDays = 7
	}
	if c.Thresholds.CriticalAgeDays == 0 {
		c.Thresholds.CriticalAgeDays = 14
	}
	if c.Thresholds.StaleUpdateDays == 0 {
		c.Thresholds.StaleUpdateDays = 3
	}
}

func (c *Config) validate() error {
	if len(c.Repositories) == 0 {
		return fmt.Errorf("no repositories configured")
	}
	for i, repo := range c.Repositories {
		if repo.Owner == "" || repo.Name == "" {
			return fmt.Errorf("repository at index %d has empty owner or name", i)
		}
	}
	if c.Thresholds.WarningAgeDays >= c.Thresholds.CriticalAgeDays {
		return fmt.Errorf("warning_age_days (%d) must be less than critical_age_days (%d)",
			c.Thresholds.WarningAgeDays, c.Thresholds.CriticalAgeDays)
	}
	return nil
}
