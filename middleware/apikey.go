package middleware

import (
	"crypto/subtle"
	"strings"

	"ais.local/ais-go-pkg/logger"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

/* ========================================================================
 * API Key Authentication Middleware
 * ========================================================================
 * 职责: 验证 API Key 请求
 * 支持两种方式:
 *   1. X-API-Key Header
 *   2. Authorization Bearer Token
 * ======================================================================== */

// APIKeyConfig API Key 配置
type APIKeyConfig struct {
	Enabled bool              `yaml:"enabled"`
	Keys    map[string]string `yaml:"keys"` // key_id -> api_key
}

// APIKeyAuth API Key 认证中间件
type APIKeyAuth struct {
	config *APIKeyConfig
	log    *logger.Logger
}

// NewAPIKeyAuth 创建 API Key 认证中间件
func NewAPIKeyAuth(cfg *APIKeyConfig, log *logger.Logger) *APIKeyAuth {
	return &APIKeyAuth{
		config: cfg,
		log:    log,
	}
}

// Authenticate 返回 Fiber 中间件
func (a *APIKeyAuth) Authenticate() fiber.Handler {
	return func(c fiber.Ctx) error {
		// 如果未启用认证，直接放行
		if !a.config.Enabled {
			return c.Next()
		}

		// 从 X-API-Key Header 获取
		apiKey := c.Get("X-API-Key")
		if apiKey == "" {
			// 尝试从 Authorization Bearer 获取
			auth := c.Get("Authorization")
			if strings.HasPrefix(auth, "Bearer ") {
				apiKey = strings.TrimPrefix(auth, "Bearer ")
			}
		}

		if apiKey == "" {
			a.log.Warn("Missing API Key",
				zap.String("ip", c.IP()),
				zap.String("path", c.Path()),
			)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code": 401,
				"msg":  "missing api key",
			})
		}

		// 验证 API Key (constant-time 比较防止时序攻击)
		keyID, valid := a.validateAPIKey(apiKey)
		if !valid {
			a.log.Warn("Invalid API Key",
				zap.String("ip", c.IP()),
				zap.String("path", c.Path()),
			)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code": 401,
				"msg":  "invalid api key",
			})
		}

		// 将 key_id 存储到 context，用于后续的 tenant_id 映射
		c.Locals("key_id", keyID)

		return c.Next()
	}
}

// validateAPIKey 验证 API Key
// 使用 constant-time 比较防止时序攻击
func (a *APIKeyAuth) validateAPIKey(apiKey string) (string, bool) {
	for keyID, storedKey := range a.config.Keys {
		if subtle.ConstantTimeCompare([]byte(apiKey), []byte(storedKey)) == 1 {
			return keyID, true
		}
	}
	return "", false
}
