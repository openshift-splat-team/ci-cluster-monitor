package vmmonitor

import (
	"testing"
	"time"

	"github.com/nutanix-cloud-native/ci-cluster-monitor/internal/nutanix"
)

func TestIsOrphaned(t *testing.T) {
	tests := []struct {
		name     string
		age      time.Duration
		ttl      time.Duration
		expected bool
	}{
		{
			name:     "VM younger than TTL is not orphaned",
			age:      2 * time.Hour,
			ttl:      8 * time.Hour,
			expected: false,
		},
		{
			name:     "VM exactly at TTL boundary is orphaned",
			age:      8 * time.Hour,
			ttl:      8 * time.Hour,
			expected: true,
		},
		{
			name:     "VM older than TTL is orphaned",
			age:      12 * time.Hour,
			ttl:      8 * time.Hour,
			expected: true,
		},
		{
			name:     "VM just under TTL is not orphaned",
			age:      7*time.Hour + 59*time.Minute,
			ttl:      8 * time.Hour,
			expected: false,
		},
		{
			name:     "zero age is not orphaned",
			age:      0,
			ttl:      8 * time.Hour,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isOrphaned(tt.age, tt.ttl)
			if got != tt.expected {
				t.Errorf("isOrphaned(%v, %v) = %v, want %v", tt.age, tt.ttl, got, tt.expected)
			}
		})
	}
}

func TestDeduplicateVMs(t *testing.T) {
	vms := []nutanix.VMInfo{
		{ExtID: "aaa", Name: "ci-op-test-master-0"},
		{ExtID: "bbb", Name: "ci-op-test-master-1"},
		{ExtID: "aaa", Name: "ci-op-test-master-0"},
		{ExtID: "ccc", Name: "ci-op-test-worker-abc"},
		{ExtID: "bbb", Name: "ci-op-test-master-1"},
	}

	result := deduplicateVMs(vms)

	if len(result) != 3 {
		t.Fatalf("expected 3 unique VMs, got %d", len(result))
	}

	expected := []string{"aaa", "bbb", "ccc"}
	for i, id := range expected {
		if result[i].ExtID != id {
			t.Errorf("result[%d].ExtID = %s, want %s", i, result[i].ExtID, id)
		}
	}
}

func TestDeduplicateVMsEmpty(t *testing.T) {
	result := deduplicateVMs(nil)
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d", len(result))
	}
}
