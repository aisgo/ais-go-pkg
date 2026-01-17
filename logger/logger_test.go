package logger

import (
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

func TestValidateConfig(t *testing.T) {
	if err := ValidateConfig(Config{Level: "info", Format: "json"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ValidateConfig(Config{Level: "bad"}); err == nil {
		t.Fatalf("expected error for invalid level")
	}
	if err := ValidateConfig(Config{Format: "xml"}); err == nil {
		t.Fatalf("expected error for invalid format")
	}
}

func TestNewLoggerLevel(t *testing.T) {
	log := NewLogger(Config{Level: "debug"})
	if !log.Core().Enabled(zap.DebugLevel) {
		t.Fatalf("expected debug enabled")
	}

	log = NewLogger(Config{Level: "info"})
	if log.Core().Enabled(zap.DebugLevel) {
		t.Fatalf("expected debug disabled")
	}

	log = NewLogger(Config{Level: "not-a-level"})
	if log.Core().Enabled(zap.DebugLevel) {
		t.Fatalf("expected debug disabled on invalid level")
	}
}

func TestNewLoggerFileOutput(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")

	log := NewLogger(Config{Level: "info", Format: "json", Output: path})
	log.Info("hello", zap.String("k", "v"))
	_ = log.Sync()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat log file: %v", err)
	}
	if info.Size() == 0 {
		t.Fatalf("expected log file not empty")
	}
}
