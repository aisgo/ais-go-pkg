package shutdown

import "time"

/* ========================================================================
 * Shutdown Config - 优雅关停配置
 * ========================================================================
 * 职责: 定义优雅关停的配置结构
 * ======================================================================== */

// Config 优雅关停配置
type Config struct {
	// Timeout 关停超时时间
	// 超时后将强制退出，即使有钩子未执行完成
	Timeout time.Duration `yaml:"timeout"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Timeout: 30 * time.Second,
	}
}
