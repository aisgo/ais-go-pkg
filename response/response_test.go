package response

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	aiserrors "github.com/aisgo/ais-go-pkg/errors"
	"github.com/gofiber/fiber/v3"
)

func TestError_BizError(t *testing.T) {
	t.Parallel()

	app := fiber.New()
	app.Get("/err", func(c fiber.Ctx) error {
		return Error(c, aiserrors.New(aiserrors.ErrCodeInvalidArgument, "bad request"))
	})

	req := httptest.NewRequest("GET", "/err", nil)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 2 * time.Second})
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("unexpected status: got=%d want=%d", resp.StatusCode, fiber.StatusBadRequest)
	}

	var got Result
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Code != int(aiserrors.ErrCodeInvalidArgument) {
		t.Fatalf("unexpected code: got=%d want=%d", got.Code, int(aiserrors.ErrCodeInvalidArgument))
	}
	if got.Msg != "bad request" {
		t.Fatalf("unexpected msg: got=%q want=%q", got.Msg, "bad request")
	}
}
