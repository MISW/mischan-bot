package ghsink

import (
	"context"
	"net/http"

	"github.com/MISW/mischan-bot/config"
	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v53/github"
	"golang.org/x/xerrors"
)

// GitHubSink is a base to initialize GitHub App clients
type GitHubSink struct {
	appsTransport *ghinstallation.AppsTransport
}

// NewGitHubSink initializes a utility to initialize GitHub App client
func NewGitHubSink(cfg *config.Config) (*GitHubSink, error) {
	tr := http.DefaultTransport

	var appsTransport *ghinstallation.AppsTransport
	var err error

	if cfg.PrivateKey.Path != "" {
		appsTransport, err = ghinstallation.NewAppsTransportKeyFromFile(tr, cfg.AppID, cfg.PrivateKey.Path)
	} else {
		appsTransport, err = ghinstallation.NewAppsTransport(tr, cfg.AppID, []byte(cfg.PrivateKey.Raw))
	}

	if err != nil {
		return nil, xerrors.Errorf("failed to initialize app transport: %w", err)
	}

	return &GitHubSink{
		appsTransport: appsTransport,
	}, nil
}

// AppsClient returns apps client(no specific installation)
func (ghs *GitHubSink) AppsClient() *github.Client {
	return github.NewClient(&http.Client{Transport: ghs.appsTransport})
}

// InstallationClient returns API client for specific installation
func (ghs *GitHubSink) InstallationClient(installationID int64) *github.Client {
	itr := ghinstallation.NewFromAppsTransport(ghs.appsTransport, installationID)

	return github.NewClient(&http.Client{Transport: itr})
}

// InstallationToken returns token for specific installation
func (ghs *GitHubSink) InstallationToken(ctx context.Context, installationID int64) (string, error) {
	itr := ghinstallation.NewFromAppsTransport(ghs.appsTransport, installationID)

	token, err := itr.Token(ctx)

	if err != nil {
		return "", xerrors.Errorf("failed to generate token: %w", err)
	}

	return token, nil
}
