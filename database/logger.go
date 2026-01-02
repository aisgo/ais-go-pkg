package database

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

/* ========================================================================
 * ZapGormLogger - GORM 日志适配器
 * ========================================================================
 * 职责: 将 GORM 日志适配到 Zap
 * ======================================================================== */

// ZapGormLogger 适配 GORM Logger 接口到 Zap
type ZapGormLogger struct {
	ZapLogger                 *zap.Logger
	LogLevel                  gormlogger.LogLevel
	SlowThreshold             time.Duration
	SkipCallerLookup          bool
	IgnoreRecordNotFoundError bool
}

func NewZapGormLogger(zapLogger *zap.Logger) *ZapGormLogger {
	return &ZapGormLogger{
		ZapLogger:                 zapLogger,
		LogLevel:                  gormlogger.Info,
		SlowThreshold:             200 * time.Millisecond,
		SkipCallerLookup:          false,
		IgnoreRecordNotFoundError: true,
	}
}

func (l *ZapGormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

func (l *ZapGormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Info {
		l.ZapLogger.Sugar().Infof(msg, data...)
	}
}

func (l *ZapGormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Warn {
		l.ZapLogger.Sugar().Warnf(msg, data...)
	}
}

func (l *ZapGormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Error {
		l.ZapLogger.Sugar().Errorf(msg, data...)
	}
}

func (l *ZapGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()
	fields := []zap.Field{
		zap.String("sql", sql),
		zap.Int64("rows", rows),
		zap.Duration("elapsed", elapsed),
	}

	if err != nil && (!l.IgnoreRecordNotFoundError || !errors.Is(err, gorm.ErrRecordNotFound)) {
		fields = append(fields, zap.Error(err))
		l.ZapLogger.Error("trace", fields...)
		return
	}

	if l.SlowThreshold != 0 && elapsed > l.SlowThreshold && l.LogLevel >= gormlogger.Warn {
		l.ZapLogger.Warn("slow sql", fields...)
		return
	}

	if l.LogLevel >= gormlogger.Info {
		l.ZapLogger.Debug("trace", fields...)
	}
}
