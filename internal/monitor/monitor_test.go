// Copyright 2025.
// SPDX-License-Identifier: Apache-2.0

package monitor

import (
	"testing"
	"time"

	"github.com/nutanix-cloud-native/ci-cluster-monitor/internal/config"
)

func TestClassifyPR(t *testing.T) {
	thresholds := config.Thresholds{
		WarningAgeDays:  7,
		CriticalAgeDays: 14,
		StaleUpdateDays: 3,
	}

	tests := []struct {
		name     string
		age      time.Duration
		lastUpd  time.Duration
		expected Severity
	}{
		{
			name:     "normal - fresh PR",
			age:      2 * 24 * time.Hour,
			lastUpd:  1 * time.Hour,
			expected: SeverityNormal,
		},
		{
			name:     "warning - old but recently updated",
			age:      8 * 24 * time.Hour,
			lastUpd:  1 * time.Hour,
			expected: SeverityWarning,
		},
		{
			name:     "critical - very old",
			age:      15 * 24 * time.Hour,
			lastUpd:  1 * time.Hour,
			expected: SeverityCritical,
		},
		{
			name:     "stale - old and not updated",
			age:      10 * 24 * time.Hour,
			lastUpd:  5 * 24 * time.Hour,
			expected: SeverityStale,
		},
		{
			name:     "normal - young even if not updated",
			age:      2 * 24 * time.Hour,
			lastUpd:  5 * 24 * time.Hour,
			expected: SeverityNormal,
		},
		{
			name:     "exactly at warning boundary",
			age:      7 * 24 * time.Hour,
			lastUpd:  1 * time.Hour,
			expected: SeverityWarning,
		},
		{
			name:     "exactly at critical boundary",
			age:      14 * 24 * time.Hour,
			lastUpd:  1 * time.Hour,
			expected: SeverityCritical,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := PRMetrics{
				Age:             tt.age,
				SinceLastUpdate: tt.lastUpd,
			}
			got := classifyPR(pr, thresholds)
			if got != tt.expected {
				t.Errorf("classifyPR() = %v, want %v", got, tt.expected)
			}
		})
	}
}
