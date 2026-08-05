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

	"github.com/nutanix-cloud-native/ci-cluster-monitor/internal/config"
	"github.com/nutanix-cloud-native/ci-cluster-monitor/internal/nutanix"
	"github.com/nutanix-cloud-native/ci-cluster-monitor/internal/vmmonitor"
)

func main() {
	klog.InitFlags(nil)
	configPath := flag.String("config", "config/nutanix.yaml", "path to configuration file")
	format := flag.String("format", "table", "output format: table, json")
	credentialsDir := flag.String("credentials-dir", "/tmp/secret", "path to directory containing credential files")
	cleanup := flag.Bool("cleanup", false, "enable deletion of orphaned VMs")
	dryRun := flag.Bool("dry-run", false, "preview deletions without executing (requires --cleanup)")
	flag.Parse()

	exitCode := run(*configPath, *format, *credentialsDir, *cleanup, *dryRun)
	klog.Flush()
	os.Exit(exitCode)
}

func run(configPath, format, credentialsDir string, cleanup, dryRun bool) int {
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

	mon := vmmonitor.New(client, cfg)
	report, err := mon.Run(ctx)
	if err != nil {
		klog.Errorf("Failed to run VM monitor: %v", err)
		return 1
	}

	if cleanup && len(report.OrphanedVMs) > 0 {
		report.CleanupReport = mon.Cleanup(ctx, report.OrphanedVMs, dryRun)
	}

	switch format {
	case "json":
		outputJSON(report)
	default:
		outputTable(report)
		if report.CleanupReport != nil && !dryRun {
			outputCleanupTable(report.CleanupReport)
		}
	}

	if report.CleanupReport != nil {
		if report.CleanupReport.Failed > 0 {
			return 2
		}
		if report.CleanupReport.Succeeded == len(report.OrphanedVMs) {
			return 0
		}
	}
	if len(report.OrphanedVMs) > 0 {
		klog.Infof("Orphaned VMs detected: %d", len(report.OrphanedVMs))
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

func outputTable(report *vmmonitor.VMReport) {
	fmt.Printf("\nNutanix CI VM Report (TTL: %dh)\n", report.TTLHours)
	fmt.Printf("Total CI VMs: %d, Orphaned: %d\n", report.TotalCIVMs, len(report.OrphanedVMs))

	if len(report.OrphanedVMs) == 0 {
		fmt.Println("No orphaned VMs found.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tEXT_ID\tAGE\tPOWER STATE\tCLUSTER\tSTATUS")
	fmt.Fprintln(w, "----\t------\t---\t-----------\t-------\t------")

	for i := range report.OrphanedVMs {
		vm := &report.OrphanedVMs[i]
		name := vm.Name
		if len(name) > 50 {
			name = name[:47] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%.1fh\t%s\t%s\t%s\n",
			name,
			vm.ExtID,
			vm.Age.Hours(),
			vm.PowerState,
			vm.ClusterName,
			vm.Status,
		)
	}
	w.Flush()
}

func outputCleanupTable(report *vmmonitor.CleanupReport) {
	fmt.Printf("\nCleanup Results: %d attempted, %d succeeded, %d failed\n",
		report.Attempted, report.Succeeded, report.Failed)

	if len(report.Results) == 0 {
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tEXT_ID\tRESULT\tERROR")
	fmt.Fprintln(w, "----\t------\t------\t-----")

	for i := range report.Results {
		r := &report.Results[i]
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.Name, r.ExtID, r.Status, r.Error)
	}
	w.Flush()
}

func outputJSON(report *vmmonitor.VMReport) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		klog.Errorf("Failed to encode JSON: %v", err)
		os.Exit(1)
	}
}
