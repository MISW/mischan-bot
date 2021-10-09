package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/MISW/mischan-bot/config"
	"github.com/MISW/mischan-bot/handler"
	"github.com/MISW/mischan-bot/intenral/ghsink"
	"github.com/MISW/mischan-bot/repository"
	"github.com/MISW/mischan-bot/repository/birdol"
	"github.com/MISW/mischan-bot/repository/mischanbot"
	"github.com/MISW/mischan-bot/repository/modoki"
	"github.com/MISW/mischan-bot/repository/portal"
	"github.com/MISW/mischan-bot/usecase"
	"github.com/google/go-github/v30/github"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/dig"
	"golang.org/x/xerrors"
)

func must(err error) {
	if err != nil {
		log.Fatalf("failed to initialize container: %+v", err)
	}
}

func main() {
	container := dig.New()

	must(container.Provide(func() *echo.Echo {
		return echo.New()
	}))

	must(container.Provide(func() (*config.Config, error) {
		cfg, err := config.ReadConfig()

		if err != nil {
			return nil, xerrors.Errorf("failed to initialize config: %w", err)
		}

		return cfg, nil
	}))

	must(container.Provide(usecase.NewGitHubEventUsecase))

	must(container.Provide(repository.NewRepositoryBundler))

	must(container.Provide(func(cfg *config.Config) (*ghsink.GitHubSink, error) {
		ghs, err := ghsink.NewGitHubSink(cfg)

		if err != nil {
			return nil, xerrors.Errorf("failed to initialize GitHub App client sink: %w", err)
		}

		return ghs, nil
	}))

	must(container.Provide(func(ghs *ghsink.GitHubSink) (*github.App, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		app, _, err := ghs.AppsClient().Apps.Get(ctx, "")

		if err != nil {
			return nil, err
		}

		return app, nil
	}))

	must(container.Provide(func(ghs *ghsink.GitHubSink, app *github.App) (*github.User, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		list, _, err := ghs.AppsClient().Apps.ListInstallations(ctx, nil)

		if err != nil {
			return nil, err
		}

		if len(list) == 0 {
			// TODO: Fix so that client gets bot user without a token for installation
			return nil, nil
		}

		user, _, err := ghs.InstallationClient(list[0].GetID()).Users.Get(ctx, app.GetSlug()+"[bot]")

		return user, nil
	}))

	// Register app repositories
	must(container.Invoke(func(repoBundler *repository.RepositoryBundler, cfg *config.Config, ghs *ghsink.GitHubSink, app *github.App, botUser *github.User) {
		repoBundler.RegisterRepository(portal.NewPortalRepository(cfg, ghs, app, botUser))
	}))
	must(container.Invoke(func(repoBundler *repository.RepositoryBundler, cfg *config.Config, ghs *ghsink.GitHubSink, app *github.App, botUser *github.User) {
		repoBundler.RegisterRepository(mischanbot.NewMischanBotRepository(cfg, ghs, app, botUser))
	}))
	must(container.Invoke(func(repoBundler *repository.RepositoryBundler, cfg *config.Config, ghs *ghsink.GitHubSink, app *github.App, botUser *github.User) {
		repoBundler.RegisterRepository(modoki.NewModokiRepository(cfg, ghs, app, botUser))
	}))
	must(container.Invoke(func(repoBundler *repository.RepositoryBundler, cfg *config.Config, ghs *ghsink.GitHubSink, app *github.App, botUser *github.User) {
		repoBundler.RegisterRepository(birdol.NewBirdolRepository(cfg, ghs, app, botUser))
	}))

	must(container.Invoke(func(e *echo.Echo, cfg *config.Config, ghu usecase.GitHubEventUsecase) error {
		e.Use(middleware.Recover())
		e.Use(middleware.Logger())

		handler.BindHandler(e, cfg, ghu)

		if err := e.Start(fmt.Sprintf(":%d", cfg.Port)); err != nil {
			return xerrors.Errorf("failed to start handler: %w", err)
		}

		return nil
	}))
}
