package service

import (
	"context"
	"sync"

	"github.com/giantswarm/github-exporter/flag"
	"github.com/giantswarm/github-exporter/service/collector"
	"github.com/giantswarm/microendpoint/service/version"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/google/go-github/github"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

type Config struct {
	Logger micrologger.Logger

	Description string
	Flag        *flag.Flag
	GitCommit   string
	ProjectName string
	Source      string
	Viper       *viper.Viper
}

type Service struct {
	Version *version.Service

	bootOnce          sync.Once
	exporterCollector *collector.Set
}

func New(config Config) (*Service, error) {
	// Dependencies.
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	// Settings.
	if config.Flag == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Flag must not be empty", config)
	}
	if config.Viper == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Viper must not be empty", config)
	}

	var err error

	var githubClient *github.Client
	{
		ctx := context.Background()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{
				AccessToken: config.Viper.GetString(config.Flag.Service.Github.Auth.Token),
			},
		)

		githubClient = github.NewClient(oauth2.NewClient(ctx, ts))
	}

	var exporterCollector *collector.Set
	{
		c := collector.SetConfig{
			GithubClient: githubClient,
			Logger:       config.Logger,
		}

		exporterCollector, err = collector.NewSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var versionService *version.Service
	{
		c := version.Config{
			Description: config.Description,
			GitCommit:   config.GitCommit,
			Name:        config.ProjectName,
			Source:      config.Source,
		}

		versionService, err = version.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	s := &Service{
		Version: versionService,

		bootOnce:          sync.Once{},
		exporterCollector: exporterCollector,
	}

	return s, nil
}

func (s *Service) Boot(ctx context.Context) {
	s.bootOnce.Do(func() {
		go s.exporterCollector.Boot(ctx)
	})
}
