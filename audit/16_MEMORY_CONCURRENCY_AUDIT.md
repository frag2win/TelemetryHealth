# 16 — Memory & Concurrency Audit

### MC-01: Rate Limiter Map Grows Unbounded
- **Severity**: 🟠 High
- **File**: `control-plane/internal/api/rest/server.go` L178-201
- **Type**: Memory leak
- **Explanation**: The cleanup goroutine uses `time.Tick` (which never stops), and the `rlVisitors` map is only cleaned every 5 minutes. Under a DDoS with spoofed IPs, this map will grow without bound, consuming heap memory.
- **Can it OOM**: Yes, under sustained attack.
- **Can it deadlock**: No — uses `sync.Mutex` correctly.
- **Fix**: Use `sync.Map` with TTL, or use a fixed-size LRU cache (e.g., `golang-lru`).

### MC-02: Goroutine Leak in Rate Limiter Cleanup
- **Severity**: 🟡 Medium
- **File**: `control-plane/internal/api/rest/server.go` L190-201
- **Type**: Goroutine leak
- **Explanation**: `time.Tick` returns a channel that is never closed. The goroutine lives for the lifetime of the process. While this isn't a problem for long-running servers, it's a leak in test scenarios and prevents garbage collection of the middleware.
- **Fix**: Use `time.NewTicker` with explicit `Stop()` tied to server shutdown.

### MC-03: `TelemetryPoller` Has No Stop Mechanism
- **Severity**: 🟡 Medium
- **File**: `control-plane/internal/alerting/poller.go` L35-48
- **Type**: Goroutine leak
- **Explanation**: `Start()` launches a goroutine that only stops when the context is cancelled. However, in `api-server/main.go`, the context passed may be `context.Background()`, which is never cancelled. The poller goroutine will leak on server shutdown.
- **Fix**: Pass the signal-aware context (`runCtx`) or return a cancel function.

### MC-04: SigNozBridge.lastFired Map Grows Without Bound
- **Severity**: 🟡 Medium
- **File**: `control-plane/internal/alerting/signoz_bridge.go` L69
- **Type**: Memory leak (slow)
- **Explanation**: `lastFired` map entries are never cleaned up. Each unique `alert_id` adds an entry that persists forever.
- **Fix**: Add a cleanup routine that removes entries older than `2 * cooldown`, or use an LRU cache.

### MC-05: Circuit Breaker Correctly Releases Lock During Execution
- **Severity**: ✅ No issue
- **File**: `processor/failopen/circuit_breaker.go` L51-64
- **Explanation**: The mutex is released at L64 before executing `fn`, then re-acquired at L77. This is correct and avoids holding the lock during potentially long operations. Well done.

### MC-06: Cardinality Tracker Uses Single Mutex
- **Severity**: 🟢 Low
- **File**: `processor/cardinality/tracker.go` L68
- **Type**: Contention risk
- **Explanation**: All `Observe()` and `Flush()` calls are serialized through a single `sync.Mutex`. Under high throughput (100K+ spans/sec), this will become a bottleneck.
- **Fix**: Use sharded locks (e.g., `sync.Map`, or shard by service name hash) for `Observe`, and a read-write lock for `Flush`.

### MC-07: `errgroup` Captures Variable by Reference
- **Severity**: 🟢 Low (correct in this case)
- **File**: `control-plane/internal/storage/clickhouse/health_repository.go` L248-258
- **Explanation**: The `weights` variable is written inside an errgroup goroutine and read after `g.Wait()`. This is safe because errgroup guarantees all goroutines complete before Wait returns. No race condition.

### MC-08: Kafka Consumer Timer Reset Logic
- **Severity**: 🟢 Low (correct)
- **File**: `control-plane/internal/kafka/consumer.go` L100-106
- **Explanation**: The timer drain pattern (`if !timer.Stop() { select { case <-timer.C: default: } }`) is the correct Go idiom for safely resetting a timer. Well implemented.
