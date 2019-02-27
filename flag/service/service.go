package service

import (
	"github.com/giantswarm/github-exporter/flag/service/collector"
	"github.com/giantswarm/github-exporter/flag/service/github"
)

type Service struct {
	Collector collector.Collector
	Github    github.Github
}
