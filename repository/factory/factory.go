// Package factory builds the repositories for the configured database.
package factory

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm/logger"

	"github.com/xXMateus97Xx/email-pixel-tracker/repository"
	"github.com/xXMateus97Xx/email-pixel-tracker/repository/mongorepo"
	"github.com/xXMateus97Xx/email-pixel-tracker/repository/sqlrepo"
	"github.com/xXMateus97Xx/email-pixel-tracker/telemetry"
)

// slowQueryThreshold is the duration above which a SQL query is logged as a warning.
const slowQueryThreshold = 200 * time.Millisecond

const DriverMongoDB = "mongodb"

type Config struct {
	// Driver is one of: postgres, sqlserver, mysql, mongodb.
	Driver string
	// DSN is the connection string (for MongoDB, the connection URI).
	DSN string
	// Database is the MongoDB database name. Ignored by the SQL drivers.
	Database string
	// Logger receives database errors and slow queries. Optional.
	Logger *slog.Logger
	// Metrics records the duration of database operations. Optional.
	Metrics *telemetry.Metrics
}

// New connects to the configured database and always creates/updates the SQL
// schema or the MongoDB indexes.
func New(ctx context.Context, cfg Config) (*repository.Repositories, error) {
	switch cfg.Driver {
	case sqlrepo.DriverPostgres, sqlrepo.DriverSqlServer, sqlrepo.DriverMySql:
		var log logger.Interface
		if cfg.Logger != nil {
			log = logger.NewSlogLogger(cfg.Logger, logger.Config{
				LogLevel:                  logger.Warn,
				SlowThreshold:             slowQueryThreshold,
				IgnoreRecordNotFoundError: true,
				// Values stay out of the logs: they include password hashes and personal data.
				ParameterizedQueries: true,
			})
		}
		db, err := sqlrepo.Open(cfg.Driver, cfg.DSN, log, telemetry.GormPlugin(cfg.Metrics, cfg.Driver))
		if err != nil {
			return nil, fmt.Errorf("connecting to %s: %w", cfg.Driver, err)
		}
		repos := sqlrepo.New(db)
		if sqlDB, err := db.DB(); err == nil {
			if err := cfg.Metrics.RegisterSQLStats(sqlDB, cfg.Driver); err != nil {
				_ = repos.Close(ctx)
				return nil, err
			}
		}
		if err := sqlrepo.Migrate(db); err != nil {
			_ = repos.Close(ctx)
			return nil, fmt.Errorf("migrating %s: %w", cfg.Driver, err)
		}
		return repos, nil

	case DriverMongoDB:
		if cfg.Database == "" {
			return nil, errors.New("mongodb: database name is required")
		}
		client, err := mongorepo.Open(ctx, cfg.DSN, telemetry.MongoMonitor(cfg.Metrics))
		if err != nil {
			return nil, fmt.Errorf("connecting to mongodb: %w", err)
		}
		db := client.Database(cfg.Database)
		repos := mongorepo.New(client, db)
		if err := mongorepo.EnsureIndexes(ctx, db); err != nil {
			_ = repos.Close(ctx)
			return nil, fmt.Errorf("creating mongodb indexes: %w", err)
		}
		return repos, nil

	default:
		return nil, fmt.Errorf("unsupported database driver %q", cfg.Driver)
	}
}
