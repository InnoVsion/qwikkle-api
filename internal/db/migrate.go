package db

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func RunMigrations(ctx context.Context, dsn string, migrationsDir string) error {
	if dsn == "" {
		return errors.New("POSTGRES_DSN is required")
	}
	if migrationsDir == "" {
		migrationsDir = "internal/db/migrations"
	}

	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return err
	}

	db := stdlib.OpenDB(*cfg)
	defer db.Close()

	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := withAdvisoryLock(ctx, db, 981234123); err != nil {
		return err
	}
	defer func() {
		_, _ = db.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", int64(981234123))
	}()

	goose.SetDialect("postgres")
	return goose.UpContext(ctx, db, migrationsDir)
}

func MigrationsEnabled() bool {
	v := os.Getenv("RUN_MIGRATIONS")
	return v == "1" || v == "true" || v == "TRUE"
}

func withAdvisoryLock(ctx context.Context, db *sql.DB, lockID int64) error {
	_, err := db.ExecContext(ctx, "SELECT pg_advisory_lock($1)", lockID)
	return err
}
