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
	ExtID      string
	Name       string
	CreateTime time.Time
	PowerState PowerState
	ClusterID  string
	Categories []string
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

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
