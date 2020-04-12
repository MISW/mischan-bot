package portal

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
	"github.com/google/go-github/v30/github"
	"golang.org/x/xerrors"
)

const (
	branchPrefix = "mischan-bot/misw/portal/"
)

func NewPortalRepository(cfg *config.Config, ghs *ghsink.GitHubSink) repository.Repository {
	return &portalRepository{
		config:       cfg,
		ghs:          ghs,
		targetBranch: "master",

		owner: "MISW",
		repo:  "Portal",
	}
}

type portalRepository struct {
	config *config.Config
	ghs    *ghsink.GitHubSink

	targetBranch string
	owner, repo  string
}

var _ repository.Repository = &portalRepository{}

func (pr *portalRepository) FullName() string {
	return pr.owner + "/" + pr.repo
}

func (pr *portalRepository) checkSuiteStatus(
	ctx context.Context,
	installationID int64,
) (success bool, sha string, err error) {
	client := pr.ghs.InstallationClient(installationID)

	checkSuites, _, err := client.Checks.ListCheckSuitesForRef(ctx, pr.owner, pr.repo, pr.targetBranch, nil)

	if err != nil {
		return false, "", xerrors.Errorf("failed list check suites for %s/%s: %w", pr.owner, pr.repo, err)
	}

	if len(checkSuites.CheckSuites) == 0 {
		return false, "", nil
	}

	success = true
	for _, suite := range checkSuites.CheckSuites {
		if suite.GetStatus() != "completed" {
			success = false
			break
		}

		if suite.GetConclusion() != "success" {
			success = false
			break
		}

		sha = suite.GetHeadSHA()
	}

	return
}

func (pr *portalRepository) kustomize(shortSHA string) func(ctx context.Context, dir string) error {
	return func(ctx context.Context, dir string) error {
		cmd := exec.CommandContext(
			ctx, "kustomize", "edit", "set", "image", "registry.misw.jp/portal/frontend:sha-"+shortSHA,
		)
		cmd.Dir = filepath.Join(dir, "bases/portal")

		b, err := cmd.CombinedOutput()

		if err != nil {
			return xerrors.Errorf("failed to kustomize(%s): %w", string(b), err)
		}

		cmd = exec.CommandContext(
			ctx, "kustomize", "edit", "set", "image", "registry.misw.jp/portal/backend:sha-"+shortSHA,
		)
		cmd.Dir = filepath.Join(dir, "bases/portal")

		b, err = cmd.CombinedOutput()

		if err != nil {
			return xerrors.Errorf("failed to kustomize(%s): %w", string(b), err)
		}

		return nil
	}
}

func (pr *portalRepository) run(installationID int64, expectedSHA string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	success, sha, err := pr.checkSuiteStatus(ctx, installationID)

	if err != nil {
		return xerrors.Errorf("failed to get latest check suite: %w", err)
	}

	if !success {
		return nil
	}

	if len(expectedSHA) != 0 && sha != expectedSHA {
		return nil
	}

	manimani, err := manifrepo.NewManifestManipulator(ctx, pr.ghs, "MISW/k8s")

	if err != nil {
		return xerrors.Errorf("failed to initialize GitHub client for manifest repository: %w", err)
	}

	if err := manimani.CloseObsolatePRs(ctx, branchPrefix); err != nil {
		return xerrors.Errorf("failed to close obsolete PRs: %w", err)
	}

	shortSHA := sha[:7]

	if err := manimani.CreatePullRequest(
		ctx,
		branchPrefix+shortSHA,
		fmt.Sprintf("Update MISW/portal to "),
		pr.kustomize(shortSHA),
	); err != nil {
		return xerrors.Errorf("failed to create pull request: %w", err)
	}

	return nil

}

func (pr *portalRepository) OnCheckSuite(event *github.CheckSuiteEvent) error {
	if event.GetCheckSuite().GetHeadBranch() != pr.targetBranch {
		return nil
	}

	err := pr.run(
		event.GetInstallation().GetID(),
		event.GetCheckSuite().GetHeadSHA(),
	)

	if err != nil {
		return xerrors.Errorf("check suite handler failed: %w", err)
	}

	return nil
}

func (pr *portalRepository) OnCreate(event *github.CreateEvent) error {
	if event.GetRefType() != "branch" || event.GetRef() != pr.targetBranch {
		return nil
	}

	err := pr.run(
		event.GetInstallation().GetID(),
		"",
	)

	if err != nil {
		return xerrors.Errorf("check suite handler failed: %w", err)
	}

	return nil
}

func (pr *portalRepository) OnPush(event *github.PushEvent) error {
	if event.GetRef() != "refs/heads/"+pr.targetBranch {
		return nil
	}

	err := pr.run(
		event.GetInstallation().GetID(),
		"",
	)

	if err != nil {
		return xerrors.Errorf("check suite handler failed: %w", err)
	}

	return nil
}
