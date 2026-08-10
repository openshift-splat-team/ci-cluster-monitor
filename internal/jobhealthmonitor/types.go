package jobhealthmonitor

import "time"

type Severity string

const (
	SeverityHealthy  Severity = "healthy"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
	SeverityNoData   Severity = "no-data"
)

type JobHealth struct {
	JobName             string    `json:"job_name"`
	Component           string    `json:"component,omitempty"`
	TotalRuns           int       `json:"total_runs"`
	Passed              int       `json:"passed"`
	Failed              int       `json:"failed"`
	PassRate            float64   `json:"pass_rate"`
	Severity            Severity  `json:"severity"`
	LastRunTime         time.Time `json:"last_run_time"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
}

type ComponentHealth struct {
	Component string   `json:"component"`
	Jobs      int      `json:"jobs"`
	Severity  Severity `json:"severity"`
	PassRate  float64  `json:"pass_rate"`
}

type JobHealthReport struct {
	Jobs         []JobHealth       `json:"jobs"`
	Components   []ComponentHealth `json:"components,omitempty"`
	FetchedAt    time.Time         `json:"fetched_at"`
	LookbackDays int               `json:"lookback_days"`
}

func (r *JobHealthReport) HasCritical() bool {
	for i := range r.Jobs {
		if r.Jobs[i].Severity == SeverityCritical {
			return true
		}
	}
	return false
}

func (r *JobHealthReport) HasWarning() bool {
	for i := range r.Jobs {
		if r.Jobs[i].Severity == SeverityWarning {
			return true
		}
	}
	return false
}
