package metrics

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
)

func TestRegisterMetricsEndpoint(t *testing.T) {
	counter := NewCounter("test", "unit", "total", "unit test counter", []string{"k"})
	counter.WithLabelValues("v").Inc()

	app := fiber.New()
	RegisterMetricsEndpoint(app)

	req := httptest.NewRequest("GET", "/metrics", nil)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 2 * time.Second})
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !strings.Contains(string(body), "test_unit_total") {
		t.Fatalf("expected metrics output to include test_unit_total")
	}
}
