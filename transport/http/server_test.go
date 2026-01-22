package http

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
)

func TestBuildListenConfigDefaults(t *testing.T) {
	cfg := buildListenConfig(ListenOptions{})
	if cfg.ListenerNetwork != "tcp4" {
		t.Fatalf("unexpected listener network: %s", cfg.ListenerNetwork)
	}
}

func TestBuildListenConfigOverrides(t *testing.T) {
	cfg := buildListenConfig(ListenOptions{
		EnablePrefork:         true,
		DisableStartupMessage: true,
		EnablePrintRoutes:     true,
		ListenerNetwork:       "tcp6",
		ShutdownTimeout:       2 * time.Second,
		UnixSocketFileMode:    0771,
		TLSMinVersion:         772,
	})
	if !cfg.EnablePrefork || !cfg.DisableStartupMessage || !cfg.EnablePrintRoutes {
		t.Fatalf("unexpected boolean settings")
	}
	if cfg.ListenerNetwork != "tcp6" {
		t.Fatalf("unexpected listener network: %s", cfg.ListenerNetwork)
	}
	if cfg.ShutdownTimeout != 2*time.Second {
		t.Fatalf("unexpected shutdown timeout: %v", cfg.ShutdownTimeout)
	}
	if cfg.UnixSocketFileMode == 0 {
		t.Fatalf("expected unix socket file mode to be set")
	}
	if cfg.TLSMinVersion != 772 {
		t.Fatalf("unexpected tls min version: %d", cfg.TLSMinVersion)
	}
}

func TestHealthEndpoints(t *testing.T) {
	app := fiber.New()
	registerHealthEndpoints(app, nil, 2*time.Second)

	req := httptest.NewRequest("GET", "/healthz", nil)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 2 * time.Second})
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}

	req = httptest.NewRequest("GET", "/readyz", nil)
	resp, err = app.Test(req, fiber.TestConfig{Timeout: 2 * time.Second})
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("unexpected status body: %v", body["status"])
	}
}
