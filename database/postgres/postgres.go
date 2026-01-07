package postgres

import (
	"fmt"
	"time"

	"github.com/aisgo/ais-go-pkg/database"
	"github.com/aisgo/ais-go-pkg/logger"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

/* ========================================================================
 * PostgreSQL - 关系型数据库连接
 * ========================================================================
 * 职责: 提供 PostgreSQL 连接池、GORM 集成
 * 技术: gorm.io/driver/postgres
 * ======================================================================== */

// Config PostgreSQL 配置
type Config struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	User            string        `yaml:"user"`
	Password        string        `yaml:"password"`
	DBName          string        `yaml:"dbname"`
	SSLMode         string        `yaml:"sslmode"`
	Schema          string        `yaml:"schema"`             // 数据库 schema，默认 public
	MaxIdleConns    int           `yaml:"max_idle_conns"`     // 最大空闲连接数
	MaxOpenConns    int           `yaml:"max_open_conns"`     // 最大打开连接数
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`  // 连接最大生命周期
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time"` // 空闲连接最大时间
}

// NewDB 初始化 Postgres 连接
func NewDB(cfg Config, log *logger.Logger) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)

	// 如果配置了 schema，添加到 DSN
	if cfg.Schema != "" {
		dsn = fmt.Sprintf("%s search_path=%s", dsn, cfg.Schema)
	}

	// 使用自定义的 ZapGormLogger
	gormLog := database.NewZapGormLogger(log.Logger)

	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN: dsn,
	}), &gorm.Config{
		Logger: gormLog,
		NowFunc: func() time.Time {
			return time.Now().Local()
		},
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// 连接池配置（应用默认值）
	maxIdleConns := cfg.MaxIdleConns
	if maxIdleConns <= 0 {
		maxIdleConns = 10
	}

	maxOpenConns := cfg.MaxOpenConns
	if maxOpenConns <= 0 {
		maxOpenConns = 25
	}

	connMaxLifetime := cfg.ConnMaxLifetime
	if connMaxLifetime <= 0 {
		connMaxLifetime = 1 * time.Hour
	}

	connMaxIdleTime := cfg.ConnMaxIdleTime
	if connMaxIdleTime <= 0 {
		connMaxIdleTime = 20 * time.Minute
	}

	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)

	return db, nil
}
