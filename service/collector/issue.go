package collector

import (
	"context"
	"fmt"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/google/go-github/github"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	issueLabelsDesc *prometheus.Desc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subsystem, "labels_count"),
		"Github issue labels.",
		[]string{
			labelOrg,
			labelRepo,
			labelLabel,
			labelState,
		},
		nil,
	)
	issueStatesDesc *prometheus.Desc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subsystem, "states_count"),
		"Github issue states.",
		[]string{
			labelOrg,
			labelRepo,
			labelState,
		},
		nil,
	)
)

type IssueConfig struct {
	GithubClient *github.Client
	Logger       micrologger.Logger
}

type Issue struct {
	githubClient *github.Client
	logger       micrologger.Logger
}

func NewIssue(config IssueConfig) (*Issue, error) {
	if config.GithubClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.GithubClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	i := &Issue{
		githubClient: config.GithubClient,
		logger:       config.Logger,
	}

	return i, nil
}

func (i *Issue) Collect(ch chan<- prometheus.Metric) error {
	ctx := context.Background()
	opts := &github.IssueListByRepoOptions{
		ListOptions: github.ListOptions{
			Page: 1,
			// The maximum result in the API is 1000, which it not always guarantees.
			// In manual tests the number of issues received was 100. See also
			// https://developer.github.com/v3/search/#about-the-search-api.
			PerPage: 1000,
		},
		State: "all",
	}

	issueLabels := map[string]float64{}
	issueStates := map[string]float64{}
	for {
		issues, res, err := i.githubClient.Issues.ListByRepo(ctx, githubOrg, githubRepo, opts)
		if err != nil {
			return microerror.Mask(err)
		}

		i.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("collecting %3d issues of page %2d", len(issues), opts.Page))

		for _, issue := range issues {
			if issue.IsPullRequest() {
				continue
			}

			for _, label := range issue.Labels {
				k := fmt.Sprintf("%s:%s", label.GetName(), issue.GetState())

				_, ok := issueLabels[k]
				if !ok {
					issueLabels[k] = 1
				} else {
					issueLabels[k]++
				}
			}

			{
				_, ok := issueStates[issue.GetState()]
				if !ok {
					issueStates[issue.GetState()] = 1
				} else {
					issueStates[issue.GetState()]++
				}
			}
		}

		// Manage the paging mechanism. When NextPage is 0 we iterated through all
		// the pages and can stop loopong through. As long as there are pages left
		// we assign the next page to our options structure as given by the current
		// response.
		if res.NextPage == 0 {
			i.logger.LogCtx(ctx, "level", "debug", "message", "collected all issues")
			break
		}
		opts.Page = res.NextPage
	}

	for k, v := range issueLabels {
		split := strings.Split(k, ":")

		ch <- prometheus.MustNewConstMetric(
			issueLabelsDesc,
			prometheus.GaugeValue,
			v,

			githubOrg,
			githubRepo,
			split[0],
			split[1],
		)
	}

	for k, v := range issueStates {
		ch <- prometheus.MustNewConstMetric(
			issueStatesDesc,
			prometheus.GaugeValue,
			v,
			githubOrg,
			githubRepo,
			k,
		)
	}

	i.issueTimeToCloseVec.Ensure(timeToCloseLabels)

	for label, histogram := range i.issueTimeToCloseVec.Histograms() {
		ch <- prometheus.MustNewConstHistogram(
			issueTimeToCloseDesc,
			histogram.Count(), histogram.Sum(), histogram.Buckets(),
			githubOrg,
			githubRepo,
			label,
		)
	}

	return nil
}

func (i *Issue) Describe(ch chan<- *prometheus.Desc) error {
	ch <- issueLabelsDesc
	ch <- issueStatesDesc
	return nil
}
