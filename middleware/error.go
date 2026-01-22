package middleware

import (
	"github.com/aisgo/ais-go-pkg/logger"
	"github.com/aisgo/ais-go-pkg/response"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

// NewErrorHandler returns a Fiber ErrorHandler with unified logging and response formatting.
func NewErrorHandler(log *logger.Logger) fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		if err == nil {
			return nil
		}

		if log != nil {
			log.Error("unhandled error", zap.Error(err))
		}
		return response.Error(c, err)
	}
}
