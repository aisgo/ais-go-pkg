package mysql

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/aisgo/ais-go-pkg/database"
	"github.com/aisgo/ais-go-pkg/logger"

	mysqldriver "github.com/go-sql-driver/mysql"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

/* ========================================================================
 * MySQL - 关系型数据库连接
 * ========================================================================
 * 职责: 提供 MySQL 连接池、GORM 集成
 * 技术: gorm.io/driver/mysql
 * ======================================================================== */

// 默认连接池配置
const (
	DefaultMaxIdleConns    = 10
	DefaultMaxOpenConns    = 25
	DefaultConnMaxLifetime = 1 * time.Hour
	DefaultConnMaxIdleTime = 20 * time.Minute
	DefaultCharset         = "utf8mb4"
	DefaultLoc             = "Local"
)

// Config MySQL 配置
type Config struct {
	Host             string        `yaml:"host"`
	Port             int           `yaml:"port"`
	User             string        `yaml:"user"`
	Password         string        `yaml:"password"`
	DBName           string        `yaml:"dbname"`
	Charset          string        `yaml:"charset"`            // 字符集，默认 utf8mb4
	ParseTime        bool          `yaml:"parse_time"`         // 是否解析时间类型，默认 true
	DisableParseTime bool          `yaml:"disable_parse_time"` // 显式关闭 parseTime（bool 无法区分未设置/false）
	Loc              string        `yaml:"loc"`                // 时区，默认 Local
	MaxIdleConns     int           `yaml:"max_idle_conns"`     // 最大空闲连接数
	MaxOpenConns     int           `yaml:"max_open_conns"`     // 最大打开连接数
	ConnMaxLifetime  time.Duration `yaml:"conn_max_lifetime"`  // 连接最大生命周期
	ConnMaxIdleTime  time.Duration `yaml:"conn_max_idle_time"` // 空闲连接最大时间
}

// Params 依赖注入参数
type Params struct {
	fx.In
	Lc     fx.Lifecycle
	Config Config
	Logger *logger.Logger
}

// NewDB 初始化 MySQL 连接
func NewDB(p Params) (*gorm.DB, error) {
	log := p.Logger
	if log == nil {
		log = logger.NewNop()
	}
	// 设置默认值
	charset := p.Config.Charset
	if charset == "" {
		charset = DefaultCharset
	}

	parseTime := true
	if p.Config.DisableParseTime {
		parseTime = false
	}

	loc := p.Config.Loc
	if loc == "" {
		loc = DefaultLoc
	}

	driverCfg := mysqldriver.Config{
		User:   p.Config.User,
		Passwd: p.Config.Password,
		Net:    "tcp",
		Addr:   fmt.Sprintf("%s:%d", p.Config.Host, p.Config.Port),
		DBName: p.Config.DBName,
		Params: map[string]string{
			"charset":   charset,
			"parseTime": strconv.FormatBool(parseTime),
			"loc":       loc,
		},
	}
	dsn := driverCfg.FormatDSN()

	// 使用自定义的 ZapGormLogger
	gormLog := database.NewZapGormLogger(log.Logger)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
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
		if sqlDB != nil {
			_ = sqlDB.Close()
		} else if closer, ok := db.ConnPool.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
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
		OnStart: func(ctx context.Context) error {
			if err := sqlDB.PingContext(ctx); err != nil {
				log.Error("MySQL connection failed", zap.Error(err))
				return err
			}
			log.Info("MySQL connected", zap.String("db", p.Config.DBName))
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Info("Closing MySQL connection pool", zap.String("db", p.Config.DBName))
			return sqlDB.Close()
		},
	})

	return db, nil
}
