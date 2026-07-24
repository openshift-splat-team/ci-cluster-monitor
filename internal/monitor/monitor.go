package monitor

import (
	"context"
	"slices"
	"time"

	"k8s.io/klog/v2"

	"github.com/nutanix-cloud-native/ci-cluster-monitor/internal/config"
	"github.com/nutanix-cloud-native/ci-cluster-monitor/internal/github"
)

type Monitor struct {
	client *github.Client
	cfg    *config.Config
}

func New(client *github.Client, cfg *config.Config) *Monitor {
	return &Monitor{
		client: client,
		cfg:    cfg,
	}
}

func (m *Monitor) Run(ctx context.Context) ([]RepoReport, error) {
	now := time.Now()
	var reports []RepoReport

	for _, repo := range m.cfg.Repositories {
		klog.Infof("Fetching PRs for %s", repo.FullName())

		prs, err := m.client.ListOpenPRs(ctx, repo.Owner, repo.Name)
		if err != nil {
			klog.Errorf("Failed to fetch PRs for %s: %v", repo.FullName(), err)
			continue
		}

		report := RepoReport{
			Owner:     repo.Owner,
			Repo:      repo.Name,
			FetchedAt: now,
		}

		for i := range prs {
			if m.shouldExclude(&prs[i]) {
				continue
			}

			ciStatus := m.resolveCIStatus(ctx, repo, &prs[i])

			pr := &prs[i]
			metrics := PRMetrics{
				Repo:            repo.FullName(),
				Number:          pr.Number,
				Title:           pr.Title,
				Author:          pr.Author,
				CreatedAt:       pr.CreatedAt.Time,
				UpdatedAt:       pr.UpdatedAt.Time,
				Age:             now.Sub(pr.CreatedAt.Time),
				SinceLastUpdate: now.Sub(pr.UpdatedAt.Time),
				IsDraft:         pr.IsDraft,
				Labels:          pr.Labels,
				CIStatus:        ciStatus,
				HTMLURL:         pr.HTMLURL,
			}
			metrics.Severity = classifyPR(&metrics, m.cfg.Thresholds)
			report.PRs = append(report.PRs, metrics)
		}

		klog.Infof("Fetched PRs for %s: total=%d, included=%d", repo.FullName(), len(prs), len(report.PRs))
		reports = append(reports, report)
	}

	return reports, nil
}

func (m *Monitor) shouldExclude(pr *github.PullRequest) bool {
	if m.cfg.Filters.ExcludeDrafts && pr.IsDraft {
		return true
	}

	if slices.Contains(m.cfg.Filters.ExcludeAuthors, pr.Author) {
		return true
	}

	for _, label := range pr.Labels {
		if slices.Contains(m.cfg.Filters.ExcludeLabels, label) {
			return true
		}
	}

	return false
}

func (m *Monitor) resolveCIStatus(
	ctx context.Context,
	repo config.Repository,
	pr *github.PullRequest,
) string {
	if len(repo.Workflows) == 0 {
		return "no-workflows"
	}

	statuses, err := m.client.GetLatestWorkflowRunStatus(ctx, repo.Owner, repo.Name, pr, repo.Workflows)
	if err != nil {
		klog.Warningf("Failed to get CI status for %s PR #%d: %v", repo.FullName(), pr.Number, err)
		return "unknown"
	}

	hasFailure := false
	hasRunning := false
	allNoRuns := true

	for _, s := range statuses {
		if s.Status != "no-runs" {
			allNoRuns = false
		}
		switch {
		case s.Status == "completed" && s.Conclusion == "failure":
			hasFailure = true
		case s.Status == "completed" && s.Conclusion == "action_required":
			hasFailure = true
		case s.Status == "in_progress" || s.Status == "queued":
			hasRunning = true
		}
	}

	switch {
	case allNoRuns:
		return "no-runs"
	case hasFailure:
		return "failing"
	case hasRunning:
		return "running"
	default:
		return "passing"
	}
}

func classifyPR(pr *PRMetrics, thresholds config.Thresholds) Severity {
	ageDays := pr.Age.Hours() / 24
	staleDays := pr.SinceLastUpdate.Hours() / 24

	if staleDays >= float64(thresholds.StaleUpdateDays) && ageDays >= float64(thresholds.WarningAgeDays) {
		return SeverityStale
	}
	if ageDays >= float64(thresholds.CriticalAgeDays) {
		return SeverityCritical
	}
	if ageDays >= float64(thresholds.WarningAgeDays) {
		return SeverityWarning
	}
	return SeverityNormal
}
