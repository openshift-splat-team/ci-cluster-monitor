package nutanix

import (
	"context"
	"fmt"
	"time"

	"k8s.io/klog/v2"

	prismgoclient "github.com/nutanix-cloud-native/prism-go-client"
	"github.com/nutanix-cloud-native/prism-go-client/converged"
	convergedv4 "github.com/nutanix-cloud-native/prism-go-client/converged/v4"
)

type PowerState string

const (
	PowerStateOn      PowerState = "ON"
	PowerStateOff     PowerState = "OFF"
	PowerStateUnknown PowerState = "UNKNOWN"
)

type VMInfo struct {
	ExtID             string
	Name              string
	CreateTime        time.Time
	PowerState        PowerState
	ClusterID         string
	Categories        []string
	NumSockets        int
	NumCoresPerSocket int
	MemorySizeBytes   int64
}

func (v *VMInfo) NumVCPUs() int {
	return v.NumSockets * v.NumCoresPerSocket
}

type ClusterInfo struct {
	ExtID     string
	Name      string
	NodeCount int
	VMCount   int64
}

type HostInfo struct {
	ExtID            string
	Name             string
	ClusterID        string
	CPUCapacityHz    int64
	CPUFrequencyHz   int64
	MemorySizeBytes  int64
	NumberOfCPUCores int64
}

type Client struct {
	converged *convergedv4.Client
}

func NewClient(endpoint, port, username, password string, insecure bool) (*Client, error) {
	creds := prismgoclient.Credentials{
		URL:      fmt.Sprintf("%s:%s", endpoint, port),
		Endpoint: endpoint,
		Port:     port,
		Username: username,
		Password: password,
		Insecure: insecure,
	}
	c, err := convergedv4.NewClient(creds)
	if err != nil {
		return nil, fmt.Errorf("creating Nutanix client: %w", err)
	}
	return &Client{converged: c}, nil
}

func (c *Client) ListCIVMs(ctx context.Context, namePrefix string) ([]VMInfo, error) {
	filter := fmt.Sprintf("startswith(name, '%s')", namePrefix)
	klog.V(2).Infof("Listing VMs with filter: %s", filter)

	vms, err := c.converged.VMs.List(ctx, converged.WithFilter(filter))
	if err != nil {
		return nil, fmt.Errorf("listing VMs with prefix %q: %w", namePrefix, err)
	}

	results := make([]VMInfo, 0, len(vms))
	for i := range vms {
		vm := &vms[i]
		info := VMInfo{
			ExtID: derefString(vm.ExtId),
			Name:  derefString(vm.Name),
		}
		if vm.CreateTime != nil {
			info.CreateTime = *vm.CreateTime
		}
		if vm.PowerState != nil {
			info.PowerState = PowerState(vm.PowerState.GetName())
		}
		if vm.Cluster != nil {
			info.ClusterID = derefString(vm.Cluster.ExtId)
		}
		if vm.NumSockets != nil {
			info.NumSockets = *vm.NumSockets
		}
		if vm.NumCoresPerSocket != nil {
			info.NumCoresPerSocket = *vm.NumCoresPerSocket
		}
		if vm.MemorySizeBytes != nil {
			info.MemorySizeBytes = *vm.MemorySizeBytes
		}
		for _, cat := range vm.Categories {
			if cat.ExtId != nil {
				info.Categories = append(info.Categories, *cat.ExtId)
			}
		}
		results = append(results, info)
	}

	klog.V(2).Infof("Found %d VMs with prefix %q", len(results), namePrefix)
	return results, nil
}

func (c *Client) DeleteVM(ctx context.Context, extID string) error {
	klog.V(2).Infof("Deleting VM %s", extID)
	if err := c.converged.VMs.Delete(ctx, extID); err != nil {
		return fmt.Errorf("deleting VM %s: %w", extID, err)
	}
	return nil
}

func (c *Client) GetCluster(ctx context.Context, uuid string) (*ClusterInfo, error) {
	klog.V(2).Infof("Getting cluster %s", uuid)

	cluster, err := c.converged.Clusters.Get(ctx, uuid)
	if err != nil {
		return nil, fmt.Errorf("getting cluster %s: %w", uuid, err)
	}

	info := &ClusterInfo{
		ExtID: derefString(cluster.ExtId),
		Name:  derefString(cluster.Name),
	}
	if cluster.Nodes != nil && cluster.Nodes.NumberOfNodes != nil {
		info.NodeCount = *cluster.Nodes.NumberOfNodes
	}
	if cluster.VmCount != nil {
		info.VMCount = *cluster.VmCount
	}

	return info, nil
}

func (c *Client) ListClusterHosts(ctx context.Context, clusterUUID string) ([]HostInfo, error) {
	klog.V(2).Infof("Listing hosts for cluster %s", clusterUUID)

	hosts, err := c.converged.Clusters.ListClusterHosts(ctx, clusterUUID)
	if err != nil {
		return nil, fmt.Errorf("listing hosts for cluster %s: %w", clusterUUID, err)
	}

	results := make([]HostInfo, 0, len(hosts))
	for i := range hosts {
		h := &hosts[i]
		info := HostInfo{
			ExtID: derefString(h.ExtId),
			Name:  derefString(h.HostName),
		}
		if h.Cluster != nil {
			info.ClusterID = derefString(h.Cluster.Uuid)
		}
		if h.CpuCapacityHz != nil {
			info.CPUCapacityHz = *h.CpuCapacityHz
		}
		if h.CpuFrequencyHz != nil {
			info.CPUFrequencyHz = *h.CpuFrequencyHz
		}
		if h.MemorySizeBytes != nil {
			info.MemorySizeBytes = *h.MemorySizeBytes
		}
		if h.NumberOfCpuCores != nil {
			info.NumberOfCPUCores = *h.NumberOfCpuCores
		}
		results = append(results, info)
	}

	klog.V(2).Infof("Found %d hosts for cluster %s", len(results), clusterUUID)
	return results, nil
}

func (c *Client) ListCIVMsForCluster(ctx context.Context, clusterUUID, namePrefix string) ([]VMInfo, error) {
	filter := fmt.Sprintf("startswith(name, '%s') and cluster/extId eq '%s'", namePrefix, clusterUUID)
	klog.V(2).Infof("Listing VMs with filter: %s", filter)

	vms, err := c.converged.VMs.List(ctx, converged.WithFilter(filter))
	if err != nil {
		return nil, fmt.Errorf("listing VMs for cluster %s with prefix %q: %w", clusterUUID, namePrefix, err)
	}

	results := make([]VMInfo, 0, len(vms))
	for i := range vms {
		vm := &vms[i]
		info := VMInfo{
			ExtID: derefString(vm.ExtId),
			Name:  derefString(vm.Name),
		}
		if vm.CreateTime != nil {
			info.CreateTime = *vm.CreateTime
		}
		if vm.PowerState != nil {
			info.PowerState = PowerState(vm.PowerState.GetName())
		}
		if vm.Cluster != nil {
			info.ClusterID = derefString(vm.Cluster.ExtId)
		}
		if vm.NumSockets != nil {
			info.NumSockets = *vm.NumSockets
		}
		if vm.NumCoresPerSocket != nil {
			info.NumCoresPerSocket = *vm.NumCoresPerSocket
		}
		if vm.MemorySizeBytes != nil {
			info.MemorySizeBytes = *vm.MemorySizeBytes
		}
		results = append(results, info)
	}

	klog.V(2).Infof("Found %d VMs for cluster %s with prefix %q", len(results), clusterUUID, namePrefix)
	return results, nil
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
