package config

import (
	"fmt"

	"github.com/caarlos0/env/v9"
	"golang.org/x/xerrors"
)

// PrivateKey represents a private key
type PrivateKey struct {
	Path string `env:"PRIVATE_KEY_PATH"`
	Raw  string `env:"PRIVATE_KEY"`
}

// Config represents a config to load from file
type Config struct {
	WebhookSecret string `env:"WEBHOOK_SECRET"`
	AppID         int64  `env:"APP_ID"`
	ManifestRepo  string `env:"MANIFEST_REPO"`
	Port          int    `env:"PORT"`

	PrivateKey PrivateKey
}

// ReadConfig reads config from env, json and yaml
func ReadConfig() (*Config, error) {
	var cfg Config

	err := env.Parse(&cfg)
	if err != nil {
		return nil, xerrors.Errorf("failed to perse config: %w", err)
	}

	err = env.Parse(&cfg.PrivateKey)
	if err != nil {
		return nil, xerrors.Errorf("failed to perse config: %w", err)
	}

	fmt.Println(cfg)

	return &cfg, err
}
