// Copyright 2025.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"k8s.io/klog/v2"

	"github.com/nutanix-cloud-native/ci-cluster-monitor/internal/alerts"
	"github.com/nutanix-cloud-native/ci-cluster-monitor/internal/config"
	ghclient "github.com/nutanix-cloud-native/ci-cluster-monitor/internal/github"
	"github.com/nutanix-cloud-native/ci-cluster-monitor/internal/monitor"
)

func main() {
	klog.InitFlags(nil)
	configPath := flag.String("config", "config/repos.yaml", "path to configuration file")
	format := flag.String("format", "table", "output format: table, json")
	notifySlack := flag.Bool("notify-slack", false, "send Slack notification for threshold violations")
	flag.Parse()
	defer klog.Flush()

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		klog.Errorf("GITHUB_TOKEN environment variable is required")
		os.Exit(1)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		klog.Errorf("Failed to load configuration: %v", err)
		os.Exit(1)
	}

	ctx := context.Background()
	client := ghclient.NewClient(ctx, token)
	mon := monitor.New(client, cfg)

	reports, err := mon.Run(ctx)
	if err != nil {
		klog.Errorf("Failed to run monitor: %v", err)
		os.Exit(1)
	}

	switch *format {
	case "json":
		outputJSON(reports)
	default:
		outputTable(reports)
	}

	if *notifySlack {
		webhookURL := os.Getenv("SLACK_WEBHOOK_URL")
		if webhookURL == "" {
			klog.Errorf("SLACK_WEBHOOK_URL environment variable is required when --notify-slack is set")
			os.Exit(1)
		}
		if err := alerts.SendSlackReport(webhookURL, reports, cfg.Thresholds); err != nil {
			klog.Errorf("Failed to send Slack notification: %v", err)
			os.Exit(1)
		}
	}
}

func outputTable(reports []monitor.RepoReport) {
	for _, report := range reports {
		if len(report.PRs) == 0 {
			fmt.Printf("\n%s: no open PRs\n", report.FullName())
			continue
		}

		fmt.Printf("\n%s (%d open PRs)\n", report.FullName(), len(report.PRs))

		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "PR\tTITLE\tAUTHOR\tAGE\tLAST UPDATE\tCI\tSEVERITY")
		fmt.Fprintln(w, "--\t-----\t------\t---\t-----------\t--\t--------")

		for _, pr := range report.PRs {
			ageDays := fmt.Sprintf("%.1fd", pr.Age.Hours()/24)
			lastUpdate := formatDuration(pr.SinceLastUpdate)
			title := pr.Title
			if len(title) > 50 {
				title = title[:47] + "..."
			}

			fmt.Fprintf(w, "#%d\t%s\t%s\t%s\t%s\t%s\t%s\n",
				pr.Number,
				title,
				pr.Author,
				ageDays,
				lastUpdate,
				pr.CIStatus,
				pr.Severity,
			)
		}
		w.Flush()
	}

	printSummary(reports)
}

func outputJSON(reports []monitor.RepoReport) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(reports); err != nil {
		klog.Errorf("Failed to encode JSON: %v", err)
		os.Exit(1)
	}
}

func formatDuration(d time.Duration) string {
	hours := d.Hours()
	switch {
	case hours < 1:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case hours < 24:
		return fmt.Sprintf("%dh ago", int(hours))
	default:
		return fmt.Sprintf("%dd ago", int(hours/24))
	}
}

func printSummary(reports []monitor.RepoReport) {
	var total, warning, critical, stale int
	for _, report := range reports {
		for _, pr := range report.PRs {
			total++
			switch pr.Severity {
			case monitor.SeverityWarning:
				warning++
			case monitor.SeverityCritical:
				critical++
			case monitor.SeverityStale:
				stale++
			}
		}
	}

	fmt.Printf("\nSummary: %d open PRs", total)
	var parts []string
	if critical > 0 {
		parts = append(parts, fmt.Sprintf("%d critical", critical))
	}
	if stale > 0 {
		parts = append(parts, fmt.Sprintf("%d stale", stale))
	}
	if warning > 0 {
		parts = append(parts, fmt.Sprintf("%d warning", warning))
	}
	if len(parts) > 0 {
		fmt.Printf(" (%s)", strings.Join(parts, ", "))
	}
	fmt.Println()
}
