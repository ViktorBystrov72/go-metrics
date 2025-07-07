package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// RunMigrations выполняет миграции базы данных
func RunMigrations(ctx context.Context, dsn string, migrationsDir string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database for migrations: %w", err)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	if err := goose.Up(db, migrationsDir); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Printf("Database migrations completed successfully")
	return nil
}

// CheckMigrationsStatus проверяет статус миграций
func CheckMigrationsStatus(ctx context.Context, dsn string, migrationsDir string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database for status check: %w", err)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	if err := goose.Status(db, migrationsDir); err != nil {
		return fmt.Errorf("failed to check migration status: %w", err)
	}

	return nil
}
