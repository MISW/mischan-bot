package manifrepo

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/MISW/mischan-bot/intenral/ghsink"
	"github.com/MISW/mischan-bot/intenral/gitutil"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/go-github/v52/github"
	"golang.org/x/xerrors"
)

// ManifestManipulator is a utility for manifest repository
type ManifestManipulator struct {
	BaseBranch                  string
	CommiterEmail, CommiterName string

	ghs    *ghsink.GitHubSink
	client *github.Client

	owner, repo    string
	installationID int64
	token          string // will expire soon

	cachedLatestSHA string
}

// NewManifestManipulator initializes a manupulator for manifests
func NewManifestManipulator(ctx context.Context, ghs *ghsink.GitHubSink, repoName string) (*ManifestManipulator, error) {
	arr := strings.SplitN(repoName, "/", 2)

	if len(arr) != 2 {
		return nil, xerrors.New("repo name should be org_name/repo_name")
	}

	mm := &ManifestManipulator{
		ghs:   ghs,
		owner: arr[0],
		repo:  arr[1],

		BaseBranch:    "master",
		CommiterName:  "mischan-bot",
		CommiterEmail: "mischan-bot@users.noreply.github.com",
	}

	if err := mm.getInstallation(ctx); err != nil {
		return nil, xerrors.Errorf("failed to initialize installation id or client: %w", err)
	}

	return mm, nil
}

// GetInstallation initializes installation id and clients for manifest repository
func (mm *ManifestManipulator) getInstallation(ctx context.Context) error {
	ins, _, err := mm.ghs.AppsClient().Apps.FindRepositoryInstallation(ctx, mm.owner, mm.repo)

	if err != nil {
		return xerrors.Errorf("failed to get installation for manifest repository: %w", err)
	}

	token, err := mm.ghs.InstallationToken(ctx, ins.GetID())

	if err != nil {
		return xerrors.Errorf("failed to get token for installation: %w", err)
	}

	mm.installationID = ins.GetID()
	mm.token = token
	mm.client = mm.ghs.InstallationClient(ins.GetID())

	return nil
}

func (mm *ManifestManipulator) CloseObsoletePRs(ctx context.Context, branchPrefix string) error {
	obsoletePRs, _, err := mm.client.PullRequests.List(
		ctx,
		mm.owner,
		mm.repo,
		&github.PullRequestListOptions{
			State: "open",
		},
	)

	if err != nil {
		return xerrors.Errorf("failed to list obsolete prs: %w", err)
	}

	if len(obsoletePRs) != 0 {
		mm.cachedLatestSHA = obsoletePRs[0].GetBase().GetSHA()
	}

	var wg sync.WaitGroup
	for i := range obsoletePRs {
		if !strings.HasPrefix(
			obsoletePRs[i].GetHead().GetRef(),
			branchPrefix,
		) {
			continue
		}

		wg.Add(1)
		go func(pr *github.PullRequest) {
			defer wg.Done()

			_, _, err := mm.client.PullRequests.Edit(
				ctx,
				mm.owner,
				mm.repo,
				pr.GetNumber(),
				&github.PullRequest{
					State: github.String("closed"),
				},
			)

			if err != nil {
				log.Printf("failed to close pull request %d for %s/%s: %+v", pr.GetNumber(), mm.owner, mm.repo, err)
			}

			_, err = mm.client.Git.DeleteRef(ctx, mm.owner, mm.repo, "heads/"+pr.GetHead().GetRef())

			if err != nil {
				log.Printf("failed to delete branch for pull requesst %d for %s/%s: %+v", pr.GetNumber(), mm.owner, mm.repo, err)
			}
		}(obsoletePRs[i])
	}

	wg.Wait()

	return nil
}

func (mm *ManifestManipulator) getLatestSHA(ctx context.Context) error {
	ref, _, err := mm.client.Git.GetRef(ctx, mm.owner, mm.repo, "heads/"+mm.BaseBranch)

	if err != nil {
		return xerrors.Errorf("failed to get ref: %w", err)
	}

	mm.cachedLatestSHA = ref.GetObject().GetSHA()

	return nil
}

func (mm *ManifestManipulator) CreatePullRequest(
	ctx context.Context,
	branchName, commitMessage string,
	manipulator func(ctx context.Context, dir string) error,
) error {
	if len(mm.cachedLatestSHA) == 0 {
		if err := mm.getLatestSHA(ctx); err != nil {
			return xerrors.Errorf("failed to get latest SHA in %s: %w", mm.BaseBranch, err)
		}
	}

	_, _, err := mm.client.Git.CreateRef(
		ctx, mm.owner, mm.repo, &github.Reference{
			Ref:    github.String(branchName),
			Object: &github.GitObject{SHA: github.String(mm.cachedLatestSHA)},
		},
	)

	if err != nil {
		return xerrors.Errorf("failed to create branch(%s): %w", branchName, err)
	}

	gitutil := gitutil.NewGitHubUtil(mm.token, mm.client)

	gitrepo, dir, err := gitutil.CloneRepository(
		ctx,
		fmt.Sprintf("https://github.com/%s/%s.git", mm.owner, mm.repo),
		mm.BaseBranch,
	)
	defer os.RemoveAll(dir)

	if err != nil {
		return xerrors.Errorf("failed to clone repository: %w", err)
	}

	wt, err := gitrepo.Worktree()

	if err != nil {
		return xerrors.Errorf("failed to get worktree for git repo: %w", err)
	}

	if err := wt.Checkout(&git.CheckoutOptions{
		Create: true,
		Force:  true,
		Branch: plumbing.NewBranchReferenceName(branchName),
	}); err != nil {
		return xerrors.Errorf("failed to checkout branch %s: %w", branchName, err)
	}

	if err := manipulator(ctx, dir); err != nil {
		return xerrors.Errorf("updating image tag failed: %w", err)
	}

	stat, err := wt.Status()

	if err != nil {
		return xerrors.Errorf("failed to get status for git repository: %w", err)
	}

	if stat.IsClean() {
		return nil
	}

	if _, err := wt.Commit(commitMessage, &git.CommitOptions{
		All: true,
		Author: &object.Signature{
			Name:  mm.CommiterName,
			Email: mm.CommiterEmail,
			When:  time.Now(),
		},
	}); err != nil {
		return xerrors.Errorf("failed to commit changes: %w", err)
	}

	if err := gitrepo.PushContext(
		ctx,
		&git.PushOptions{
			RefSpecs: []config.RefSpec{
				config.RefSpec("+refs/heads/*:refs/heads/*"),
			},
		},
	); err != nil {
		return xerrors.Errorf("failed to push to remote repository: %w", err)
	}

	if _, _, err := mm.client.PullRequests.Create(
		ctx,
		mm.owner,
		mm.repo,
		&github.NewPullRequest{
			Title:               github.String(commitMessage),
			Head:                github.String(branchName),
			Base:                github.String(mm.BaseBranch),
			MaintainerCanModify: github.Bool(true),
		},
	); err != nil {
		return xerrors.Errorf("failed to create pull request: %w", err)
	}

	return nil
}
