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
	if len(cfg.VMFilters.NamePrefixes) != 1 || cfg.VMFilters.NamePrefixes[0] != "ci-op-" {
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
	if len(cfg.VMFilters.NamePrefixes) != 1 || cfg.VMFilters.NamePrefixes[0] != "ci-op-" {
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
