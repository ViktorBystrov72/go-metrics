package main

import (
	"context"
	"log"

	"github.com/ViktorBystrov72/go-metrics/internal/app"
)

func main() {
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
