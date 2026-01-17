//go:build integration

package mysql

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/aisgo/ais-go-pkg/logger"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
	"go.uber.org/fx"
)

type testLifecycle struct {
	hooks []fx.Hook
}

func (l *testLifecycle) Append(h fx.Hook) {
	l.hooks = append(l.hooks, h)
}

func (l *testLifecycle) stop(ctx context.Context) {
	for _, h := range l.hooks {
		if h.OnStop != nil {
			_ = h.OnStop(ctx)
		}
	}
}

func TestNewDBIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skip integration test in short mode")
	}
	testcontainers.SkipIfProviderIsNotHealthy(t)

	ctx := context.Background()

	container, err := mysql.Run(ctx,
		"mysql:8.0",
		mysql.WithDatabase("testdb"),
		mysql.WithUsername("test"),
		mysql.WithPassword("testpass"),
	)
	if err != nil {
		t.Fatalf("start mysql container: %v", err)
	}
	t.Cleanup(func() {
		_ = container.Terminate(ctx)
	})

	dsn, err := container.ConnectionString(ctx, "charset=utf8mb4&parseTime=true&loc=Local")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	cfg, err := mysqldriver.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("parse dsn: %v", err)
	}

	host, portStr, err := net.SplitHostPort(cfg.Addr)
	if err != nil {
		t.Fatalf("split host port: %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("parse port: %v", err)
	}

	lc := &testLifecycle{}
	db, err := NewDB(Params{
		Lc: lc,
		Config: Config{
			Host:            host,
			Port:            port,
			User:            cfg.User,
			Password:        cfg.Passwd,
			DBName:          cfg.DBName,
			Charset:         "utf8mb4",
			Loc:             "Local",
			MaxIdleConns:    2,
			MaxOpenConns:    4,
			ConnMaxLifetime: time.Minute,
			ConnMaxIdleTime: 30 * time.Second,
		},
		Logger: logger.NewNop(),
	})
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}

	var one int
	if err := db.Raw("SELECT 1").Scan(&one).Error; err != nil {
		t.Fatalf("select 1: %v", err)
	}
	if one != 1 {
		t.Fatalf("unexpected result: %d", one)
	}

	lc.stop(context.Background())
}
