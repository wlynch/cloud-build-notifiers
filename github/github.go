package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/v32/github"
	pb "google.golang.org/genproto/googleapis/devtools/cloudbuild/v1"
	tspb "google.golang.org/protobuf/types/known/timestamppb"
)

type Client struct {
	*github.Client
	owner, repo, commit string
}

func (n *notifier) githubClient(ctx context.Context, b *pb.Build, t *pb.BuildTrigger) (*Client, error) {
	owner := t.GetGithub().GetOwner()
	repo := t.GetGithub().GetName()
	commit := b.GetSubstitutions()["COMMIT_SHA"]

	// Since we're using a different GitHub App, we need to look up the installation for the repo.
	fmt.Println(n.atr)
	client := github.NewClient(&http.Client{Transport: n.atr})
	i, _, err := client.Apps.FindRepositoryInstallation(ctx, owner, repo)
	if err != nil {
		return nil, err
	}

	itr := ghinstallation.NewFromAppsTransport(n.atr, i.GetID())
	return &Client{
		Client: github.NewClient(&http.Client{Transport: itr}),
		owner:  owner,
		repo:   repo,
		commit: commit,
	}, nil
}

// UpsertCheckRun updates or creates a check run for the given TaskRun.
func (c *Client) UpsertCheckRun(ctx context.Context, b *pb.Build, t *pb.BuildTrigger, output *github.CheckRunOutput) (*github.CheckRun, error) {
	status, conclusion := status(b.GetStatus())
	curr := c.findCheckRun(ctx, b.GetId())
	if id := curr.GetID(); id != 0 {
		cr, _, err := c.Checks.UpdateCheckRun(ctx, c.owner, c.repo, id, github.UpdateCheckRunOptions{
			ExternalID:  github.String(b.GetId()),
			Name:        t.GetName(),
			Status:      github.String(status),
			Conclusion:  github.String(conclusion),
			HeadSHA:     github.String(c.commit),
			Output:      output,
			CompletedAt: ghtime(b.GetFinishTime()),
			DetailsURL:  github.String(b.GetLogUrl()),
		})
		if err != nil {
			return nil, fmt.Errorf("CreateCheck: %w", err)
		}
		return cr, nil
	}

	// There's no existing CheckRun - create.
	cr, _, err := c.Checks.CreateCheckRun(ctx, c.owner, c.repo, github.CreateCheckRunOptions{
		ExternalID:  github.String(b.GetId()),
		Name:        t.GetName(),
		Status:      github.String(status),
		Conclusion:  github.String(conclusion),
		HeadSHA:     c.commit,
		Output:      output,
		StartedAt:   ghtime(b.GetCreateTime()),
		CompletedAt: ghtime(b.GetFinishTime()),
		DetailsURL:  github.String(b.GetLogUrl()),
	})
	if err != nil {
		return nil, fmt.Errorf("CreateCheck: %w", err)
	}
	return cr, nil
}

func (c *Client) findCheckRun(ctx context.Context, name string) *github.CheckRun {
	checks, _, err := c.Checks.ListCheckRunsForRef(ctx, c.owner, c.repo, c.commit, &github.ListCheckRunsOptions{CheckName: github.String(name)})
	if err != nil {
		return nil
	}
	if cr := checks.CheckRuns; len(cr) > 0 {
		return cr[0]
	}
	return nil
}

const (
	CheckRunStatusQueued     = "queued"
	CheckRunStatusInProgress = "in_progress"
	CheckRunStatusCompleted  = "completed"

	CheckRunConclusionSuccess        = "success"
	CheckRunConclusionFailure        = "failure"
	CheckRunConclusionCancelled      = "cancelled"
	CheckRunConclusionTimeout        = "timed_out"
	CheckRunConclusionActionRequired = "action_required"
)

func status(s pb.Build_Status) (status, conclusion string) {
	switch s {
	case pb.Build_QUEUED:
		return CheckRunStatusQueued, ""
	case pb.Build_WORKING:
		return CheckRunStatusInProgress, ""
	case pb.Build_SUCCESS:
		return CheckRunStatusCompleted, CheckRunConclusionSuccess
	case pb.Build_FAILURE, pb.Build_INTERNAL_ERROR, pb.Build_STATUS_UNKNOWN:
		return CheckRunStatusCompleted, CheckRunConclusionFailure
	case pb.Build_CANCELLED, pb.Build_EXPIRED:
		return CheckRunStatusCompleted, CheckRunConclusionCancelled
	case pb.Build_TIMEOUT:
		return CheckRunStatusCompleted, CheckRunConclusionTimeout
	}

	return "", ""
}

func ghtime(t *tspb.Timestamp) *github.Timestamp {
	if t == nil {
		return nil
	}
	return &github.Timestamp{Time: t.AsTime()}
}
