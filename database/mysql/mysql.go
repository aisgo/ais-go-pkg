package mysql

import (
	"fmt"
	"time"

	"github.com/aisgo/ais-go-pkg/database"
	"github.com/aisgo/ais-go-pkg/logger"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

/* ========================================================================
 * MySQL - 关系型数据库连接
 * ========================================================================
 * 职责: 提供 MySQL 连接池、GORM 集成
 * 技术: gorm.io/driver/mysql
 * ======================================================================== */

// Config MySQL 配置
type Config struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	User            string        `yaml:"user"`
	Password        string        `yaml:"password"`
	DBName          string        `yaml:"dbname"`
	Charset         string        `yaml:"charset"`            // 字符集，默认 utf8mb4
	ParseTime       bool          `yaml:"parse_time"`         // 是否解析时间类型，默认 true
	Loc             string        `yaml:"loc"`                // 时区，默认 Local
	MaxIdleConns    int           `yaml:"max_idle_conns"`     // 最大空闲连接数
	MaxOpenConns    int           `yaml:"max_open_conns"`     // 最大打开连接数
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`  // 连接最大生命周期
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time"` // 空闲连接最大时间
}

// NewDB 初始化 MySQL 连接
func NewDB(cfg Config, log *logger.Logger) (*gorm.DB, error) {
	// 设置默认值
	charset := cfg.Charset
	if charset == "" {
		charset = "utf8mb4"
	}

	parseTime := cfg.ParseTime
	if !parseTime {
		parseTime = true
	}

	loc := cfg.Loc
	if loc == "" {
		loc = "Local"
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName, charset, parseTime, loc)

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
