package handler

import (
	"encoding/json"
	"net/http"

	"github.com/MISW/mischan-bot/config"
	"github.com/MISW/mischan-bot/usecase"
	"github.com/google/go-github/v49/github"
	"github.com/labstack/echo/v4"
)

// RootHandler is a echo handler
type RootHandler interface {
	Webhook(c echo.Context) error
}

type rootHandler struct {
	webhookSecret      string
	githubEventUsecase usecase.GitHubEventUsecase
}

// BindHandler binds root handler for Echo
func BindHandler(e *echo.Echo, cfg *config.Config, ghu usecase.GitHubEventUsecase) {
	rh := &rootHandler{
		webhookSecret:      cfg.WebhookSecret,
		githubEventUsecase: ghu,
	}

	e.POST("/webhook", rh.Webhook)
}

var _ RootHandler = &rootHandler{}

func (rh *rootHandler) Webhook(c echo.Context) error {
	payload, err := github.ValidatePayload(c.Request(), []byte(rh.webhookSecret))

	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"message": "token is invalid"})
	}

	switch c.Request().Header.Get("X-GitHub-Event") {
	case "push":
		event := &github.PushEvent{}
		if err := json.Unmarshal(payload, event); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"message": "payload is invalid json", "error": err.Error()})
		}

		if err := rh.githubEventUsecase.Push(event); err != nil {
			return err
		}
	case "check_suite":
		event := &github.CheckSuiteEvent{}
		if err := json.Unmarshal(payload, event); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"message": "payload is invalid json", "error": err.Error()})
		}

		if err := rh.githubEventUsecase.CheckSuite(event); err != nil {
			return err
		}
	case "create":
		event := &github.CreateEvent{}
		if err := json.Unmarshal(payload, event); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"message": "payload is invalid json", "error": err.Error()})
		}

		if err := rh.githubEventUsecase.Create(event); err != nil {
			return err
		}
	}

	return nil
}
