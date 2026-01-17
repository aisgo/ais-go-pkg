package postgres

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/aisgo/ais-go-pkg/database"
	"github.com/aisgo/ais-go-pkg/logger"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

/* ========================================================================
 * PostgreSQL - 关系型数据库连接
 * ========================================================================
 * 职责: 提供 PostgreSQL 连接池、GORM 集成
 * 技术: gorm.io/driver/postgres
 * ======================================================================== */

// 默认连接池配置
const (
	DefaultMaxIdleConns    = 10
	DefaultMaxOpenConns    = 25
	DefaultConnMaxLifetime = 1 * time.Hour
	DefaultConnMaxIdleTime = 20 * time.Minute
)

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

// Params 依赖注入参数
type Params struct {
	fx.In
	Lc     fx.Lifecycle
	Config Config
	Logger *logger.Logger
}

// NewDB 初始化 Postgres 连接
func NewDB(p Params) (*gorm.DB, error) {
	log := p.Logger
	if log == nil {
		log = logger.NewNop()
	}
	sslMode := p.Config.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}

	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(p.Config.User, p.Config.Password),
		Host:   fmt.Sprintf("%s:%d", p.Config.Host, p.Config.Port),
		Path:   p.Config.DBName,
	}
	q := u.Query()
	q.Set("sslmode", sslMode)
	if p.Config.Schema != "" {
		q.Set("search_path", p.Config.Schema)
	}
	u.RawQuery = q.Encode()
	dsn := u.String()
	log.Info("Connecting to PostgreSQL", zap.String("dsn", sanitizeDSN(dsn)))

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
	maxIdleConns := p.Config.MaxIdleConns
	if maxIdleConns <= 0 {
		maxIdleConns = DefaultMaxIdleConns
	}

	maxOpenConns := p.Config.MaxOpenConns
	if maxOpenConns <= 0 {
		maxOpenConns = DefaultMaxOpenConns
	}

	connMaxLifetime := p.Config.ConnMaxLifetime
	if connMaxLifetime <= 0 {
		connMaxLifetime = DefaultConnMaxLifetime
	}

	connMaxIdleTime := p.Config.ConnMaxIdleTime
	if connMaxIdleTime <= 0 {
		connMaxIdleTime = DefaultConnMaxIdleTime
	}

	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)

	// 注册生命周期钩子
	p.Lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			log.Info("Closing PostgreSQL connection pool", zap.String("db", p.Config.DBName))
			return sqlDB.Close()
		},
	})

	return db, nil
}

func sanitizeDSN(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil || u == nil {
		return dsn
	}
	if u.User != nil {
		u.User = url.UserPassword(u.User.Username(), "***")
	}
	return u.String()
}
