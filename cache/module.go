package cache

import (
	"github.com/aisgo/ais-go-pkg/cache/redis"
	"go.uber.org/fx"
)

/* ========================================================================
 * Cache Module
 * ========================================================================
 * 职责: 提供 Redis 缓存依赖注入模块
 * ======================================================================== */

// Module 缓存模块
// 提供: redis.Clienter, *redis.Client
var Module = fx.Module("cache",
	fx.Provide(
		redis.NewClient,
		func(c *redis.Client) redis.Clienter { return c },
	),
)
