package collector

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/google/go-github/github"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	issueLabelCountDesc *prometheus.Desc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subsystem, "label_count"),
		"Github issues per label.",
		[]string{
			labelOrg,
			labelRepo,
			labelLabel,
			labelState,
		},
		nil,
	)
	issueLabelsCountDesc *prometheus.Desc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subsystem, "labels_count"),
		"Github issues per combined labels.",
		[]string{
			labelOrg,
			labelRepo,
			labelLabels,
			labelState,
		},
		nil,
	)
	issueStatesCountDesc *prometheus.Desc = prometheus.NewDesc(
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

var (
	issueLabelLifetimeHistogramVec = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    prometheus.BuildFQName(namespace, subsystem, "label_lifetime"),
			Buckets: prometheus.ExponentialBuckets(60*60*24, 2, 10),
		},
		[]string{
			labelOrg,
			labelRepo,
			labelLabels,
		},
	)
	issueLabelsLifetimeHistogramVec = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    prometheus.BuildFQName(namespace, subsystem, "labels_lifetime"),
			Buckets: prometheus.ExponentialBuckets(60*60*24, 2, 10),
		},
		[]string{
			labelOrg,
			labelRepo,
			labelLabels,
		},
	)
)

func init() {
	prometheus.MustRegister(issueLabelLifetimeHistogramVec)
	prometheus.MustRegister(issueLabelsLifetimeHistogramVec)
}

type IssueConfig struct {
	GithubClient *github.Client
	Logger       micrologger.Logger

	CustomLabels []string
}

type Issue struct {
	githubClient *github.Client
	logger       micrologger.Logger

	customLabels []string
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

		customLabels: config.CustomLabels,
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
		Since: time.Now().AddDate(-1, 0, 0), // one year ago
		State: "all",
	}

	type key struct {
		Label  string
		Number string
		State  string
	}

	issueLabelCount := map[key]float64{}
	issueLabelsCount := map[key]float64{}
	issueStatesCount := map[string]float64{}

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
				{
					k := key{
						Label: label.GetName(),
						State: issue.GetState(),
					}
					issueLabelCount[k] = issueLabelCount[k] + 1
				}

				if issue.GetState() == "closed" {
					f := float64(issue.GetClosedAt().Unix() - issue.GetCreatedAt().Unix())
					issueLabelLifetimeHistogramVec.WithLabelValues(githubOrg, githubRepo, label.GetName()).Observe(f)
				}
			}

			for _, selector := range i.customLabels {
				if !hasLabels(issue, selector) {
					continue
				}

				{
					k := key{
						Label: selector,
						State: issue.GetState(),
					}
					issueLabelsCount[k] = issueLabelsCount[k] + 1
				}

				if issue.GetState() == "closed" {
					f := float64(issue.GetClosedAt().Unix() - issue.GetCreatedAt().Unix())
					issueLabelsLifetimeHistogramVec.WithLabelValues(githubOrg, githubRepo, selector).Observe(f)
				}
			}

			{
				issueStatesCount[issue.GetState()] = issueStatesCount[issue.GetState()] + 1
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

	for k, v := range issueLabelCount {
		ch <- prometheus.MustNewConstMetric(
			issueLabelCountDesc,
			prometheus.GaugeValue,
			v,
			githubOrg,
			githubRepo,
			k.Label,
			k.State,
		)
	}

	for k, v := range issueLabelsCount {
		ch <- prometheus.MustNewConstMetric(
			issueLabelsCountDesc,
			prometheus.GaugeValue,
			v,
			githubOrg,
			githubRepo,
			k.Label,
			k.State,
		)
	}

	for k, v := range issueStatesCount {
		ch <- prometheus.MustNewConstMetric(
			issueStatesCountDesc,
			prometheus.GaugeValue,
			v,
			githubOrg,
			githubRepo,
			k,
		)
	}

	return nil
}

func (i *Issue) Describe(ch chan<- *prometheus.Desc) error {
	ch <- issueLabelCountDesc
	ch <- issueLabelsCountDesc
	ch <- issueStatesCountDesc
	return nil
}

func hasLabels(issue *github.Issue, selector string) bool {
	selectorLabels := strings.Split(selector, ",")

	for _, selectorLabel := range selectorLabels {
		found := false

		for _, issueLabel := range issue.Labels {
			if issueLabel.GetName() == selectorLabel {
				found = true
				break
			}
		}

		if !found {
			return false
		}
	}

	return true
}
