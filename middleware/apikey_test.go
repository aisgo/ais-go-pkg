package middleware

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aisgo/ais-go-pkg/logger"

	"github.com/gofiber/fiber/v3"
)

func TestAPIKeyAuthDisabled(t *testing.T) {
	app := fiber.New()
	auth := NewAPIKeyAuth(&APIKeyConfig{Enabled: false}, logger.NewNop())

	app.Use(auth.Authenticate())
	app.Get("/ping", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/ping", nil)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 2 * time.Second})
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
}

func TestAPIKeyAuthMissingKey(t *testing.T) {
	app := fiber.New()
	auth := NewAPIKeyAuth(&APIKeyConfig{Enabled: true}, logger.NewNop())

	app.Use(auth.Authenticate())
	app.Get("/ping", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/ping", nil)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 2 * time.Second})
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["msg"] != "missing api key" {
		t.Fatalf("unexpected msg: %v", body["msg"])
	}
}

func TestAPIKeyAuthValid(t *testing.T) {
	app := fiber.New()
	auth := NewAPIKeyAuth(&APIKeyConfig{
		Enabled: true,
		Keys: map[string]string{
			"client1": "sk_test_1234567890",
		},
	}, logger.NewNop())

	app.Use(auth.Authenticate())
	app.Get("/ping", func(c fiber.Ctx) error {
		id, ok := KeyIDFromContext(c)
		if !ok || id != "client1" {
			return c.Status(fiber.StatusUnauthorized).SendString("bad")
		}
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/ping", nil)
	req.Header.Set("X-API-Key", "sk_test_1234567890")
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 2 * time.Second})
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
}

func TestAPIKeyAuthBearer(t *testing.T) {
	app := fiber.New()
	auth := NewAPIKeyAuth(&APIKeyConfig{
		Enabled: true,
		Keys: map[string]string{
			"client1": "sk_test_bearer",
		},
	}, logger.NewNop())

	app.Use(auth.Authenticate())
	app.Get("/ping", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/ping", nil)
	req.Header.Set("Authorization", "Bearer sk_test_bearer")
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 2 * time.Second})
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
}
