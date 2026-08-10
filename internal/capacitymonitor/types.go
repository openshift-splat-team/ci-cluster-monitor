package capacitymonitor

import "time"

type Severity string

const (
	SeverityNormal   Severity = "normal"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

type ResourceUsage struct {
	TotalHz  int64   `json:"total_hz,omitempty"`
	TotalMHz float64 `json:"total_mhz,omitempty"`
	UsedHz   int64   `json:"used_hz,omitempty"`
	UsedMHz  float64 `json:"used_mhz,omitempty"`
	FreeHz   int64   `json:"free_hz,omitempty"`
	FreeMHz  float64 `json:"free_mhz,omitempty"`

	TotalBytes int64   `json:"total_bytes,omitempty"`
	TotalGiB   float64 `json:"total_gib,omitempty"`
	UsedBytes  int64   `json:"used_bytes,omitempty"`
	UsedGiB    float64 `json:"used_gib,omitempty"`
	FreeBytes  int64   `json:"free_bytes,omitempty"`
	FreeGiB    float64 `json:"free_gib,omitempty"`

	UsedPercent float64  `json:"used_percent"`
	Severity    Severity `json:"severity"`
}

type ClusterCapacity struct {
	ClusterUUID string        `json:"cluster_uuid"`
	ClusterName string        `json:"cluster_name"`
	NodeCount   int           `json:"node_count"`
	VMCount     int64         `json:"vm_count"`
	CPU         ResourceUsage `json:"cpu"`
	Memory      ResourceUsage `json:"memory"`
}

type CapacityReport struct {
	Clusters  []ClusterCapacity `json:"clusters"`
	FetchedAt time.Time         `json:"fetched_at"`
}

func (r *CapacityReport) HasCritical() bool {
	for i := range r.Clusters {
		if r.Clusters[i].CPU.Severity == SeverityCritical ||
			r.Clusters[i].Memory.Severity == SeverityCritical {
			return true
		}
	}
	return false
}

func (r *CapacityReport) HasWarning() bool {
	for i := range r.Clusters {
		if r.Clusters[i].CPU.Severity == SeverityWarning ||
			r.Clusters[i].Memory.Severity == SeverityWarning {
			return true
		}
	}
	return false
}
