package repository

import (
	"sync"

	"github.com/google/go-github/v45/github"
	"golang.org/x/xerrors"
)

// Repository handles webhook events for each repository
type Repository interface {
	OnPush(event *github.PushEvent) error

	OnCheckSuite(event *github.CheckSuiteEvent) error

	OnCreate(event *github.CreateEvent) error

	FullName() string
}

var (
	ErrUnknownRepository = xerrors.New("unknown repository")
)

type RepositoryBundler struct {
	repositories map[string]Repository
	lock         sync.RWMutex
}

func NewRepositoryBundler() *RepositoryBundler {
	return &RepositoryBundler{
		repositories: map[string]Repository{},
	}
}

// RegisterRepository registers a new target repository to update code
func (rb *RepositoryBundler) RegisterRepository(repository Repository) {
	rb.lock.Lock()
	defer rb.lock.Unlock()

	rb.repositories[repository.FullName()] = repository
}

func (rb *RepositoryBundler) OnCreate(event *github.CreateEvent) error {
	rb.lock.RLock()
	defer rb.lock.RUnlock()

	repo := event.GetRepo().GetFullName()

	handler, ok := rb.repositories[repo]

	if !ok {
		return ErrUnknownRepository
	}

	return handler.OnCreate(event)
}

func (rb *RepositoryBundler) OnCheckSuite(event *github.CheckSuiteEvent) error {
	rb.lock.RLock()
	defer rb.lock.RUnlock()

	repo := event.GetRepo().GetFullName()

	handler, ok := rb.repositories[repo]

	if !ok {
		return ErrUnknownRepository
	}

	return handler.OnCheckSuite(event)
}

func (rb *RepositoryBundler) OnPush(event *github.PushEvent) error {
	rb.lock.RLock()
	defer rb.lock.RUnlock()

	repo := event.GetRepo().GetFullName()

	handler, ok := rb.repositories[repo]

	if !ok {
		return ErrUnknownRepository
	}

	return handler.OnPush(event)
}
