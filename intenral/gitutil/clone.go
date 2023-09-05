package gitutil

import (
	"context"
	"net/url"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/go-github/v54/github"
	"golang.org/x/xerrors"
)

// GitHubUtil is a utility library for Git clients
type GitHubUtil struct {
	token  string
	client *github.Client
}

// NewGitHubUtil initializes GitHubUtil
func NewGitHubUtil(token string, client *github.Client) *GitHubUtil {
	return &GitHubUtil{
		token:  token,
		client: client,
	}
}

// CloneRepository clones a repository on local filesystem
func (ghu *GitHubUtil) CloneRepository(ctx context.Context, gitURL string, ref string) (repo *git.Repository, dir string, err error) {
	dir, err = os.MkdirTemp("", "mischan-bot-")

	if err != nil {
		return nil, "", xerrors.Errorf("failed to create temporary directory: %w", err)
	}

	u, err := url.Parse(gitURL)
	if err != nil {
		return nil, "", xerrors.Errorf("failed to parse git url: %w", err)
	}
	u.User = url.UserPassword("x-access-token", ghu.token)

	repo, err = git.PlainCloneContext(ctx, dir, false, &git.CloneOptions{
		URL:           u.String(),
		ReferenceName: plumbing.NewBranchReferenceName(ref),
	})

	if err != nil {
		return nil, "", xerrors.Errorf("failed to clone repository: %w", err)
	}

	return repo, dir, nil
}
