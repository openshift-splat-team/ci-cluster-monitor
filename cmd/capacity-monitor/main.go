package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"text/tabwriter"

	"k8s.io/klog/v2"

	"github.com/nutanix-cloud-native/ci-cluster-monitor/internal/capacitymonitor"
	"github.com/nutanix-cloud-native/ci-cluster-monitor/internal/config"
	"github.com/nutanix-cloud-native/ci-cluster-monitor/internal/nutanix"
)

func main() {
	klog.InitFlags(nil)
	configPath := flag.String("config", "config/nutanix.yaml", "path to configuration file")
	format := flag.String("format", "table", "output format: table, json")
	credentialsDir := flag.String("credentials-dir", "/tmp/secret", "path to directory containing credential files")
	endpoint := flag.String("endpoint", "", "override prism central endpoint from config")
	port := flag.String("port", "", "override prism central port from config")
	flag.Parse()

	exitCode := run(*configPath, *format, *credentialsDir, *endpoint, *port)
	klog.Flush()
	os.Exit(exitCode)
}

func run(configPath, format, credentialsDir, endpoint, port string) int {
	username, err := readCredentialFile(filepath.Join(credentialsDir, "nutanix-username"))
	if err != nil {
		klog.Errorf("Failed to read Nutanix username: %v", err)
		return 1
	}
	password, err := readCredentialFile(filepath.Join(credentialsDir, "nutanix-password"))
	if err != nil {
		klog.Errorf("Failed to read Nutanix password: %v", err)
		return 1
	}

	cfg, err := config.LoadNutanix(configPath)
	if err != nil {
		klog.Errorf("Failed to load configuration: %v", err)
		return 1
	}

	if endpoint != "" {
		cfg.PrismCentral.Endpoint = endpoint
	}
	if port != "" {
		cfg.PrismCentral.Port = port
	}

	if len(cfg.CapacityMonitoring.ClusterUUIDs) == 0 {
		klog.Errorf("No cluster UUIDs configured for capacity monitoring")
		return 1
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client, err := nutanix.NewClient(
		cfg.PrismCentral.Endpoint,
		cfg.PrismCentral.Port,
		username,
		password,
		cfg.PrismCentral.Insecure,
	)
	if err != nil {
		klog.Errorf("Failed to create Nutanix client: %v", err)
		return 1
	}

	mon := capacitymonitor.New(client, cfg)
	report, err := mon.Run(ctx)
	if err != nil {
		klog.Errorf("Failed to run capacity monitor: %v", err)
		return 1
	}

	switch format {
	case "json":
		outputJSON(report)
	default:
		outputTable(report)
	}

	if report.HasCritical() {
		klog.Infof("Critical capacity thresholds exceeded")
		return 2
	}
	if report.HasWarning() {
		klog.Infof("Warning capacity thresholds exceeded")
		return 1
	}
	return 0
}

func readCredentialFile(path string) (string, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- path is from CLI flag, not user input
	if err != nil {
		return "", fmt.Errorf("reading credential file %s: %w", path, err)
	}
	return strings.TrimSpace(string(data)), nil
}

func outputTable(report *capacitymonitor.CapacityReport) {
	fmt.Printf("\nNutanix Cluster Capacity Report (%d clusters)\n", len(report.Clusters))

	if len(report.Clusters) == 0 {
		fmt.Println("No clusters to report.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "CLUSTER\tNODES\tVMs\tCPU USED\tCPU FREE\tCPU %\tCPU STATUS\tMEM USED\tMEM FREE\tMEM %\tMEM STATUS")
	fmt.Fprintln(w, "-------\t-----\t---\t--------\t--------\t-----\t----------\t--------\t--------\t-----\t----------")

	for i := range report.Clusters {
		c := &report.Clusters[i]
		name := c.ClusterName
		if name == "" {
			name = c.ClusterUUID
		}
		if len(name) > 30 {
			name = name[:27] + "..."
		}

		fmt.Fprintf(w, "%s\t%d\t%d\t%.0f MHz\t%.0f MHz\t%.1f%%\t%s\t%.1f GiB\t%.1f GiB\t%.1f%%\t%s\n",
			name,
			c.NodeCount,
			c.VMCount,
			c.CPU.UsedMHz,
			c.CPU.FreeMHz,
			c.CPU.UsedPercent,
			c.CPU.Severity,
			c.Memory.UsedGiB,
			c.Memory.FreeGiB,
			c.Memory.UsedPercent,
			c.Memory.Severity,
		)
	}
	w.Flush()

	printSummary(report)
}

func printSummary(report *capacitymonitor.CapacityReport) {
	var normal, warning, critical int
	for i := range report.Clusters {
		c := &report.Clusters[i]
		worst := c.CPU.Severity
		if severityRank(c.Memory.Severity) > severityRank(worst) {
			worst = c.Memory.Severity
		}
		switch worst {
		case capacitymonitor.SeverityCritical:
			critical++
		case capacitymonitor.SeverityWarning:
			warning++
		default:
			normal++
		}
	}

	fmt.Printf("\nSummary: %d clusters", len(report.Clusters))
	var parts []string
	if critical > 0 {
		parts = append(parts, fmt.Sprintf("%d critical", critical))
	}
	if warning > 0 {
		parts = append(parts, fmt.Sprintf("%d warning", warning))
	}
	if normal > 0 {
		parts = append(parts, fmt.Sprintf("%d normal", normal))
	}
	if len(parts) > 0 {
		fmt.Printf(" (%s)", strings.Join(parts, ", "))
	}
	fmt.Println()
}

func severityRank(s capacitymonitor.Severity) int {
	switch s {
	case capacitymonitor.SeverityCritical:
		return 2
	case capacitymonitor.SeverityWarning:
		return 1
	default:
		return 0
	}
}

func outputJSON(report *capacitymonitor.CapacityReport) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		klog.Errorf("Failed to encode JSON: %v", err)
		os.Exit(1)
	}
}
