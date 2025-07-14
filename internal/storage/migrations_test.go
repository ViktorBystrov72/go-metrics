package storage

import (
	"context"
	"testing"
)

func TestRunMigrations(t *testing.T) {
	ctx := context.Background()
	err := RunMigrations(ctx, "invalid-dsn", "./migrations")
	if err == nil {
		t.Error("RunMigrations() должен вернуть ошибку для неверного DSN")
	}
}

func TestCheckMigrationsStatus(t *testing.T) {
	ctx := context.Background()
	err := CheckMigrationsStatus(ctx, "invalid-dsn", "./migrations")
	if err == nil {
		t.Error("CheckMigrationsStatus() должен вернуть ошибку для неверного DSN")
	}
}
