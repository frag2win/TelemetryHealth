package main

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/storage/clickhouse"
	"go.uber.org/zap"
)

func main() {
	// Connect to default DB
	chHost := os.Getenv("CH_HOST")
	if chHost == "" {
		chHost = "127.0.0.1"
	}
	chPort := os.Getenv("CH_PORT")
	if chPort == "" {
		chPort = "9000"
	}
	chAddr := "clickhouse://" + chHost + ":" + chPort + "?dial_timeout=5s"
	db, err := sql.Open("clickhouse", chAddr)
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
