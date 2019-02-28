[![CircleCI](https://circleci.com/gh/giantswarm/github-exporter.svg?&style=shield)](https://circleci.com/gh/giantswarm/github-exporter)

# github-exporter

The github-exporter exports Prometheus metrics for Github data.



### Example Execution

```
./github-exporter daemon --service.collector.issue.customlabels='[ "kind/okr,goal/achieved", "kind/okr,goal/missed", "postmortem,team/batman", "postmortem,team/magic", "postmortem,team/spirit" ]' --service.github.auth.token=$(cat ~/.credential/github-exporter-github-token)
```



### Example Queries

Showing a graph of the total number of open and closed issues.

```
github_exporter_issue_states_count
```

Showing a graph of open and closed postmortem issues.

```
github_exporter_issue_label_count{label="postmortem"}
```

Showing a graph of open and closed bug issues.

```
github_exporter_issue_label_count{label="kind/bug"}
```

Showing a graph of open and closed postmortem issues per team.

```
github_exporter_issue_labels_count{labels="postmortem,team/batman"}
github_exporter_issue_labels_count{labels="postmortem,team/magic"}
github_exporter_issue_labels_count{labels="postmortem,team/spirit"}
```

```
github_exporter_issue_labels_count{labels=~"postmortem,team/.*"}
```

Showing a graph of open and closed OKR issues per goal.

```
github_exporter_issue_labels_count{labels="kind/okr,goal/achieved"}
github_exporter_issue_labels_count{labels="kind/okr,goal/missed"}
```

```
github_exporter_issue_labels_count{labels=~"kind/okr,goal/.*"}
```

Showing a graph of postmortem issues per team to see how many days it took to
resolve them.

```
histogram_quantile(0.95, github_exporter_issue_labels_lifetime_bucket)
```
