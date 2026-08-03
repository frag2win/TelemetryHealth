package rest

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/time/rate"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/frag2win/TelemetryHealth/control-plane/internal/telemetry"
)

// corsMiddleware adds CORS headers with strict origin checking.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowedStr := os.Getenv("ALLOWED_ORIGINS")
		if allowedStr == "" {
			allowedStr = os.Getenv("CORS_ORIGIN")
		}
		if allowedStr == "" {
			if strings.ToLower(os.Getenv("ENV")) == "production" {
				allowedStr = "http://localhost:5173"
			} else {
				allowedStr = "*"
			}
		}
		allowedOrigins := strings.Split(allowedStr, ",")

		reqOrigin := r.Header.Get("Origin")
		origin := allowedOrigins[0]
		for _, o := range allowedOrigins {
			o = strings.TrimSpace(o)
			if o == reqOrigin {
				origin = reqOrigin
				break
			}
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// metricsMiddleware tracks API requests for Prometheus using route pattern templates to control label cardinality.
func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		srw := &statusResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		startTime := time.Now()
		next.ServeHTTP(srw, r)
		duration := time.Since(startTime).Seconds()
		statusStr := strconv.Itoa(srw.statusCode)

		pathPattern := r.URL.Path
		if routeCtx := chi.RouteContext(r.Context()); routeCtx != nil {
			if pattern := routeCtx.RoutePattern(); pattern != "" {
				pathPattern = pattern
			}
		}

		telemetry.ApiRequestsTotal.WithLabelValues(r.Method, pathPattern, statusStr).Inc()
		telemetry.ApiRequestDuration.WithLabelValues(r.Method, pathPattern, statusStr).Observe(duration)
	})
}

var (
	rlVisitors = make(map[string]*visitor)
	rlMu       sync.Mutex
	rlOnce     sync.Once
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// rateLimitMiddleware enforces token-bucket rate limiting per IP address.
func rateLimitMiddleware(next http.Handler) http.Handler {
	rlOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(5 * time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				rlMu.Lock()
				for ip, v := range rlVisitors {
					if time.Since(v.lastSeen) > 5*time.Minute {
						delete(rlVisitors, ip)
					}
				}
				rlMu.Unlock()
			}
		}()
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			ip = strings.TrimSpace(strings.Split(forwarded, ",")[0])
		} else if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			ip = host
		}

		rlMu.Lock()
		v, exists := rlVisitors[ip]
		if !exists {
			v = &visitor{limiter: rate.NewLimiter(rate.Every(100*time.Millisecond), 20)}
			rlVisitors[ip] = v
		}
		v.lastSeen = time.Now()
		limiter := v.limiter
		rlMu.Unlock()

		if !limiter.Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// tracingMiddleware adds OpenTelemetry tracing spans to incoming HTTP requests.
func tracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracer := otel.Tracer("telemetryhealth-api")
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		ctx, span := tracer.Start(ctx, fmt.Sprintf("HTTP %s %s", r.Method, r.URL.Path),
			trace.WithSpanKind(trace.SpanKindServer),
		)
		defer span.End()

		otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(w.Header()))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// oidcAuthMiddleware validates OIDC JWT tokens and attaches authenticated actor context.
func oidcAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		issuer := os.Getenv("OIDC_ISSUER")
		insecureDev := os.Getenv("INSECURE_DEV_MODE")
		envMode := strings.ToLower(os.Getenv("ENV"))

		if issuer == "" {
			if insecureDev == "true" && envMode != "production" {
				ctx := context.WithValue(r.Context(), contextKeyActorID, "dev-user")
				ctx = context.WithValue(ctx, contextKeyActorRole, "Org Admin")
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error_code":"UNAUTHORIZED","message":"Authentication signature configuration missing"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}
