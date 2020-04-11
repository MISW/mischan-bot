package main

import (
	"log"
	"os"

	"github.com/MISW/mischan-bot/config"
	"github.com/MISW/mischan-bot/handler"
	"github.com/MISW/mischan-bot/intenral/ghsink"
	"github.com/MISW/mischan-bot/repository"
	"github.com/MISW/mischan-bot/repository/portal"
	"github.com/MISW/mischan-bot/usecase"
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

	// Register app repositories
	must(container.Invoke(func(repoBundler *repository.RepositoryBundler, cfg *config.Config, ghs *ghsink.GitHubSink) {
		repoBundler.RegisterRepository(portal.NewPortalRepository(cfg, ghs))
	}))

	must(container.Invoke(func(e *echo.Echo, cfg *config.Config, ghu usecase.GitHubEventUsecase) error {
		e.Use(middleware.Logger())

		handler.BindHandler(e, cfg, ghu)

		if err := e.Start(":" + os.Getenv("PORT")); err != nil {
			return xerrors.Errorf("failed to start handler: %w", err)
		}

		return nil
	}))
}
