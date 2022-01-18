package birdol

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/MISW/mischan-bot/config"
	"github.com/MISW/mischan-bot/intenral/ghsink"
	"github.com/MISW/mischan-bot/intenral/manifrepo"
	"github.com/MISW/mischan-bot/repository"
	"github.com/google/go-github/v42/github"
	"golang.org/x/xerrors"
)

const (
	branchPrefix = "mischan-bot/misw/birdol-server/"
)

// NewBirdolRepository initializes repository for MISW/birdol-server
func NewBirdolRepository(cfg *config.Config, ghs *ghsink.GitHubSink, app *github.App, botUser *github.User) repository.Repository {
	return &gitOpsRepository{
		config:       cfg,
		ghs:          ghs,
		app:          app,
		botUser:      botUser,
		targetBranch: "main",

		owner: "MISW",
		repo:  "birdol-server",
	}
}

type gitOpsRepository struct {
	config  *config.Config
	ghs     *ghsink.GitHubSink
	app     *github.App
	botUser *github.User

	targetBranch string
	owner, repo  string
}

var _ repository.Repository = &gitOpsRepository{}

func (gor *gitOpsRepository) FullName() string {
	return gor.owner + "/" + gor.repo
}

func (gor *gitOpsRepository) checkSuiteStatus(
	ctx context.Context,
	installationID int64,
) (success bool, sha string, err error) {
	client := gor.ghs.InstallationClient(installationID)

	checkRuns, _, err := client.Checks.ListCheckRunsForRef(ctx, gor.owner, gor.repo, gor.targetBranch, nil)

	if err != nil {
		return false, "", xerrors.Errorf("failed list check suites for %s/%s: %w", gor.owner, gor.repo, err)
	}

	if len(checkRuns.CheckRuns) == 0 {
		return false, "", nil
	}

	success = true
	for _, suite := range checkRuns.CheckRuns {
		if suite.GetStatus() != "completed" {
			success = false
			break
		}

		conclusion := suite.GetConclusion()
		
		if conclusion != "success" && conclusion != "neutral" && conclusion != "skipped" {
			success = false
			break
		}

		sha = suite.GetHeadSHA()
	}

	return
}

func (gor *gitOpsRepository) kustomize(shortSHA string) func(ctx context.Context, dir string) error {
	return func(ctx context.Context, dir string) error {
		cmd := exec.CommandContext(
			ctx, "kustomize", "edit", "set", "image", "registry.misw.jp/birdol/web:sha-"+shortSHA,
		)
		cmd.Dir = filepath.Join(dir, "bases/birdol")

		b, err := cmd.CombinedOutput()

		if err != nil {
			return xerrors.Errorf("failed to kustomize(%s): %w", string(b), err)
		}

		return nil
	}
}

func (gor *gitOpsRepository) run(installationID int64, expectedSHA string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	success, sha, err := gor.checkSuiteStatus(ctx, installationID)

	if err != nil {
		return xerrors.Errorf("failed to get latest check suite: %w", err)
	}

	if !success {
		return nil
	}

	if len(expectedSHA) != 0 && sha != expectedSHA {
		return nil
	}

	manimani, err := manifrepo.NewManifestManipulator(ctx, gor.ghs, "MISW/k8s")

	if err != nil {
		return xerrors.Errorf("failed to initialize GitHub client for manifest repository: %w", err)
	}

	manimani.CommiterName = gor.app.GetName()
	manimani.CommiterEmail = fmt.Sprintf("%d+%s[bot]@users.noreply.github.com", gor.botUser.GetID(), gor.app.GetSlug())

	if err := manimani.CloseObsoletePRs(ctx, branchPrefix); err != nil {
		return xerrors.Errorf("failed to close obsolete PRs: %w", err)
	}

	shortSHA := sha[:7]

	if err := manimani.CreatePullRequest(
		ctx,
		branchPrefix+shortSHA,
		fmt.Sprintf("Update MISW/birdol-server to %s", shortSHA),
		gor.kustomize(shortSHA),
	); err != nil {
		return xerrors.Errorf("failed to create pull request: %w", err)
	}

	return nil

}

func (gor *gitOpsRepository) OnCheckSuite(event *github.CheckSuiteEvent) error {
	if event.GetCheckSuite().GetHeadBranch() != gor.targetBranch {
		return nil
	}

	err := gor.run(
		event.GetInstallation().GetID(),
		event.GetCheckSuite().GetHeadSHA(),
	)

	if err != nil {
		return xerrors.Errorf("check suite handler failed: %w", err)
	}

	return nil
}

func (gor *gitOpsRepository) OnCreate(event *github.CreateEvent) error {
	if event.GetRefType() != "branch" || event.GetRef() != gor.targetBranch {
		return nil
	}

	err := gor.run(
		event.GetInstallation().GetID(),
		"",
	)

	if err != nil {
		return xerrors.Errorf("check suite handler failed: %w", err)
	}

	return nil
}

func (gor *gitOpsRepository) OnPush(event *github.PushEvent) error {
	if event.GetRef() != "refs/heads/"+gor.targetBranch {
		return nil
	}

	err := gor.run(
		event.GetInstallation().GetID(),
		"",
	)

	if err != nil {
		return xerrors.Errorf("check suite handler failed: %w", err)
	}

	return nil
}
