package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v72/github"
	"k8s.io/klog/v2"
)

type PullRequest struct {
	Number    int
	Title     string
	Author    string
	CreatedAt github.Timestamp
	UpdatedAt github.Timestamp
	IsDraft   bool
	Labels    []string
	HeadRef   string
	HeadSHA   string
	HTMLURL   string
}

type WorkflowRunStatus struct {
	WorkflowName string
	Status       string
	Conclusion   string
}

func (c *Client) ListOpenPRs(ctx context.Context, owner, repo string) ([]PullRequest, error) {
	var allPRs []PullRequest

	opts := &github.PullRequestListOptions{
		State:     "open",
		Sort:      "created",
		Direction: "asc",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		prs, resp, err := c.gh.PullRequests.List(ctx, owner, repo, opts)
		if err != nil {
			return nil, fmt.Errorf("listing PRs for %s/%s: %w", owner, repo, err)
		}
		c.logRateLimit(resp)

		for _, pr := range prs {
			labels := make([]string, 0, len(pr.Labels))
			for _, l := range pr.Labels {
				labels = append(labels, l.GetName())
			}

			allPRs = append(allPRs, PullRequest{
				Number:    pr.GetNumber(),
				Title:     pr.GetTitle(),
				Author:    pr.GetUser().GetLogin(),
				CreatedAt: pr.GetCreatedAt(),
				UpdatedAt: pr.GetUpdatedAt(),
				IsDraft:   pr.GetDraft(),
				Labels:    labels,
				HeadRef:   pr.GetHead().GetRef(),
				HeadSHA:   pr.GetHead().GetSHA(),
				HTMLURL:   pr.GetHTMLURL(),
			})
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allPRs, nil
}

func (c *Client) GetLatestWorkflowRunStatus(
	ctx context.Context,
	owner, repo string,
	pr *PullRequest,
	workflows []string,
) ([]WorkflowRunStatus, error) {
	var statuses []WorkflowRunStatus

	for _, wf := range workflows {
		opts := &github.ListWorkflowRunsOptions{
			Branch: pr.HeadRef,
			Event:  "pull_request",
			ListOptions: github.ListOptions{
				PerPage: 5,
			},
		}

		runs, resp, err := c.gh.Actions.ListWorkflowRunsByFileName(ctx, owner, repo, wf, opts)
		if err != nil {
			klog.Warningf("Failed to list workflow runs for %s workflow %s PR #%d: %v", owner+"/"+repo, wf, pr.Number, err)
			statuses = append(statuses, WorkflowRunStatus{
				WorkflowName: wf,
				Status:       "unavailable",
			})
			continue
		}
		c.logRateLimit(resp)

		status := WorkflowRunStatus{
			WorkflowName: wf,
			Status:       "no-runs",
			Conclusion:   "",
		}

		for _, run := range runs.WorkflowRuns {
			if run.GetHeadSHA() == pr.HeadSHA {
				status.Status = run.GetStatus()
				status.Conclusion = run.GetConclusion()
				break
			}
		}

		statuses = append(statuses, status)
	}

	return statuses, nil
}
