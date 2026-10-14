// Package sqlrepo implements the repositories on top of GORM, supporting
// PostgreSQL, SQL Server and MySQL.
package sqlrepo

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/xXMateus97Xx/email-pixel-tracker/repository"
)

const (
	DriverPostgres  = "postgres"
	DriverSqlServer = "sqlserver"
	DriverMySql     = "mysql"
)

// Open connects to the database. For MySQL the DSN must contain parseTime=true.
// A nil log uses GORM's default logger; plugins add instrumentation.
func Open(driver, dsn string, log logger.Interface, plugins ...gorm.Plugin) (*gorm.DB, error) {
	var dialector gorm.Dialector
	switch driver {
	case DriverPostgres:
		dialector = postgres.Open(dsn)
	case DriverSqlServer:
		dialector = sqlserver.Open(dsn)
	case DriverMySql:
		dialector = mysql.Open(dsn)
	default:
		return nil, fmt.Errorf("sqlrepo: unsupported driver %q", driver)
	}

	if log == nil {
		log = logger.Default.LogMode(logger.Warn)
	}
	db, err := gorm.Open(dialector, &gorm.Config{
		TranslateError: true,
		Logger:         log,
	})
	if err != nil {
		return nil, err
	}
	for _, plugin := range plugins {
		if err := db.Use(plugin); err != nil {
			if sqlDB, dbErr := db.DB(); dbErr == nil {
				_ = sqlDB.Close()
			}
			return nil, fmt.Errorf("sqlrepo: plugin %s: %w", plugin.Name(), err)
		}
	}
	return db, nil
}

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&tenantModel{}, &userModel{}, &campaignModel{}, &campaignViewModel{})
}

func New(db *gorm.DB) *repository.Repositories {
	return repository.NewRepositories(repository.Constructors{
		Tenants:       func() repository.TenantRepository { return &TenantRepository{db: db} },
		Users:         func() repository.UserRepository { return &UserRepository{db: db} },
		Campaigns:     func() repository.CampaignRepository { return &CampaignRepository{db: db} },
		CampaignViews: func() repository.CampaignViewRepository { return newCampaignViewRepository(db) },
		Ping: func(ctx context.Context) error {
			sqlDB, err := db.DB()
			if err != nil {
				return err
			}
			return sqlDB.PingContext(ctx)
		},
		Close: func(context.Context) error {
			sqlDB, err := db.DB()
			if err != nil {
				return err
			}
			return sqlDB.Close()
		},
	})
}

func translateError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, gorm.ErrRecordNotFound):
		return repository.ErrNotFound
	case errors.Is(err, gorm.ErrDuplicatedKey):
		return repository.ErrDuplicate
	default:
		return err
	}
}

func paginate(db *gorm.DB, page repository.Pagination) *gorm.DB {
	// SQL Server requires ORDER BY when OFFSET is used; id breaks ties so pages are stable.
	return db.Order("created_at_utc DESC").Order("id").Offset(page.Offset).Limit(page.Limit)
}

func exists(db *gorm.DB) (bool, error) {
	var count int64
	if err := db.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
