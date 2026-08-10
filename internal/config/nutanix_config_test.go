package config

import (
	"testing"
)

func TestLoadNutanix(t *testing.T) {
	content := `
prism_central:
  endpoint: "prism-central.example.com"
  port: "9440"
  insecure: true

vm_filters:
  name_prefixes:
    - "ci-op-"

thresholds:
  orphan_ttl_hours: 6
`
	path := writeTestFile(t, content)

	cfg, err := LoadNutanix(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.PrismCentral.Endpoint != "prism-central.example.com" {
		t.Errorf("expected endpoint prism-central.example.com, got %s", cfg.PrismCentral.Endpoint)
	}
	if cfg.PrismCentral.Port != "9440" {
		t.Errorf("expected port 9440, got %s", cfg.PrismCentral.Port)
	}
	if !cfg.PrismCentral.Insecure {
		t.Error("expected insecure true")
	}
	if len(cfg.VMFilters.NamePrefixes) != 1 || cfg.VMFilters.NamePrefixes[0] != DefaultNamePrefix {
		t.Errorf("expected name_prefixes [ci-op-], got %v", cfg.VMFilters.NamePrefixes)
	}
	if cfg.Thresholds.OrphanTTLHours != 6 {
		t.Errorf("expected orphan_ttl_hours 6, got %d", cfg.Thresholds.OrphanTTLHours)
	}
}

func TestLoadNutanixAppliesDefaults(t *testing.T) {
	content := `
prism_central:
  endpoint: "pc.example.com"
`
	path := writeTestFile(t, content)

	cfg, err := LoadNutanix(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.PrismCentral.Port != "9440" {
		t.Errorf("expected default port 9440, got %s", cfg.PrismCentral.Port)
	}
	if len(cfg.VMFilters.NamePrefixes) != 1 || cfg.VMFilters.NamePrefixes[0] != DefaultNamePrefix {
		t.Errorf("expected default name_prefixes [ci-op-], got %v", cfg.VMFilters.NamePrefixes)
	}
	if cfg.Thresholds.OrphanTTLHours != 8 {
		t.Errorf("expected default orphan_ttl_hours 8, got %d", cfg.Thresholds.OrphanTTLHours)
	}
}

func TestLoadNutanixValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name: "missing endpoint",
			content: `
prism_central:
  port: "9440"
`,
		},
		{
			name: "negative ttl",
			content: `
prism_central:
  endpoint: "pc.example.com"
thresholds:
  orphan_ttl_hours: -1
`,
		},
		{
			name: "cpu warning >= critical",
			content: `
prism_central:
  endpoint: "pc.example.com"
capacity_monitoring:
  thresholds:
    cpu_warning_percent: 90
    cpu_critical_percent: 80
`,
		},
		{
			name: "memory warning >= critical",
			content: `
prism_central:
  endpoint: "pc.example.com"
capacity_monitoring:
  thresholds:
    memory_warning_percent: 90
    memory_critical_percent: 85
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTestFile(t, tt.content)
			_, err := LoadNutanix(path)
			if err == nil {
				t.Error("expected validation error, got nil")
			}
		})
	}
}

func TestLoadNutanixCapacityDefaults(t *testing.T) {
	content := `
prism_central:
  endpoint: "pc.example.com"
`
	path := writeTestFile(t, content)

	cfg, err := LoadNutanix(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.CapacityMonitoring.CapacityThresholds.CPUWarningPercent != 70 {
		t.Errorf("expected default cpu_warning_percent 70, got %d",
			cfg.CapacityMonitoring.CapacityThresholds.CPUWarningPercent)
	}
	if cfg.CapacityMonitoring.CapacityThresholds.CPUCriticalPercent != 85 {
		t.Errorf("expected default cpu_critical_percent 85, got %d",
			cfg.CapacityMonitoring.CapacityThresholds.CPUCriticalPercent)
	}
	if cfg.CapacityMonitoring.CapacityThresholds.MemoryWarningPercent != 70 {
		t.Errorf("expected default memory_warning_percent 70, got %d",
			cfg.CapacityMonitoring.CapacityThresholds.MemoryWarningPercent)
	}
	if cfg.CapacityMonitoring.CapacityThresholds.MemoryCriticalPercent != 85 {
		t.Errorf("expected default memory_critical_percent 85, got %d",
			cfg.CapacityMonitoring.CapacityThresholds.MemoryCriticalPercent)
	}
}

func TestLoadNutanixWithCapacityConfig(t *testing.T) {
	content := `
prism_central:
  endpoint: "pc.example.com"
capacity_monitoring:
  cluster_uuids:
    - "uuid-1"
    - "uuid-2"
  thresholds:
    cpu_warning_percent: 60
    cpu_critical_percent: 80
    memory_warning_percent: 65
    memory_critical_percent: 90
`
	path := writeTestFile(t, content)

	cfg, err := LoadNutanix(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.CapacityMonitoring.ClusterUUIDs) != 2 {
		t.Fatalf("expected 2 cluster UUIDs, got %d", len(cfg.CapacityMonitoring.ClusterUUIDs))
	}
	if cfg.CapacityMonitoring.ClusterUUIDs[0] != "uuid-1" {
		t.Errorf("expected uuid-1, got %s", cfg.CapacityMonitoring.ClusterUUIDs[0])
	}
	if cfg.CapacityMonitoring.CapacityThresholds.CPUWarningPercent != 60 {
		t.Errorf("expected cpu_warning_percent 60, got %d",
			cfg.CapacityMonitoring.CapacityThresholds.CPUWarningPercent)
	}
	if cfg.CapacityMonitoring.CapacityThresholds.MemoryCriticalPercent != 90 {
		t.Errorf("expected memory_critical_percent 90, got %d",
			cfg.CapacityMonitoring.CapacityThresholds.MemoryCriticalPercent)
	}
}
