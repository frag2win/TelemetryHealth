package main

import (
	"database/sql"
	"log"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/storage/clickhouse"
	"go.uber.org/zap"
)

func main() {
	// Connect to default DB
	db, err := sql.Open("clickhouse", "clickhouse://127.0.0.1:9000?dial_timeout=5s")
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer db.Close()

	logger, _ := zap.NewDevelopment()
	schema := clickhouse.NewSchema(db, logger)
	if err := schema.InitSchema(); err != nil {
		log.Fatalf("InitSchema failed: %v", err)
	}

	log.Println("Schema initialized successfully!")
}
