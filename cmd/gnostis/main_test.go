package main

import (
	"log/slog"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Run("valid env", func(t *testing.T) {
		t.Setenv("GS_LOG_LEVEL", "debug")
		t.Setenv("GS_DATA_DIR", t.TempDir())
		t.Setenv("GS_WEB_PORT", "0")
		cfg, err := loadConfig()
		if err != nil {
			t.Fatalf("loadConfig: %v", err)
		}
		if cfg.LogLevel != "debug" {
			t.Errorf("LogLevel = %q", cfg.LogLevel)
		}
	})

	t.Run("invalid log level", func(t *testing.T) {
		t.Setenv("GS_LOG_LEVEL", "bogus")
		if _, err := loadConfig(); err == nil {
			t.Fatal("expected error for invalid log level")
		}
	})
}

func TestParseLogLevel(t *testing.T) {
	cases := []struct {
		in      string
		want    slog.Level
		wantErr bool
	}{
		{"debug", slog.LevelDebug, false},
		{"info", slog.LevelInfo, false},
		{"warn", slog.LevelWarn, false},
		{"error", slog.LevelError, false},
		{"", slog.LevelInfo, true},
		{"verbose", slog.LevelInfo, true},
		{"DEBUG", slog.LevelInfo, true}, // case-sensitive
	}
	for _, c := range cases {
		got, err := parseLogLevel(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("parseLogLevel(%q) expected error", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseLogLevel(%q) unexpected error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("parseLogLevel(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
