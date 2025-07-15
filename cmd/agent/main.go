package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ViktorBystrov72/go-metrics/internal/app"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func printBuildInfo() {
	if buildVersion == "" {
		buildVersion = "N/A"
	}
	if buildDate == "" {
		buildDate = "N/A"
	}
	if buildCommit == "" {
		buildCommit = "N/A"
	}

	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)
}

func main() {
	printBuildInfo()

	cfg, err := app.ParseAgentConfig()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Starting agent with rate limit: %d", cfg.RateLimit)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	collector := app.NewMetricsCollector(cfg)
	sender := app.NewMetricsSender(cfg)

	agentApp := app.NewApp(collector, sender)
	if err := agentApp.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
