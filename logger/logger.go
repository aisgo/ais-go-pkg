package logger

import (
	"context"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

/* ========================================================================
 * Logger - 统一日志组件
 * ========================================================================
 * 职责: 提供结构化日志能力，支持 JSON / Console 格式
 * 技术: Uber Zap
 * ======================================================================== */

// Config Logger 配置
type Config struct {
	Level  string `yaml:"level"`  // debug, info, warn, error
	Format string `yaml:"format"` // json, console
	Output string `yaml:"output"` // stdout, file
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
		_ = level.UnmarshalText([]byte(cfg.Level))
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

	// 配置输出 (简化实现，始终输出到 stdout)
	var writer zapcore.WriteSyncer
	writer = zapcore.AddSync(os.Stdout)

	core := zapcore.NewCore(
		encoder,
		writer,
		level,
	)

	logger := zap.New(core, zap.AddCaller())
	return &Logger{Logger: logger}
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
