package capacitymonitor

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

func (m *Monitor) Run(ctx context.Context) (*CapacityReport, error) {
	report := &CapacityReport{
		FetchedAt: time.Now(),
	}

	for _, clusterUUID := range m.cfg.CapacityMonitoring.ClusterUUIDs {
		klog.Infof("Checking capacity for cluster %s", clusterUUID)

		cc, err := m.collectClusterCapacity(ctx, clusterUUID)
		if err != nil {
			return nil, fmt.Errorf("collecting capacity for cluster %s: %w", clusterUUID, err)
		}

		report.Clusters = append(report.Clusters, *cc)
	}

	return report, nil
}

func (m *Monitor) collectClusterCapacity(ctx context.Context, clusterUUID string) (*ClusterCapacity, error) {
	cluster, err := m.client.GetCluster(ctx, clusterUUID)
	if err != nil {
		return nil, fmt.Errorf("getting cluster info: %w", err)
	}

	hosts, err := m.client.ListClusterHosts(ctx, clusterUUID)
	if err != nil {
		return nil, fmt.Errorf("listing cluster hosts: %w", err)
	}

	var totalCPUHz, totalMemoryBytes int64
	for i := range hosts {
		totalCPUHz += hosts[i].CPUCapacityHz
		totalMemoryBytes += hosts[i].MemorySizeBytes
	}

	var ciVMs []nutanix.VMInfo
	for _, prefix := range m.cfg.VMFilters.NamePrefixes {
		vms, err := m.client.ListCIVMsForCluster(ctx, clusterUUID, prefix)
		if err != nil {
			return nil, fmt.Errorf("listing CI VMs with prefix %q: %w", prefix, err)
		}
		ciVMs = append(ciVMs, vms...)
	}

	var usedCPUHz, usedMemoryBytes int64
	for i := range ciVMs {
		if ciVMs[i].PowerState == nutanix.PowerStateOn {
			usedCPUHz += estimateVMCPUHz(hosts)
			usedMemoryBytes += estimateVMMemoryBytes(hosts)
		}
	}

	thresholds := m.cfg.CapacityMonitoring.CapacityThresholds

	cc := &ClusterCapacity{
		ClusterUUID: clusterUUID,
		ClusterName: cluster.Name,
		NodeCount:   cluster.NodeCount,
		VMCount:     cluster.VMCount,
		CPU:         buildCPUUsage(totalCPUHz, usedCPUHz, thresholds),
		Memory:      buildMemoryUsage(totalMemoryBytes, usedMemoryBytes, thresholds),
	}

	klog.Infof("Cluster %s (%s): nodes=%d, vms=%d, cpu=%.1f%%, memory=%.1f%%",
		cc.ClusterName, cc.ClusterUUID, cc.NodeCount, cc.VMCount,
		cc.CPU.UsedPercent, cc.Memory.UsedPercent)

	return cc, nil
}

func buildCPUUsage(totalHz, usedHz int64, thresholds config.CapacityThresholds) ResourceUsage {
	usedPercent := calcPercent(usedHz, totalHz)
	freeHz := totalHz - usedHz

	return ResourceUsage{
		TotalHz:     totalHz,
		TotalMHz:    hzToMHz(totalHz),
		UsedHz:      usedHz,
		UsedMHz:     hzToMHz(usedHz),
		FreeHz:      freeHz,
		FreeMHz:     hzToMHz(freeHz),
		UsedPercent: usedPercent,
		Severity:    classifySeverity(usedPercent, thresholds.CPUWarningPercent, thresholds.CPUCriticalPercent),
	}
}

func buildMemoryUsage(totalBytes, usedBytes int64, thresholds config.CapacityThresholds) ResourceUsage {
	usedPercent := calcPercent(usedBytes, totalBytes)
	freeBytes := totalBytes - usedBytes

	return ResourceUsage{
		TotalBytes:  totalBytes,
		TotalGiB:    bytesToGiB(totalBytes),
		UsedBytes:   usedBytes,
		UsedGiB:     bytesToGiB(usedBytes),
		FreeBytes:   freeBytes,
		FreeGiB:     bytesToGiB(freeBytes),
		UsedPercent: usedPercent,
		Severity:    classifySeverity(usedPercent, thresholds.MemoryWarningPercent, thresholds.MemoryCriticalPercent),
	}
}

func classifySeverity(usedPercent float64, warningThreshold, criticalThreshold int) Severity {
	switch {
	case usedPercent >= float64(criticalThreshold):
		return SeverityCritical
	case usedPercent >= float64(warningThreshold):
		return SeverityWarning
	default:
		return SeverityNormal
	}
}

func calcPercent(used, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(used) / float64(total) * 100
}

func hzToMHz(hz int64) float64 {
	return float64(hz) / 1_000_000
}

func bytesToGiB(b int64) float64 {
	return float64(b) / (1024 * 1024 * 1024)
}

func estimateVMCPUHz(hosts []nutanix.HostInfo) int64 {
	if len(hosts) == 0 {
		return 0
	}
	var totalCores int64
	var totalHz int64
	for i := range hosts {
		totalCores += hosts[i].NumberOfCPUCores
		totalHz += hosts[i].CPUCapacityHz
	}
	if totalCores == 0 {
		return 0
	}
	return totalHz / totalCores
}

func estimateVMMemoryBytes(hosts []nutanix.HostInfo) int64 {
	if len(hosts) == 0 {
		return 0
	}
	var totalCores int64
	var totalMem int64
	for i := range hosts {
		totalCores += hosts[i].NumberOfCPUCores
		totalMem += hosts[i].MemorySizeBytes
	}
	if totalCores == 0 {
		return 0
	}
	return totalMem / totalCores
}
