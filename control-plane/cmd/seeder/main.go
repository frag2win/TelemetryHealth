package main

import (
	"context"
	"log"
	"math/rand"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/uuid"
)

// Seeder injects realistic mock rows into ClickHouse for development and demo purposes.
// Usage: go run ./cmd/seeder --host localhost:9000 --tenant <uuid>
func main() {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{"localhost:9000"},
		Auth: clickhouse.Auth{
			Database: "telemetry_health",
			Username: "telemetry",
			Password: "",
		},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer conn.Close()

	ctx := context.Background()
	tenantID := "00000000-0000-0000-0000-000000000001"

	// --- Seed cardinality_signal ---
	log.Println("Seeding cardinality_signal...")
	cardBatch, err := conn.PrepareBatch(ctx, `INSERT INTO telemetry_health.cardinality_signal 
		(tenant_id, service, attribute_key, window_start, unique_estimate)`)
	if err != nil {
		log.Fatalf("prepare cardinality batch: %v", err)
	}

	services := []string{"checkout", "payments", "user-service", "api-gateway", "notifications"}
	attrs := []string{"user_id", "session_id", "request_id", "trace_id"}

	for _, svc := range services {
		for _, attr := range attrs {
			_ = cardBatch.Append(
				uuid.MustParse(tenantID),
				svc,
				attr,
				time.Now().Add(-10*time.Minute),
				uint64(rand.Intn(2_000_000)),
			)
		}
	}
	if err := cardBatch.Send(); err != nil {
		log.Printf("cardinality seed error: %v", err)
	} else {
		log.Println("✓ cardinality_signal seeded")
	}

	// --- Seed orphan_signal ---
	log.Println("Seeding orphan_signal...")
	orphanBatch, err := conn.PrepareBatch(ctx, `INSERT INTO telemetry_health.orphan_signal 
		(tenant_id, trace_id, span_id, parent_span_id, collector_id, detected_at)`)
	if err != nil {
		log.Fatalf("prepare orphan batch: %v", err)
	}

	for i := 0; i < 432; i++ {
		_ = orphanBatch.Append(
			uuid.MustParse(tenantID),
			uuid.New().String(),
			uuid.New().String(),
			uuid.New().String(),
			"collector-01",
			time.Now().Add(-time.Duration(rand.Intn(25))*time.Minute),
		)
	}
	if err := orphanBatch.Send(); err != nil {
		log.Printf("orphan seed error: %v", err)
	} else {
		log.Println("✓ orphan_signal seeded")
	}

	// --- Seed coverage_signal ---
	log.Println("Seeding coverage_signal...")
	covBatch, err := conn.PrepareBatch(ctx, `INSERT INTO telemetry_health.coverage_signal 
		(tenant_id, service, last_seen_at, baseline_expected)`)
	if err != nil {
		log.Fatalf("prepare coverage batch: %v", err)
	}

	for _, svc := range services {
		_ = covBatch.Append(
			uuid.MustParse(tenantID),
			svc,
			time.Now().Add(-time.Duration(rand.Intn(5))*time.Minute),
			uint8(1),
		)
	}
	if err := covBatch.Send(); err != nil {
		log.Printf("coverage seed error: %v", err)
	} else {
		log.Println("✓ coverage_signal seeded")
	}

	log.Printf("\nSeeding complete for tenant %s", tenantID)
}
