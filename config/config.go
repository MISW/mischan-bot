package config

import (
	"context"

	"github.com/heetch/confita"
	"github.com/heetch/confita/backend/env"
	"github.com/heetch/confita/backend/file"
	"golang.org/x/xerrors"
)

// PrivateKey represents a private key
type PrivateKey struct {
	Path string `config:"private_key_path" json:"path" yaml:"path"`
	Raw  string `config:"private_key" json:"raw" yaml:"raw"`
}

// Config represents a config to load from file
type Config struct {
	WebhookSecret string     `config:"webhook_secret" json:"webhook_secret" yaml:"webhook_secret"`
	PrivateKey    PrivateKey `json:"private_key" yaml:"private_key"`
	AppID         int64      `config:"app_id" json:"app_id" yaml:"app_id"`
	ManifestRepo  string     `config:"manifest_repo" json:"manifest_repo" yaml:"manifest_repo"`
}

// ReadConfig reads config from env, json and yaml
func ReadConfig() (*Config, error) {
	loader := confita.NewLoader(
		env.NewBackend(),
		file.NewBackend("./mischan.json"),
		file.NewBackend("./mischan.yaml"),
	)

	cfg := &Config{}

	err := loader.Load(context.Background(), cfg)

	if err != nil {
		return nil, xerrors.Errorf("failed to load config: %w", err)
	}

	return cfg, nil
}
