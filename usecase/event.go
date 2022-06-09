package usecase

import (
	"log"

	"github.com/MISW/mischan-bot/repository"
	"github.com/google/go-github/v45/github"
)

// GitHubEventUsecase handles GtiHub webhook events
type GitHubEventUsecase interface {
	Create(e *github.CreateEvent) error
	CheckSuite(e *github.CheckSuiteEvent) error
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

// Push handles push events
// ref. https://developer.github.com/v3/activity/events/types/#pushevent
func (geu *gitHubEventUsecase) Push(e *github.PushEvent) error {
	go func() {
		if err := geu.repoBundler.OnPush(e); err != nil {
			if err == repository.ErrUnknownRepository {
				return
			}

			log.Printf("push event failed for %s: %+v", e.GetRepo().GetFullName(), err)
		}
	}()

	return nil
}

// Create handles create events
// ref. https://developer.github.com/v3/activity/events/types/#createevent
func (geu *gitHubEventUsecase) Create(e *github.CreateEvent) error {
	go func() {
		if err := geu.repoBundler.OnCreate(e); err != nil {
			if err == repository.ErrUnknownRepository {
				return
			}

			log.Printf("create event failed for %s: %+v", e.GetRepo().GetFullName(), err)
		}
	}()

	return nil
}

// CheckSuite handles check suite events
// ref. https://developer.github.com/v3/activity/events/types/#checksuiteevent
func (geu *gitHubEventUsecase) CheckSuite(e *github.CheckSuiteEvent) error {
	go func() {
		if err := geu.repoBundler.OnCheckSuite(e); err != nil {
			if err == repository.ErrUnknownRepository {
				return
			}

			log.Printf("check_suite event failed for %s: %+v", e.GetRepo().GetFullName(), err)
		}
	}()

	return nil
}
