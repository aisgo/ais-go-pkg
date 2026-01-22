package middleware

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aisgo/ais-go-pkg/response"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"
	redisstore "github.com/ulule/limiter/v3/drivers/store/redis"
)

const (
	defaultRateLimit  = 1000
	defaultRatePeriod = time.Second
)

// RateLimitKeyFunc returns an identifier used for rate limiting.
type RateLimitKeyFunc func(fiber.Ctx) string

var (
	rateLimiterMu      sync.RWMutex
	rateLimiter        *limiter.Limiter
	defaultLimiter     *limiter.Limiter
	defaultLimiterOnce sync.Once

	rateLimitKeyMu   sync.RWMutex
	rateLimitKeyFunc RateLimitKeyFunc
)

// SetRateLimiter replaces the global limiter and returns the previous one.
func SetRateLimiter(lim *limiter.Limiter) *limiter.Limiter {
	rateLimiterMu.Lock()
	defer rateLimiterMu.Unlock()
	prev := rateLimiter
	rateLimiter = lim
	return prev
}

// SetRateLimitKeyFunc replaces the key function and returns the previous one.
func SetRateLimitKeyFunc(fn RateLimitKeyFunc) RateLimitKeyFunc {
	rateLimitKeyMu.Lock()
	defer rateLimitKeyMu.Unlock()
	prev := rateLimitKeyFunc
	rateLimitKeyFunc = fn
	return prev
}

// InitRateLimiter initializes a redis-based limiter with default settings.
func InitRateLimiter(client *redis.Client) error {
	if client == nil {
		return nil
	}
	store, err := redisstore.NewStore(client)
	if err != nil {
		return err
	}
	lim := limiter.New(store, limiter.Rate{Period: defaultRatePeriod, Limit: defaultRateLimit})
	SetRateLimiter(lim)
	return nil
}

// RateLimitMiddleware applies request rate limiting.
func RateLimitMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		lim := currentRateLimiter()
		key := rateLimitKey(c)

		ctx, err := lim.Get(c.Context(), key)
		if err != nil {
			return response.ErrorWithCode(c, fiber.StatusInternalServerError, fmt.Errorf("rate limit check failed: %w", err))
		}

		c.Set("X-RateLimit-Limit", fmt.Sprintf("%d", ctx.Limit))
		c.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", ctx.Remaining))

		if ctx.Reached {
			return response.ErrorWithCode(c, fiber.StatusTooManyRequests, fmt.Errorf("too many requests"))
		}

		return c.Next()
	}
}

func currentRateLimiter() *limiter.Limiter {
	rateLimiterMu.RLock()
	if rateLimiter != nil {
		lim := rateLimiter
		rateLimiterMu.RUnlock()
		return lim
	}
	rateLimiterMu.RUnlock()

	defaultLimiterOnce.Do(func() {
		store := memory.NewStore()
		defaultLimiter = limiter.New(store, limiter.Rate{Period: defaultRatePeriod, Limit: defaultRateLimit})
	})

	return defaultLimiter
}

func rateLimitKey(c fiber.Ctx) string {
	rateLimitKeyMu.RLock()
	fn := rateLimitKeyFunc
	rateLimitKeyMu.RUnlock()
	if fn != nil {
		key := strings.TrimSpace(fn(c))
		if key != "" {
			return key
		}
	}
	return "ip:" + c.IP()
}
