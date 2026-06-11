// Package postgres implements ports.DB using GORM + PostgreSQL.
// To swap to MySQL: implement ports.DB using gorm.io/driver/mysql and wire in main.go.
package postgres

import (
	"context"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// DB wraps *gorm.DB and implements ports.DB.
type DB struct {
	db *gorm.DB
}

// Ensure compile-time interface satisfaction.
var _ ports.DB = (*DB)(nil)

// Config holds PostgreSQL connection options.
type Config struct {
	DSN         string
	MaxOpenConn int
	MaxIdleConn int
	LogLevel    logger.LogLevel
}

// New connects to PostgreSQL and returns a ports.DB implementation.
func New(cfg Config) (*DB, error) {
	if cfg.MaxOpenConn == 0 {
		cfg.MaxOpenConn = 25
	}
	if cfg.MaxIdleConn == 0 {
		cfg.MaxIdleConn = 10
	}
	if cfg.LogLevel == 0 {
		cfg.LogLevel = logger.Warn
	}

	gormDB, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(cfg.LogLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("postgres: open: %w", err)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("postgres: get sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConn)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConn)

	return &DB{db: gormDB}, nil
}

func (d *DB) Conn() *gorm.DB { return d.db }

func (d *DB) Ping(ctx context.Context) error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

func (d *DB) Migrate(models ...any) error {
	return d.db.AutoMigrate(models...)
}

func (d *DB) Close() error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
