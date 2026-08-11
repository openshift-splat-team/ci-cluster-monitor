package config

import (
	"testing"
)

func TestLoadJobHealth(t *testing.T) {
	content := `
prow:
  url: "https://prow.ci.openshift.org"

jobs:
  - name: periodic-ci-openshift-splat-team-test-job
    component: nutanix

thresholds:
  pass_rate_warning_percent: 75
  pass_rate_critical_percent: 50
  lookback_days: 3
  min_runs: 5
`
	path := writeTestFile(t, content)

	cfg, err := LoadJobHealth(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(cfg.Jobs))
	}
	if cfg.Jobs[0].Name != "periodic-ci-openshift-splat-team-test-job" {
		t.Errorf("expected job name periodic-ci-openshift-splat-team-test-job, got %s", cfg.Jobs[0].Name)
	}
	if cfg.Jobs[0].Component != "nutanix" {
		t.Errorf("expected component nutanix, got %s", cfg.Jobs[0].Component)
	}
	if cfg.Thresholds.PassRateWarningPercent != 75 {
		t.Errorf("expected warning 75, got %d", cfg.Thresholds.PassRateWarningPercent)
	}
	if cfg.Thresholds.PassRateCriticalPercent != 50 {
		t.Errorf("expected critical 50, got %d", cfg.Thresholds.PassRateCriticalPercent)
	}
	if cfg.Thresholds.LookbackDays != 3 {
		t.Errorf("expected lookback 3, got %d", cfg.Thresholds.LookbackDays)
	}
	if cfg.Thresholds.MinRuns != 5 {
		t.Errorf("expected min_runs 5, got %d", cfg.Thresholds.MinRuns)
	}
}

func TestLoadJobHealthAppliesDefaults(t *testing.T) {
	content := `
jobs:
  - name: test-job
`
	path := writeTestFile(t, content)

	cfg, err := LoadJobHealth(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Prow.URL != "https://prow.ci.openshift.org" {
		t.Errorf("expected default prow URL, got %s", cfg.Prow.URL)
	}
	if cfg.Thresholds.PassRateWarningPercent != 80 {
		t.Errorf("expected default warning 80, got %d", cfg.Thresholds.PassRateWarningPercent)
	}
	if cfg.Thresholds.PassRateCriticalPercent != 60 {
		t.Errorf("expected default critical 60, got %d", cfg.Thresholds.PassRateCriticalPercent)
	}
	if cfg.Thresholds.LookbackDays != 2 {
		t.Errorf("expected default lookback 2, got %d", cfg.Thresholds.LookbackDays)
	}
	if cfg.Thresholds.MinRuns != 3 {
		t.Errorf("expected default min_runs 3, got %d", cfg.Thresholds.MinRuns)
	}
}

func TestLoadJobHealthValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "no jobs",
			content: "jobs: []",
		},
		{
			name: "empty job name",
			content: `
jobs:
  - name: ""
`,
		},
		{
			name: "critical >= warning",
			content: `
jobs:
  - name: test-job
thresholds:
  pass_rate_warning_percent: 60
  pass_rate_critical_percent: 80
`,
		},
		{
			name: "warning above 100",
			content: `
jobs:
  - name: test-job
thresholds:
  pass_rate_warning_percent: 101
  pass_rate_critical_percent: 60
`,
		},
		{
			name: "critical above 100",
			content: `
jobs:
  - name: test-job
thresholds:
  pass_rate_warning_percent: 80
  pass_rate_critical_percent: 101
`,
		},
		{
			name: "negative warning",
			content: `
jobs:
  - name: test-job
thresholds:
  pass_rate_warning_percent: -1
  pass_rate_critical_percent: 60
`,
		},
		{
			name: "negative critical",
			content: `
jobs:
  - name: test-job
thresholds:
  pass_rate_warning_percent: 80
  pass_rate_critical_percent: -5
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTestFile(t, tt.content)
			_, err := LoadJobHealth(path)
			if err == nil {
				t.Error("expected validation error, got nil")
			}
		})
	}
}
