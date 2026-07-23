// Copyright 2025.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	content := `
repositories:
  - owner: nutanix-cloud-native
    name: test-repo
    workflows:
      - ci.yaml

thresholds:
  warning_age_days: 5
  critical_age_days: 10
  stale_update_days: 2

filters:
  exclude_drafts: true
  exclude_authors:
    - "bot[bot]"
`
	path := writeTestFile(t, content)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Repositories) != 1 {
		t.Fatalf("expected 1 repository, got %d", len(cfg.Repositories))
	}
	if cfg.Repositories[0].FullName() != "nutanix-cloud-native/test-repo" {
		t.Errorf("expected nutanix-cloud-native/test-repo, got %s", cfg.Repositories[0].FullName())
	}
	if cfg.Thresholds.WarningAgeDays != 5 {
		t.Errorf("expected warning 5, got %d", cfg.Thresholds.WarningAgeDays)
	}
	if cfg.Thresholds.CriticalAgeDays != 10 {
		t.Errorf("expected critical 10, got %d", cfg.Thresholds.CriticalAgeDays)
	}
	if !cfg.Filters.ExcludeDrafts {
		t.Error("expected exclude_drafts true")
	}
}

func TestLoadAppliesDefaults(t *testing.T) {
	content := `
repositories:
  - owner: org
    name: repo
`
	path := writeTestFile(t, content)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Thresholds.WarningAgeDays != 7 {
		t.Errorf("expected default warning 7, got %d", cfg.Thresholds.WarningAgeDays)
	}
	if cfg.Thresholds.CriticalAgeDays != 14 {
		t.Errorf("expected default critical 14, got %d", cfg.Thresholds.CriticalAgeDays)
	}
	if cfg.Thresholds.StaleUpdateDays != 3 {
		t.Errorf("expected default stale 3, got %d", cfg.Thresholds.StaleUpdateDays)
	}
}

func TestLoadValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "no repositories",
			content: "repositories: []",
		},
		{
			name: "empty owner",
			content: `
repositories:
  - owner: ""
    name: repo
`,
		},
		{
			name: "warning >= critical",
			content: `
repositories:
  - owner: org
    name: repo
thresholds:
  warning_age_days: 14
  critical_age_days: 7
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTestFile(t, tt.content)
			_, err := Load(path)
			if err == nil {
				t.Error("expected validation error, got nil")
			}
		})
	}
}

func writeTestFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writing test file: %v", err)
	}
	return path
}
