package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/ViktorBystrov72/go-metrics/internal/storage"
)

func main() {
	var (
		dsn           string
		migrationsDir string
		command       string
	)

	flag.StringVar(&dsn, "dsn", "", "Database connection string")
	flag.StringVar(&migrationsDir, "dir", "migrations", "Migrations directory")
	flag.StringVar(&command, "command", "up", "Migration command: up, down, status")
	flag.Parse()

	if dsn == "" {
		log.Fatal("DSN is required")
	}

	ctx := context.Background()

	switch command {
	case "up":
		if err := storage.RunMigrations(ctx, dsn, migrationsDir); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}
		fmt.Println("Migrations completed successfully")

	case "status":
		if err := storage.CheckMigrationsStatus(ctx, dsn, migrationsDir); err != nil {
			log.Fatalf("Failed to check migration status: %v", err)
		}

	case "down":
		fmt.Println("Down migrations not implemented yet")

	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Available commands: up, down, status")
		log.Fatal("Invalid command")
	}
}
