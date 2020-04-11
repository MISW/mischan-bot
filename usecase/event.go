package usecase

import (
	"log"

	"github.com/MISW/mischan-bot/repository"
	"github.com/google/go-github/v30/github"
)

// GitHubEventUsecase handles GtiHub webhook events
type GitHubEventUsecase interface {
	Status(e *github.StatusEvent) error
	Push(e *github.PushEvent) error
}

var _ GitHubEventUsecase = &gitHubEventUsecase{}

type gitHubEventUsecase struct {
	repoBundler *repository.RepositoryBundler
}

// NewGitHubEventUsecase initializes GitHubEventUsecase
func NewGitHubEventUsecase(repoBundler *repository.RepositoryBundler) GitHubEventUsecase {
	return &gitHubEventUsecase{
		repoBundler: repoBundler,
	}
}

// Status handles status event
func (geu *gitHubEventUsecase) Status(e *github.StatusEvent) error {
	if err := geu.repoBundler.OnStatus(e); err != nil {
		if err == repository.ErrUnknownRepository {
			return nil
		}

		log.Printf("status event failed for %s: %+v", e.GetRepo().GetFullName(), err)
	}

	return nil
}

// Push handles push event
func (geu *gitHubEventUsecase) Push(e *github.PushEvent) error {
	if err := geu.repoBundler.OnPush(e); err != nil {
		if err == repository.ErrUnknownRepository {
			return nil
		}

		log.Printf("push event failed for %s: %+v", e.GetRepo().GetFullName(), err)
	}

	return nil
}
