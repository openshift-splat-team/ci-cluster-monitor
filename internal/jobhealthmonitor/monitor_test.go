package jobhealthmonitor

import (
	"testing"
	"time"

	"github.com/nutanix-cloud-native/ci-cluster-monitor/internal/config"
	"github.com/nutanix-cloud-native/ci-cluster-monitor/internal/prow"
)

func TestClassifyPassRate(t *testing.T) {
	tests := []struct {
		name     string
		passRate float64
		warning  int
		critical int
		expected Severity
	}{
		{
			name:     "healthy - well above warning",
			passRate: 95.0,
			warning:  80,
			critical: 60,
			expected: SeverityHealthy,
		},
		{
			name:     "healthy - exactly at warning",
			passRate: 80.0,
			warning:  80,
			critical: 60,
			expected: SeverityHealthy,
		},
		{
			name:     "warning - just below warning",
			passRate: 79.9,
			warning:  80,
			critical: 60,
			expected: SeverityWarning,
		},
		{
			name:     "warning - between warning and critical",
			passRate: 70.0,
			warning:  80,
			critical: 60,
			expected: SeverityWarning,
		},
		{
			name:     "warning - exactly at critical",
			passRate: 60.0,
			warning:  80,
			critical: 60,
			expected: SeverityWarning,
		},
		{
			name:     "critical - below critical",
			passRate: 50.0,
			warning:  80,
			critical: 60,
			expected: SeverityCritical,
		},
		{
			name:     "critical - zero pass rate",
			passRate: 0.0,
			warning:  80,
			critical: 60,
			expected: SeverityCritical,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyPassRate(tt.passRate, tt.warning, tt.critical)
			if got != tt.expected {
				t.Errorf("classifyPassRate(%.1f, %d, %d) = %v, want %v",
					tt.passRate, tt.warning, tt.critical, got, tt.expected)
			}
		})
	}
}

func TestCalcPassRate(t *testing.T) {
	tests := []struct {
		name     string
		passed   int
		total    int
		expected float64
	}{
		{name: "100%", passed: 10, total: 10, expected: 100.0},
		{name: "50%", passed: 5, total: 10, expected: 50.0},
		{name: "0%", passed: 0, total: 10, expected: 0.0},
		{name: "zero total", passed: 0, total: 0, expected: 0.0},
		{name: "negative total", passed: 0, total: -1, expected: 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calcPassRate(tt.passed, tt.total)
			if got != tt.expected {
				t.Errorf("calcPassRate(%d, %d) = %f, want %f", tt.passed, tt.total, got, tt.expected)
			}
		})
	}
}

func TestCountConsecutiveFailures(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		runs     []prow.JobRun
		expected int
	}{
		{
			name:     "empty runs",
			runs:     nil,
			expected: 0,
		},
		{
			name: "all success",
			runs: []prow.JobRun{
				{State: prow.JobStateSuccess, StartTime: now},
				{State: prow.JobStateSuccess, StartTime: now.Add(-time.Hour)},
			},
			expected: 0,
		},
		{
			name: "two consecutive failures then success",
			runs: []prow.JobRun{
				{State: prow.JobStateFailure, StartTime: now},
				{State: prow.JobStateError, StartTime: now.Add(-time.Hour)},
				{State: prow.JobStateSuccess, StartTime: now.Add(-2 * time.Hour)},
			},
			expected: 2,
		},
		{
			name: "aborted runs are skipped",
			runs: []prow.JobRun{
				{State: prow.JobStateAborted, StartTime: now},
				{State: prow.JobStateFailure, StartTime: now.Add(-time.Hour)},
				{State: prow.JobStateSuccess, StartTime: now.Add(-2 * time.Hour)},
			},
			expected: 1,
		},
		{
			name: "all failures",
			runs: []prow.JobRun{
				{State: prow.JobStateFailure, StartTime: now},
				{State: prow.JobStateFailure, StartTime: now.Add(-time.Hour)},
				{State: prow.JobStateFailure, StartTime: now.Add(-2 * time.Hour)},
			},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countConsecutiveFailures(tt.runs)
			if got != tt.expected {
				t.Errorf("countConsecutiveFailures() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestBuildJobHealth(t *testing.T) {
	now := time.Now()
	thresholds := config.JobHealthThresholds{
		PassRateWarningPercent:  80,
		PassRateCriticalPercent: 60,
		MinRuns:                 3,
	}

	t.Run("healthy job", func(t *testing.T) {
		runs := []prow.JobRun{
			{Job: "test-job", State: prow.JobStateSuccess, StartTime: now},
			{Job: "test-job", State: prow.JobStateSuccess, StartTime: now.Add(-time.Hour)},
			{Job: "test-job", State: prow.JobStateSuccess, StartTime: now.Add(-2 * time.Hour)},
			{Job: "test-job", State: prow.JobStateFailure, StartTime: now.Add(-3 * time.Hour)},
		}

		health := buildJobHealth("test-job", "nutanix", runs, thresholds)

		if health.TotalRuns != 4 {
			t.Errorf("expected 4 total runs, got %d", health.TotalRuns)
		}
		if health.Passed != 3 {
			t.Errorf("expected 3 passed, got %d", health.Passed)
		}
		if health.Failed != 1 {
			t.Errorf("expected 1 failed, got %d", health.Failed)
		}
		if health.Severity != SeverityWarning {
			t.Errorf("expected warning severity (75%% pass rate), got %s", health.Severity)
		}
		if health.ConsecutiveFailures != 0 {
			t.Errorf("expected 0 consecutive failures, got %d", health.ConsecutiveFailures)
		}
	})

	t.Run("insufficient data", func(t *testing.T) {
		runs := []prow.JobRun{
			{Job: "test-job", State: prow.JobStateSuccess, StartTime: now},
		}

		health := buildJobHealth("test-job", "", runs, thresholds)

		if health.Severity != SeverityNoData {
			t.Errorf("expected no-data severity, got %s", health.Severity)
		}
	})

	t.Run("aborted runs excluded from counts", func(t *testing.T) {
		runs := []prow.JobRun{
			{Job: "test-job", State: prow.JobStateSuccess, StartTime: now},
			{Job: "test-job", State: prow.JobStateAborted, StartTime: now.Add(-time.Hour)},
			{Job: "test-job", State: prow.JobStateSuccess, StartTime: now.Add(-2 * time.Hour)},
			{Job: "test-job", State: prow.JobStateSuccess, StartTime: now.Add(-3 * time.Hour)},
		}

		health := buildJobHealth("test-job", "", runs, thresholds)

		if health.TotalRuns != 3 {
			t.Errorf("expected 3 total runs (excluding aborted), got %d", health.TotalRuns)
		}
		if health.PassRate != 100.0 {
			t.Errorf("expected 100%% pass rate, got %.1f%%", health.PassRate)
		}
	})

	t.Run("no runs", func(t *testing.T) {
		health := buildJobHealth("test-job", "", nil, thresholds)

		if health.TotalRuns != 0 {
			t.Errorf("expected 0 total runs, got %d", health.TotalRuns)
		}
		if health.Severity != SeverityNoData {
			t.Errorf("expected no-data severity, got %s", health.Severity)
		}
	})
}

func TestBuildComponentSummaries(t *testing.T) {
	t.Run("groups by component", func(t *testing.T) {
		jobs := []JobHealth{
			{JobName: "job-1", Component: "nutanix", Passed: 8, TotalRuns: 10, PassRate: 80.0, Severity: SeverityHealthy},
			{JobName: "job-2", Component: "nutanix", Passed: 5, TotalRuns: 10, PassRate: 50.0, Severity: SeverityCritical},
			{JobName: "job-3", Component: "vsphere", Passed: 9, TotalRuns: 10, PassRate: 90.0, Severity: SeverityHealthy},
		}

		components := buildComponentSummaries(jobs)

		if len(components) != 2 {
			t.Fatalf("expected 2 components, got %d", len(components))
		}

		var nutanix ComponentHealth
		for _, c := range components {
			if c.Component == "nutanix" {
				nutanix = c
			}
		}
		if nutanix.Jobs != 2 {
			t.Errorf("expected 2 nutanix jobs, got %d", nutanix.Jobs)
		}
		if nutanix.Severity != SeverityCritical {
			t.Errorf("expected critical severity (worst of jobs), got %s", nutanix.Severity)
		}
	})

	t.Run("no components", func(t *testing.T) {
		jobs := []JobHealth{
			{JobName: "job-1", Severity: SeverityHealthy},
		}

		components := buildComponentSummaries(jobs)

		if components != nil {
			t.Errorf("expected nil components, got %v", components)
		}
	})
}

func TestJobHealthReportHasCritical(t *testing.T) {
	report := &JobHealthReport{
		Jobs: []JobHealth{
			{Severity: SeverityHealthy},
			{Severity: SeverityWarning},
		},
	}
	if report.HasCritical() {
		t.Error("expected no critical, got critical")
	}

	report.Jobs = append(report.Jobs, JobHealth{Severity: SeverityCritical})
	if !report.HasCritical() {
		t.Error("expected critical, got none")
	}
}

func TestJobHealthReportHasWarning(t *testing.T) {
	report := &JobHealthReport{
		Jobs: []JobHealth{
			{Severity: SeverityHealthy},
		},
	}
	if report.HasWarning() {
		t.Error("expected no warning, got warning")
	}

	report.Jobs[0].Severity = SeverityWarning
	if !report.HasWarning() {
		t.Error("expected warning, got none")
	}
}

func TestSeverityRank(t *testing.T) {
	if severityRank(SeverityHealthy) >= severityRank(SeverityNoData) {
		t.Error("healthy should rank below no-data")
	}
	if severityRank(SeverityNoData) >= severityRank(SeverityWarning) {
		t.Error("no-data should rank below warning")
	}
	if severityRank(SeverityWarning) >= severityRank(SeverityCritical) {
		t.Error("warning should rank below critical")
	}
}
