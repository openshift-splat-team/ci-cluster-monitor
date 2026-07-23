// Copyright 2025 Nutanix. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package github

import (
	"context"

	"github.com/google/go-github/v72/github"
	"golang.org/x/oauth2"
	"k8s.io/klog/v2"
)

type Client struct {
	gh *github.Client
}

func NewClient(ctx context.Context, token string) *Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)

	return &Client{
		gh: github.NewClient(tc),
	}
}

func (c *Client) logRateLimit(resp *github.Response) {
	if resp == nil {
		return
	}
	remaining := resp.Rate.Remaining
	if remaining < 100 {
		klog.Warningf("GitHub API rate limit low, remaining=%d, limit=%d, reset=%s", remaining, resp.Rate.Limit, resp.Rate.Reset.Time)
	}
}
