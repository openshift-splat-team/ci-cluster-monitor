package monitor

import (
	"time"
)

type Severity string

const (
	SeverityNormal   Severity = "normal"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
	SeverityStale    Severity = "stale"
)

type PRMetrics struct {
	Repo            string        `json:"repo"`
	Number          int           `json:"number"`
	Title           string        `json:"title"`
	Author          string        `json:"author"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
	Age             time.Duration `json:"age"`
	SinceLastUpdate time.Duration `json:"since_last_update"`
	IsDraft         bool          `json:"is_draft"`
	Labels          []string      `json:"labels"`
	CIStatus        string        `json:"ci_status"`
	Severity        Severity      `json:"severity"`
	HTMLURL         string        `json:"html_url"`
}

type RepoReport struct {
	Owner     string      `json:"owner"`
	Repo      string      `json:"repo"`
	PRs       []PRMetrics `json:"prs"`
	FetchedAt time.Time   `json:"fetched_at"`
}

func (r *RepoReport) FullName() string {
	return r.Owner + "/" + r.Repo
}
