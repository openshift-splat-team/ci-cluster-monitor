package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"

	"k8s.io/klog/v2"

	"github.com/nutanix-cloud-native/ci-cluster-monitor/internal/config"
	"github.com/nutanix-cloud-native/ci-cluster-monitor/internal/jobhealthmonitor"
	"github.com/nutanix-cloud-native/ci-cluster-monitor/internal/prow"
)

func main() {
	klog.InitFlags(nil)
	configPath := flag.String("config", "config/jobs.yaml", "path to configuration file")
	format := flag.String("format", "table", "output format: table, json")
	prowURL := flag.String("prow-url", "", "override Prow API URL from config")
	flag.Parse()

	exitCode := run(*configPath, *format, *prowURL)
	klog.Flush()
	os.Exit(exitCode)
}

func run(configPath, format, prowURL string) int {
	cfg, err := config.LoadJobHealth(configPath)
	if err != nil {
		klog.Errorf("Failed to load configuration: %v", err)
		return 1
	}

	if prowURL != "" {
		cfg.Prow.URL = prowURL
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client := prow.NewClient(cfg.Prow.URL)
	mon := jobhealthmonitor.New(client, cfg)

	report, err := mon.Run(ctx)
	if err != nil {
		klog.Errorf("Failed to run job health monitor: %v", err)
		return 1
	}

	switch format {
	case "json":
		if err := outputJSON(report); err != nil {
			klog.Errorf("Failed to encode JSON: %v", err)
			return 1
		}
	default:
		outputTable(report)
	}

	if report.HasCritical() {
		klog.Infof("Critical pass rate thresholds exceeded")
		return 3
	}
	if report.HasWarning() {
		klog.Infof("Warning pass rate thresholds exceeded")
		return 2
	}
	return 0
}

func outputTable(report *jobhealthmonitor.JobHealthReport) {
	fmt.Printf("\nCI Job Health Report (%d jobs, %d-day lookback)\n",
		len(report.Jobs), report.LookbackDays)

	if len(report.Jobs) == 0 {
		fmt.Println("No jobs to report.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "JOB\tCOMPONENT\tRUNS\tPASSED\tFAILED\tPASS RATE\tCONSEC FAIL\tSTATUS")
	fmt.Fprintln(w, "---\t---------\t----\t------\t------\t---------\t-----------\t------")

	for i := range report.Jobs {
		j := &report.Jobs[i]
		name := j.JobName
		if len(name) > 60 {
			name = name[:57] + "..."
		}
		component := j.Component
		if component == "" {
			component = "-"
		}

		fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%d\t%.1f%%\t%d\t%s\n",
			name,
			component,
			j.TotalRuns,
			j.Passed,
			j.Failed,
			j.PassRate,
			j.ConsecutiveFailures,
			j.Severity,
		)
	}
	w.Flush()

	if len(report.Components) > 0 {
		fmt.Println()
		cw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(cw, "COMPONENT\tJOBS\tPASS RATE\tSTATUS")
		fmt.Fprintln(cw, "---------\t----\t---------\t------")
		for i := range report.Components {
			c := &report.Components[i]
			fmt.Fprintf(cw, "%s\t%d\t%.1f%%\t%s\n",
				c.Component,
				c.Jobs,
				c.PassRate,
				c.Severity,
			)
		}
		cw.Flush()
	}

	printSummary(report)
}

func printSummary(report *jobhealthmonitor.JobHealthReport) {
	var healthy, warning, critical, noData int
	for i := range report.Jobs {
		switch report.Jobs[i].Severity {
		case jobhealthmonitor.SeverityCritical:
			critical++
		case jobhealthmonitor.SeverityWarning:
			warning++
		case jobhealthmonitor.SeverityNoData:
			noData++
		default:
			healthy++
		}
	}

	fmt.Printf("\nSummary: %d jobs", len(report.Jobs))
	var parts []string
	if critical > 0 {
		parts = append(parts, fmt.Sprintf("%d critical", critical))
	}
	if warning > 0 {
		parts = append(parts, fmt.Sprintf("%d warning", warning))
	}
	if noData > 0 {
		parts = append(parts, fmt.Sprintf("%d no-data", noData))
	}
	if healthy > 0 {
		parts = append(parts, fmt.Sprintf("%d healthy", healthy))
	}
	if len(parts) > 0 {
		fmt.Printf(" (%s)", strings.Join(parts, ", "))
	}
	fmt.Println()
}

func outputJSON(report *jobhealthmonitor.JobHealthReport) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}
