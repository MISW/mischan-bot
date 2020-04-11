package mischanbot

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
	branchPrefix = "mischan-bot/misw/mischan-bot/"
)

func NewMischanBotRepository(cfg *config.Config, ghs *ghsink.GitHubSink) repository.Repository {
	return &mischanBotRepository{
		config: cfg,
		ghs:    ghs,
	}
}

type mischanBotRepository struct {
	config *config.Config
	ghs    *ghsink.GitHubSink
}

var _ repository.Repository = &mischanBotRepository{}

func (pr *mischanBotRepository) FullName() string {
	return "MISW/mischan-bot"
}

func (pr *mischanBotRepository) getManifestaInstallationID(ctx context.Context, owner, repo string) (installationID int64, token string, err error) {
	ins, _, err := pr.ghs.AppsClient().Apps.FindRepositoryInstallation(ctx, owner, repo)

	if err != nil {
		return 0, "", xerrors.Errorf("failed to get installation for manifest repository: %w", err)
	}

	token, err = pr.ghs.InstallationToken(ctx, ins.GetID())

	if err != nil {
		return 0, "", xerrors.Errorf("failed to get token for installation: %w", err)
	}

	return ins.GetID(), token, nil
}

func (pr *mischanBotRepository) getLatestSHA(ctx context.Context, event *github.StatusEvent) (string, error) {
	targetBranch := "master"

	maybeCorrectBranch := false
	for i := range event.Branches {
		if event.Branches[i].GetName() == targetBranch {
			maybeCorrectBranch = true
		}
	}

	if len(event.Branches) == 10 {
		maybeCorrectBranch = true
	}

	if !maybeCorrectBranch {
		return "", nil
	}

	client := pr.ghs.InstallationClient(event.GetInstallation().GetID())

	master, _, err := client.Repositories.GetBranch(
		ctx,
		event.GetRepo().GetOwner().GetLogin(),
		event.GetRepo().GetName(),
		targetBranch,
	)

	if err != nil {
		return "", xerrors.Errorf("failed to get branch %s: %w", targetBranch, err)
	}

	return master.GetCommit().GetSHA(), nil
}

func (pr *mischanBotRepository) kustomize(shortSHA string) func(ctx context.Context, dir string) error {
	return func(ctx context.Context, dir string) error {
		cmd := exec.CommandContext(
			ctx, "kustomize", "edit", "set", "image", "registry.misw.jp/mischan-bot/mischan-bot:sha-"+shortSHA,
		)
		cmd.Dir = filepath.Join(dir, "bases/mischan-bot")

		b, err := cmd.CombinedOutput()

		if err != nil {
			return xerrors.Errorf("failed to kustomize(%s): %w", string(b), err)
		}

		return nil
	}
}

func (pr *mischanBotRepository) OnStatus(event *github.StatusEvent) error {
	if event.GetState() != "success" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	appLatestSHA, err := pr.getLatestSHA(ctx, event)

	if err != nil {
		return xerrors.Errorf("failed to get latest sha for app repository: %w", err)
	}

	if appLatestSHA != event.GetSHA() {
		// Old commit
		return nil
	}

	manimani, err := manifrepo.NewManifestManipulator(ctx, pr.ghs, "MISW/k8s")

	if err != nil {
		return xerrors.Errorf("failed to initialize GitHub client for manifest repository: %w", err)
	}

	if err := manimani.CloseObsolatePRs(ctx, branchPrefix); err != nil {
		return xerrors.Errorf("failed to close obsolete PRs: %w", err)
	}

	shortSHA := appLatestSHA[:7]

	if err := manimani.CreatePullRequest(
		ctx,
		branchPrefix+shortSHA,
		fmt.Sprintf("Update MISW/mischan-bot to "),
		pr.kustomize(shortSHA),
	); err != nil {
		return xerrors.Errorf("failed to create pull request: %w", err)
	}

	return nil
}

func (pr *mischanBotRepository) OnPush(event *github.PushEvent) error {
	return nil
}
