package main

import (
	"fmt"
	"log/slog"

	"github.com/quonaro/gnostis/internal/config"
)

func loadConfig() (config.Config, error) {
	cfg, err := config.FromEnv()
	if err != nil {
		return config.Config{}, fmt.Errorf("load config: %w", err)
	}

	if cfg.LogLevel != "" {
		level, err := parseLogLevel(cfg.LogLevel)
		if err != nil {
			return config.Config{}, fmt.Errorf("parse log level: %w", err)
		}
		slog.SetDefault(slog.New(newLogHandler(logOutput, level)))
	}

	return cfg, nil
}
