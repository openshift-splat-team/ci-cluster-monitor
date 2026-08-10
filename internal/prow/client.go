package prow

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"k8s.io/klog/v2"
)

type JobState string

const (
	JobStateSuccess JobState = "success"
	JobStateFailure JobState = "failure"
	JobStateError   JobState = "error"
	JobStateAborted JobState = "aborted"
	JobStatePending JobState = "pending"
)

type JobRun struct {
	Job            string
	State          JobState
	Type           string
	StartTime      time.Time
	CompletionTime time.Time
}

func (r *JobRun) IsCompleted() bool {
	switch r.State {
	case JobStateSuccess, JobStateFailure, JobStateError, JobStateAborted:
		return true
	default:
		return false
	}
}

type Client struct {
	httpClient *http.Client
	baseURL    string
}

func NewClient(baseURL string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 60 * time.Second},
		baseURL:    strings.TrimRight(baseURL, "/"),
	}
}

func (c *Client) ListRecentRuns(ctx context.Context, jobNames map[string]bool, since time.Time) ([]JobRun, error) {
	url := c.baseURL + "/prowjobs.js?omit=annotations,labels,decoration_config,pod_spec"
	klog.V(2).Infof("Fetching prowjobs from %s", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching prowjobs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d from Prow API", resp.StatusCode)
	}

	var prowList prowJobList
	if err := json.NewDecoder(resp.Body).Decode(&prowList); err != nil {
		return nil, fmt.Errorf("decoding prowjobs response: %w", err)
	}

	var runs []JobRun
	for i := range prowList.Items {
		pj := &prowList.Items[i]
		if !jobNames[pj.Spec.Job] {
			continue
		}
		if pj.Status.StartTime.Before(since) {
			continue
		}

		run := JobRun{
			Job:       pj.Spec.Job,
			State:     JobState(pj.Status.State),
			Type:      pj.Spec.Type,
			StartTime: pj.Status.StartTime,
		}
		if pj.Status.CompletionTime != nil {
			run.CompletionTime = *pj.Status.CompletionTime
		}
		runs = append(runs, run)
	}

	klog.V(2).Infof("Found %d matching job runs out of %d total prowjobs", len(runs), len(prowList.Items))
	return runs, nil
}

type prowJobList struct {
	Items []prowJob `json:"items"`
}

type prowJob struct {
	Spec   prowJobSpec   `json:"spec"`
	Status prowJobStatus `json:"status"`
}

type prowJobSpec struct {
	Job  string `json:"job"`
	Type string `json:"type"`
}

type prowJobStatus struct {
	State          string     `json:"state"`
	StartTime      time.Time  `json:"startTime"`
	CompletionTime *time.Time `json:"completionTime,omitempty"`
}
