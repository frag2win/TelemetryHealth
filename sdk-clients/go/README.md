# TelemetryHealth Go SDK Client

A lightweight Go client for status polling and querying service health scores (PRD §7, §13).

## Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/frag2win/TelemetryHealth/sdk-clients/go/client"
)

func main() {
    c := client.NewClient("http://localhost:8080", "your-tenant-uuid")
    score, err := c.GetHealthScore(context.Background(), "checkout-service", "production")
    if err != nil {
        log.Fatalf("failed to query score: %v", err)
    }
    fmt.Printf("Current Health Score: %.2f\n", score)
}
```
