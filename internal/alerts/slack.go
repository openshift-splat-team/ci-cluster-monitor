// Copyright 2025.
// SPDX-License-Identifier: Apache-2.0

package alerts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"k8s.io/klog/v2"

	"github.com/nutanix-cloud-native/ci-cluster-monitor/internal/config"
	"github.com/nutanix-cloud-native/ci-cluster-monitor/internal/monitor"
)

type slackMessage struct {
	Blocks []slackBlock `json:"blocks"`
}

type slackBlock struct {
	Type string     `json:"type"`
	Text *slackText `json:"text,omitempty"`
}

type slackText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func SendSlackReport(
	webhookURL string,
	reports []monitor.RepoReport,
	thresholds config.Thresholds,
) error {
	flaggedPRs := collectFlaggedPRs(reports)
	if len(flaggedPRs) == 0 {
		klog.Info("No PRs exceed thresholds, skipping Slack notification")
		return nil
	}

	msg := buildSlackMessage(flaggedPRs, thresholds)
	return postToSlack(webhookURL, msg)
}

func collectFlaggedPRs(reports []monitor.RepoReport) []*monitor.PRMetrics {
	var flagged []*monitor.PRMetrics
	for i := range reports {
		for j := range reports[i].PRs {
			if reports[i].PRs[j].Severity != monitor.SeverityNormal {
				flagged = append(flagged, &reports[i].PRs[j])
			}
		}
	}
	return flagged
}

func buildSlackMessage(prs []*monitor.PRMetrics, thresholds config.Thresholds) slackMessage {
	var lines []string
	lines = append(lines, fmt.Sprintf(
		"*PR Age Monitor Report* (%s)\nThresholds: warning >%dd, critical >%dd, stale >%dd since update",
		time.Now().UTC().Format("2006-01-02 15:04 UTC"),
		thresholds.WarningAgeDays,
		thresholds.CriticalAgeDays,
		thresholds.StaleUpdateDays,
	))

	currentRepo := ""
	for _, pr := range prs {
		if pr.Repo != currentRepo {
			currentRepo = pr.Repo
			lines = append(lines, fmt.Sprintf("\n*%s*", currentRepo))
		}

		sIcon := severityIcon(pr.Severity)
		ageDays := int(pr.Age.Hours() / 24)
		ciIcon := ciStatusIcon(pr.CIStatus)

		lines = append(lines, fmt.Sprintf(
			"%s <%s|#%d> %s (%s, %dd old, CI: %s)",
			sIcon,
			pr.HTMLURL,
			pr.Number,
			truncate(pr.Title, 60),
			pr.Author,
			ageDays,
			ciIcon,
		))
	}

	return slackMessage{
		Blocks: []slackBlock{
			{
				Type: "section",
				Text: &slackText{
					Type: "mrkdwn",
					Text: strings.Join(lines, "\n"),
				},
			},
		},
	}
}

func severityIcon(s monitor.Severity) string {
	switch s {
	case monitor.SeverityCritical:
		return ":red_circle:"
	case monitor.SeverityStale:
		return ":large_orange_circle:"
	case monitor.SeverityWarning:
		return ":large_yellow_circle:"
	default:
		return ":white_circle:"
	}
}

func ciStatusIcon(status string) string {
	switch status {
	case "passing":
		return ":white_check_mark:"
	case "failing":
		return ":x:"
	case "running":
		return ":hourglass_flowing_sand:"
	default:
		return ":grey_question:"
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func postToSlack(webhookURL string, msg slackMessage) error {
	body, _ := json.Marshal(msg)

	//nolint:gosec // webhook URL is from trusted config
	resp, err := http.Post(webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("posting to Slack: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack returned status %d", resp.StatusCode)
	}

	klog.Infof("Slack notification sent, flagged_prs=%d", len(msg.Blocks))
	return nil
}
