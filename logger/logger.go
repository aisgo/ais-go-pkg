package logger

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

/* ========================================================================
 * Logger - 统一日志组件
 * ========================================================================
 * 职责: 提供结构化日志能力，支持 JSON / Console 格式
 * 技术: Uber Zap
 * ======================================================================== */

// Config Logger 配置
type Config struct {
	Level      string `yaml:"level"`  // debug, info, warn, error
	Format     string `yaml:"format"` // json, console
	Output     string `yaml:"output"` // stdout, file
	MaxSize    int    `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"`
	Compress   *bool  `yaml:"compress"`
}

// Logger 封装 Zap Logger
type Logger struct {
	*zap.Logger
}

// NewLogger 初始化 Logger
func NewLogger(cfg Config) *Logger {
	// 解析日志级别
	level := zap.InfoLevel
	if cfg.Level != "" {
		if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
			// 使用 stderr 输出警告（此时 logger 还未初始化）
			fmt.Fprintf(os.Stderr,
				"[WARN] Invalid log level %q, using INFO as default: %v\n",
				cfg.Level, err)
		}
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	// 根据格式选择编码器
	var encoder zapcore.Encoder
	if cfg.Format == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// 配置输出
	writer := zapcore.AddSync(os.Stdout)
	if cfg.Output != "" && cfg.Output != "stdout" {
		maxSize := cfg.MaxSize
		if maxSize <= 0 {
			maxSize = 100
		}
		maxBackups := cfg.MaxBackups
		if maxBackups <= 0 {
			maxBackups = 3
		}
		maxAge := cfg.MaxAge
		if maxAge <= 0 {
			maxAge = 28
		}
		compress := true
		if cfg.Compress != nil {
			compress = *cfg.Compress
		}

		writer = zapcore.AddSync(&lumberjack.Logger{
			Filename:   cfg.Output,
			MaxSize:    maxSize, // MB
			MaxBackups: maxBackups,
			MaxAge:     maxAge, // days
			Compress:   compress,
		})
	}

	core := zapcore.NewCore(
		encoder,
		writer,
		level,
	)

	logger := zap.New(core, zap.AddCaller())
	return &Logger{Logger: logger}
}

// ValidateConfig 验证配置（可在初始化前调用）
func ValidateConfig(cfg Config) error {
	// 验证日志级别
	if cfg.Level != "" {
		var level zapcore.Level
		if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
			return fmt.Errorf("invalid log level %q: %w", cfg.Level, err)
		}
	}

	// 验证格式
	if cfg.Format != "" && cfg.Format != "json" && cfg.Format != "console" {
		return fmt.Errorf("invalid log format %q, must be 'json' or 'console'", cfg.Format)
	}
	if cfg.MaxSize < 0 {
		return fmt.Errorf("invalid max_size %d, must be >= 0", cfg.MaxSize)
	}
	if cfg.MaxBackups < 0 {
		return fmt.Errorf("invalid max_backups %d, must be >= 0", cfg.MaxBackups)
	}
	if cfg.MaxAge < 0 {
		return fmt.Errorf("invalid max_age %d, must be >= 0", cfg.MaxAge)
	}

	return nil
}

// NewNop 返回一个 no-op Logger（用于可选依赖/测试）
func NewNop() *Logger {
	return &Logger{Logger: zap.NewNop()}
}

// WithContext 从 Context 提取 TraceID (后续实现) 并注入 Logger
func (l *Logger) WithContext(ctx context.Context) *zap.Logger {
	// 占位: 后续集成 TraceID
	// traceID := ctx.Value("trace_id")
	// if traceID != nil {
	// 	return l.Logger.With(zap.Any("trace_id", traceID))
	// }
	return l.Logger
}
