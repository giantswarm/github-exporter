package github

import (
	"github.com/giantswarm/github-exporter/flag/service/github/auth"
)

type Github struct {
	Auth auth.Auth
}
