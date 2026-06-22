package migrations

import (
	"database/sql"
	"embed"
	"errors"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"go.uber.org/zap"
)

//go:embed *.sql
var fs embed.FS

func RunMigrations(db *sql.DB) error {
	migrationsLogger.Info("prepare migrations")

	driver, err := mysql.WithInstance(db, &mysql.Config{})
	if err != nil {
		return err
	}

	source, err := iofs.New(fs, ".")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", source, "mysql", driver)
	if err != nil {
		return err
	}

	migrationsLogger.Info("run migrations")

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			migrationsLogger.Info("no migrations to apply")
			return nil
		}

		migrationsLogger.Error("Failed to apply migrations", zap.Error(err))
		return err
	}

	migrationsLogger.Info("migrations applied successfully")

	return nil
}
