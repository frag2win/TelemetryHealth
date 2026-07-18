package clickhouse

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"go.uber.org/zap"
)

// Client wraps the ClickHouse native driver connection.
type Client struct {
	conn   driver.Conn
	logger *zap.Logger
}

// NewClient establishes a native protocol connection to ClickHouse.
func NewClient(ctx context.Context, hosts []string, database, user, password string, logger *zap.Logger) (*Client, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: hosts,
		Auth: clickhouse.Auth{
			Database: database,
			Username: user,
			Password: password,
		},
		DialTimeout:     5 * time.Second,
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: 10 * time.Minute,
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("opening clickhouse connection: %w", err)
	}

	if err := conn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("pinging clickhouse: %w", err)
	}

	logger.Info("Connected to ClickHouse", zap.Strings("hosts", hosts))
	return &Client{conn: conn, logger: logger}, nil
}

// Close releases the connection pool.
func (c *Client) Close() {
	c.conn.Close()
}

// Conn exposes the raw driver connection for use by repositories.
func (c *Client) Conn() driver.Conn {
	return c.conn
}
