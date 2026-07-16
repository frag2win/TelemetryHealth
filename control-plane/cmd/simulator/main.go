package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/frag2win/TelemetryHealth/control-plane/internal/simulator"
	"go.uber.org/zap"
)

func main() {
	scenario := flag.String("scenario", "high_cardinality", "The failure scenario to simulate: high_cardinality, dropped_spans")
	tenant := flag.String("tenant", "acme-prod", "The tenant ID to inject telemetry for")
	endpoint := flag.String("endpoint", "127.0.0.1:4317", "The OTLP gRPC endpoint of the ingest-gateway")

	flag.Parse()

	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	sim := simulator.NewSimulator(logger, *endpoint)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger.Info("Starting failure simulation", zap.String("scenario", *scenario), zap.String("tenant", *tenant), zap.String("endpoint", *endpoint))

	var err error
	switch *scenario {
	case "high_cardinality":
		err = sim.InjectHighCardinality(ctx, *tenant)
	case "dropped_spans":
		err = sim.InjectDroppedSpans(ctx, *tenant)
	default:
		fmt.Printf("Unknown scenario: %s\n", *scenario)
		os.Exit(1)
	}

	if err != nil {
		logger.Fatal("Simulation failed", zap.Error(err))
	}

	logger.Info("Simulation completed successfully")
}
