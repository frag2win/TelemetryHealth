🎯 1. Rate Limiting — Where to Add It
Location: control-plane/internal/api/rest/server.go
Current middleware stack (lines 140-150):

go
r.Use(middleware.RequestID)
r.Use(middleware.RealIP)
r.Use(corsMiddleware)
r.Use(metricsMiddleware)
Implementation Needed:
Add a rate limiting middleware between RealIP and CORS:

go
// Add this import
import "golang.org/x/time/rate"

// Add this middleware
r.Use(rateLimitMiddleware)
Suggested implementation:

go
// rateLimitMiddleware adds per-IP rate limiting (10 requests/second, burst 20)
func rateLimitMiddleware(next http.Handler) http.Handler {
    visitors := make(map[string]*rate.Limiter)
    var mu sync.Mutex
    
    getLimiter := func(ip string) *rate.Limiter {
        mu.Lock()
        defer mu.Unlock()
        
        if limiter, exists := visitors[ip]; exists {
            return limiter
        }
        
        limiter := rate.NewLimiter(rate.Every(100*time.Millisecond), 20) // 10 req/s, burst 20
        visitors[ip] = limiter
        return limiter
    }
    
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ip := r.RemoteAddr
        if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
            ip = strings.Split(forwarded, ",")[0]
        }
        
        limiter := getLimiter(ip)
        if !limiter.Allow() {
            writeError(w, "RATE_LIMIT_EXCEEDED", "Too many requests, please try again later", http.StatusTooManyRequests)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}
Why this location: The middleware stack is in server.go at line 139-147. Add it after RealIP so you can extract the real client IP before rate limiting.

🎯 2. Distributed Tracing Propagation — Where to Add It
Location A: control-plane/internal/api/rest/server.go (HTTP handlers)
Current state: OTel SDK is imported (github.com/frag2win/TelemetryHealth/control-plane/internal/telemetry) but not used in handlers.

Implementation Needed:
Add tracing middleware to propagate trace context:

go
// Add this import
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/trace"
)

// Add this middleware (after metricsMiddleware)
r.Use(tracingMiddleware)
Suggested implementation:

go
// tracingMiddleware adds distributed tracing to all API requests
func tracingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        tracer := otel.Tracer("telemetryhealth-api")
        
        // Extract incoming trace context (from dashboard or other services)
        ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
        
        // Start a new span
        ctx, span := tracer.Start(ctx, fmt.Sprintf("HTTP %s %s", r.Method, r.URL.Path),
            trace.WithSpanKind(trace.SpanKindServer),
        )
        defer span.End()
        
        // Inject trace context into response headers
        otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(w.Header()))
        
        // Continue with trace context in request
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
Location B: control-plane/internal/mcp/client.go (SigNoz MCP client calls)
When calling SigNoz MCP server, you need to inject trace context so the trace chain continues:

go
// In your HTTP client preparation, add:
import "go.opentelemetry.io/otel/propagation"

// Before making HTTP request to SigNoz
req := req.WithContext(ctx)
otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
Location C: connector/ (Cross-signal correlation)
The connector should emit spans when correlating metrics/logs/traces:

go
// In connector/internal/correlation/correlator.go (if it exists)
tracer := otel.Tracer("telemetryhealth-connector")
ctx, span := tracer.Start(context.Background(), "cross-signal-correlation")
defer span.End()

// Add span attributes
span.SetAttributes(
    attribute.String("tenant.id", tenantID),
    attribute.Int("metrics.count", len(metrics)),
    attribute.Int("traces.count", len(traces)),
)
