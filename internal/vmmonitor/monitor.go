package vmmonitor

import (
	"context"
	"fmt"
	"time"

	"k8s.io/klog/v2"

	"github.com/nutanix-cloud-native/ci-cluster-monitor/internal/config"
	"github.com/nutanix-cloud-native/ci-cluster-monitor/internal/nutanix"
)

type Monitor struct {
	client *nutanix.Client
	cfg    *config.NutanixConfig
}

func New(client *nutanix.Client, cfg *config.NutanixConfig) *Monitor {
	return &Monitor{
		client: client,
		cfg:    cfg,
	}
}

func (m *Monitor) Run(ctx context.Context) (*VMReport, error) {
	now := time.Now()
	ttl := time.Duration(m.cfg.Thresholds.OrphanTTLHours) * time.Hour

	var allVMs []nutanix.VMInfo
	for _, prefix := range m.cfg.VMFilters.NamePrefixes {
		klog.Infof("Listing VMs with prefix %q", prefix)
		vms, err := m.client.ListCIVMs(ctx, prefix)
		if err != nil {
			return nil, fmt.Errorf("listing VMs with prefix %q: %w", prefix, err)
		}
		allVMs = append(allVMs, vms...)
	}

	allVMs = deduplicateVMs(allVMs)

	report := &VMReport{
		TotalCIVMs: len(allVMs),
		FetchedAt:  now,
		TTLHours:   m.cfg.Thresholds.OrphanTTLHours,
	}

	for i := range allVMs {
		vm := &allVMs[i]
		age := now.Sub(vm.CreateTime)

		if isOrphaned(age, ttl) {
			report.OrphanedVMs = append(report.OrphanedVMs, OrphanedVM{
				ExtID:       vm.ExtID,
				Name:        vm.Name,
				CreateTime:  vm.CreateTime,
				Age:         age,
				PowerState:  vm.PowerState,
				ClusterName: vm.ClusterID,
				Status:      OrphanStatusTTLExceeded,
			})
		}
	}

	klog.Infof("Found %d CI VMs, %d orphaned (TTL=%dh)",
		len(allVMs), len(report.OrphanedVMs), m.cfg.Thresholds.OrphanTTLHours)
	return report, nil
}

func (m *Monitor) Cleanup(ctx context.Context, orphans []OrphanedVM, dryRun bool) *CleanupReport {
	report := &CleanupReport{}

	if dryRun {
		for i := range orphans {
			klog.Infof("[dry-run] Would delete VM %s (%s), age=%.1fh", orphans[i].Name, orphans[i].ExtID, orphans[i].Age.Hours())
		}
		return report
	}

	report.Attempted = len(orphans)
	for i := range orphans {
		vm := &orphans[i]
		klog.Infof("Deleting VM %s (%s), age=%.1fh", vm.Name, vm.ExtID, vm.Age.Hours())

		result := DeleteResult{ExtID: vm.ExtID, Name: vm.Name}
		if err := m.client.DeleteVM(ctx, vm.ExtID); err != nil {
			klog.Errorf("Failed to delete VM %s (%s): %v", vm.Name, vm.ExtID, err)
			result.Status = DeleteStatusFailed
			result.Error = err.Error()
			report.Failed++
		} else {
			klog.Infof("Deleted VM %s (%s)", vm.Name, vm.ExtID)
			result.Status = DeleteStatusDeleted
			report.Succeeded++
		}
		report.Results = append(report.Results, result)
	}

	klog.Infof("Cleanup complete: %d attempted, %d succeeded, %d failed",
		report.Attempted, report.Succeeded, report.Failed)
	return report
}

func isOrphaned(age, ttl time.Duration) bool {
	return age >= ttl
}

func deduplicateVMs(vms []nutanix.VMInfo) []nutanix.VMInfo {
	seen := make(map[string]struct{}, len(vms))
	result := make([]nutanix.VMInfo, 0, len(vms))
	for i := range vms {
		if _, ok := seen[vms[i].ExtID]; ok {
			continue
		}
		seen[vms[i].ExtID] = struct{}{}
		result = append(result, vms[i])
	}
	return result
}
