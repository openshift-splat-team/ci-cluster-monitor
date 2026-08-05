package vmmonitor

import (
	"time"

	"github.com/nutanix-cloud-native/ci-cluster-monitor/internal/nutanix"
)

type OrphanStatus string

const (
	OrphanStatusTTLExceeded OrphanStatus = "ttl-exceeded"
)

type DeleteStatus string

const (
	DeleteStatusDeleted DeleteStatus = "deleted"
	DeleteStatusFailed  DeleteStatus = "failed"
)

type OrphanedVM struct {
	ExtID       string             `json:"ext_id"`
	Name        string             `json:"name"`
	CreateTime  time.Time          `json:"create_time"`
	Age         time.Duration      `json:"age"`
	PowerState  nutanix.PowerState `json:"power_state"`
	ClusterName string             `json:"cluster_name"`
	Status      OrphanStatus       `json:"status"`
}

type VMReport struct {
	TotalCIVMs    int            `json:"total_ci_vms"`
	OrphanedVMs   []OrphanedVM   `json:"orphaned_vms"`
	FetchedAt     time.Time      `json:"fetched_at"`
	TTLHours      int            `json:"ttl_hours"`
	CleanupReport *CleanupReport `json:"cleanup_report,omitempty"`
}

type DeleteResult struct {
	ExtID  string       `json:"ext_id"`
	Name   string       `json:"name"`
	Status DeleteStatus `json:"status"`
	Error  string       `json:"error,omitempty"`
}

type CleanupReport struct {
	Attempted int            `json:"attempted"`
	Succeeded int            `json:"succeeded"`
	Failed    int            `json:"failed"`
	Results   []DeleteResult `json:"results,omitempty"`
}
