package capacitymonitor

import (
	"testing"

	"github.com/nutanix-cloud-native/ci-cluster-monitor/internal/config"
	"github.com/nutanix-cloud-native/ci-cluster-monitor/internal/nutanix"
)

func TestClassifySeverity(t *testing.T) {
	tests := []struct {
		name     string
		used     float64
		warning  int
		critical int
		expected Severity
	}{
		{
			name:     "normal - well below warning",
			used:     30.0,
			warning:  70,
			critical: 85,
			expected: SeverityNormal,
		},
		{
			name:     "normal - just below warning",
			used:     69.9,
			warning:  70,
			critical: 85,
			expected: SeverityNormal,
		},
		{
			name:     "warning - exactly at warning threshold",
			used:     70.0,
			warning:  70,
			critical: 85,
			expected: SeverityWarning,
		},
		{
			name:     "warning - between warning and critical",
			used:     80.0,
			warning:  70,
			critical: 85,
			expected: SeverityWarning,
		},
		{
			name:     "critical - exactly at critical threshold",
			used:     85.0,
			warning:  70,
			critical: 85,
			expected: SeverityCritical,
		},
		{
			name:     "critical - above critical threshold",
			used:     95.0,
			warning:  70,
			critical: 85,
			expected: SeverityCritical,
		},
		{
			name:     "normal - zero usage",
			used:     0.0,
			warning:  70,
			critical: 85,
			expected: SeverityNormal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifySeverity(tt.used, tt.warning, tt.critical)
			if got != tt.expected {
				t.Errorf("classifySeverity(%.1f, %d, %d) = %v, want %v",
					tt.used, tt.warning, tt.critical, got, tt.expected)
			}
		})
	}
}

func TestCalcPercent(t *testing.T) {
	tests := []struct {
		name     string
		used     int64
		total    int64
		expected float64
	}{
		{name: "50%", used: 50, total: 100, expected: 50.0},
		{name: "0%", used: 0, total: 100, expected: 0.0},
		{name: "100%", used: 100, total: 100, expected: 100.0},
		{name: "zero total", used: 50, total: 0, expected: 0.0},
		{name: "negative total", used: 50, total: -1, expected: 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calcPercent(tt.used, tt.total)
			if got != tt.expected {
				t.Errorf("calcPercent(%d, %d) = %f, want %f", tt.used, tt.total, got, tt.expected)
			}
		})
	}
}

func TestHzToMHz(t *testing.T) {
	got := hzToMHz(2_400_000_000)
	expected := 2400.0
	if got != expected {
		t.Errorf("hzToMHz(2400000000) = %f, want %f", got, expected)
	}
}

func TestBytesToGiB(t *testing.T) {
	got := bytesToGiB(1024 * 1024 * 1024)
	expected := 1.0
	if got != expected {
		t.Errorf("bytesToGiB(1GiB) = %f, want %f", got, expected)
	}
}

func TestEstimateVMCPUHz(t *testing.T) {
	tests := []struct {
		name     string
		hosts    []nutanix.HostInfo
		expected int64
	}{
		{
			name:     "empty hosts",
			hosts:    nil,
			expected: 0,
		},
		{
			name: "single host with 4 cores",
			hosts: []nutanix.HostInfo{
				{CPUCapacityHz: 8_000_000_000, NumberOfCPUCores: 4},
			},
			expected: 2_000_000_000,
		},
		{
			name: "two hosts",
			hosts: []nutanix.HostInfo{
				{CPUCapacityHz: 8_000_000_000, NumberOfCPUCores: 4},
				{CPUCapacityHz: 16_000_000_000, NumberOfCPUCores: 8},
			},
			expected: 2_000_000_000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := estimateVMCPUHz(tt.hosts)
			if got != tt.expected {
				t.Errorf("estimateVMCPUHz() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestBuildCPUUsage(t *testing.T) {
	thresholds := config.CapacityThresholds{
		CPUWarningPercent:  70,
		CPUCriticalPercent: 85,
	}

	usage := buildCPUUsage(1_000_000_000, 800_000_000, thresholds)

	if usage.UsedPercent != 80.0 {
		t.Errorf("expected 80.0%% used, got %.1f%%", usage.UsedPercent)
	}
	if usage.Severity != SeverityWarning {
		t.Errorf("expected warning severity, got %s", usage.Severity)
	}
	if usage.FreeHz != 200_000_000 {
		t.Errorf("expected 200MHz free, got %d", usage.FreeHz)
	}
}

func TestBuildMemoryUsage(t *testing.T) {
	thresholds := config.CapacityThresholds{
		MemoryWarningPercent:  70,
		MemoryCriticalPercent: 85,
	}

	totalBytes := int64(100 * 1024 * 1024 * 1024) // 100 GiB
	usedBytes := int64(90 * 1024 * 1024 * 1024)   // 90 GiB

	usage := buildMemoryUsage(totalBytes, usedBytes, thresholds)

	if usage.UsedPercent != 90.0 {
		t.Errorf("expected 90.0%% used, got %.1f%%", usage.UsedPercent)
	}
	if usage.Severity != SeverityCritical {
		t.Errorf("expected critical severity, got %s", usage.Severity)
	}
}

func TestCapacityReportHasCritical(t *testing.T) {
	report := &CapacityReport{
		Clusters: []ClusterCapacity{
			{CPU: ResourceUsage{Severity: SeverityNormal}, Memory: ResourceUsage{Severity: SeverityWarning}},
		},
	}
	if report.HasCritical() {
		t.Error("expected no critical, got critical")
	}

	report.Clusters = append(report.Clusters, ClusterCapacity{
		CPU: ResourceUsage{Severity: SeverityCritical}, Memory: ResourceUsage{Severity: SeverityNormal},
	})
	if !report.HasCritical() {
		t.Error("expected critical, got none")
	}
}

func TestCapacityReportHasWarning(t *testing.T) {
	report := &CapacityReport{
		Clusters: []ClusterCapacity{
			{CPU: ResourceUsage{Severity: SeverityNormal}, Memory: ResourceUsage{Severity: SeverityNormal}},
		},
	}
	if report.HasWarning() {
		t.Error("expected no warning, got warning")
	}

	report.Clusters[0].Memory.Severity = SeverityWarning
	if !report.HasWarning() {
		t.Error("expected warning, got none")
	}
}
