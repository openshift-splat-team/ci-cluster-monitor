package jobhealthmonitor

import (
	"context"
	"fmt"
	"sort"
	"time"

	"k8s.io/klog/v2"

	"github.com/nutanix-cloud-native/ci-cluster-monitor/internal/config"
	"github.com/nutanix-cloud-native/ci-cluster-monitor/internal/prow"
)

type Monitor struct {
	client *prow.Client
	cfg    *config.JobHealthConfig
}

func New(client *prow.Client, cfg *config.JobHealthConfig) *Monitor {
	return &Monitor{
		client: client,
		cfg:    cfg,
	}
}

func (m *Monitor) Run(ctx context.Context) (*JobHealthReport, error) {
	now := time.Now()
	since := now.AddDate(0, 0, -m.cfg.Thresholds.LookbackDays)

	jobNames := make(map[string]bool, len(m.cfg.Jobs))
	jobComponents := make(map[string]string, len(m.cfg.Jobs))
	for _, j := range m.cfg.Jobs {
		jobNames[j.Name] = true
		if j.Component != "" {
			jobComponents[j.Name] = j.Component
		}
	}

	klog.Infof("Fetching job runs since %s (%d day lookback)", since.Format(time.RFC3339), m.cfg.Thresholds.LookbackDays)

	runs, err := m.client.ListRecentRuns(ctx, jobNames, since)
	if err != nil {
		return nil, fmt.Errorf("listing recent job runs: %w", err)
	}

	runsByJob := make(map[string][]prow.JobRun)
	for i := range runs {
		runsByJob[runs[i].Job] = append(runsByJob[runs[i].Job], runs[i])
	}

	report := &JobHealthReport{
		FetchedAt:    now,
		LookbackDays: m.cfg.Thresholds.LookbackDays,
	}

	for _, jobCfg := range m.cfg.Jobs {
		jobRuns := runsByJob[jobCfg.Name]
		health := buildJobHealth(jobCfg.Name, jobCfg.Component, jobRuns, m.cfg.Thresholds)

		klog.Infof("Job %s: runs=%d, passed=%d, failed=%d, pass_rate=%.1f%%, severity=%s",
			health.JobName, health.TotalRuns, health.Passed, health.Failed,
			health.PassRate, health.Severity)

		report.Jobs = append(report.Jobs, health)
	}

	report.Components = buildComponentSummaries(report.Jobs)

	return report, nil
}

func buildJobHealth(name, component string, runs []prow.JobRun, thresholds config.JobHealthThresholds) JobHealth {
	health := JobHealth{
		JobName:   name,
		Component: component,
	}

	sort.Slice(runs, func(i, j int) bool {
		return runs[i].StartTime.After(runs[j].StartTime)
	})

	var passed, failed int
	for i := range runs {
		if !runs[i].IsCompleted() {
			continue
		}
		if runs[i].State == prow.JobStateAborted {
			continue
		}
		if runs[i].State == prow.JobStateSuccess {
			passed++
		} else {
			failed++
		}
	}

	health.TotalRuns = passed + failed
	health.Passed = passed
	health.Failed = failed
	health.PassRate = calcPassRate(passed, health.TotalRuns)
	health.ConsecutiveFailures = countConsecutiveFailures(runs)

	if len(runs) > 0 {
		health.LastRunTime = runs[0].StartTime
	}

	if health.TotalRuns < thresholds.MinRuns {
		health.Severity = SeverityNoData
	} else {
		health.Severity = classifyPassRate(
			health.PassRate, thresholds.PassRateWarningPercent, thresholds.PassRateCriticalPercent)
	}

	return health
}

func buildComponentSummaries(jobs []JobHealth) []ComponentHealth {
	componentJobs := make(map[string][]JobHealth)
	for i := range jobs {
		if jobs[i].Component == "" {
			continue
		}
		componentJobs[jobs[i].Component] = append(componentJobs[jobs[i].Component], jobs[i])
	}

	if len(componentJobs) == 0 {
		return nil
	}

	var components []ComponentHealth
	for name, cJobs := range componentJobs {
		var totalPassed, totalRuns int
		worst := SeverityHealthy
		for i := range cJobs {
			totalPassed += cJobs[i].Passed
			totalRuns += cJobs[i].TotalRuns
			if severityRank(cJobs[i].Severity) > severityRank(worst) {
				worst = cJobs[i].Severity
			}
		}

		components = append(components, ComponentHealth{
			Component: name,
			Jobs:      len(cJobs),
			Severity:  worst,
			PassRate:  calcPassRate(totalPassed, totalRuns),
		})
	}

	sort.Slice(components, func(i, j int) bool {
		return components[i].Component < components[j].Component
	})

	return components
}

func classifyPassRate(passRate float64, warningThreshold, criticalThreshold int) Severity {
	switch {
	case passRate < float64(criticalThreshold):
		return SeverityCritical
	case passRate < float64(warningThreshold):
		return SeverityWarning
	default:
		return SeverityHealthy
	}
}

func calcPassRate(passed, total int) float64 {
	if total <= 0 {
		return 0
	}
	return float64(passed) / float64(total) * 100
}

func countConsecutiveFailures(runs []prow.JobRun) int {
	count := 0
	for i := range runs {
		if !runs[i].IsCompleted() || runs[i].State == prow.JobStateAborted {
			continue
		}
		if runs[i].State != prow.JobStateSuccess {
			count++
		} else {
			break
		}
	}
	return count
}

func severityRank(s Severity) int {
	switch s {
	case SeverityCritical:
		return 3
	case SeverityWarning:
		return 2
	case SeverityNoData:
		return 1
	default:
		return 0
	}
}
