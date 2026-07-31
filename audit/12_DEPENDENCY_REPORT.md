# 12 — Dependency Report

## Direct Go Dependencies (go.mod)

| Dependency | Version | Purpose | Risk |
|-----------|---------|---------|------|
| `clickhouse-go/v2` | v2.47.0 | ClickHouse native driver | ✅ Actively maintained |
| `hyperloglog` | v0.2.6 | HLL sketches for cardinality | ⚠️ Small project, limited maintenance |
| `go-oidc/v3` | v3.20.0 | OIDC JWT verification | ✅ CoreOS maintained |
| `chi/v5` | v5.3.1 | HTTP router | ✅ Popular, maintained |
| `kafka-go` | v0.4.51 | Kafka client | ✅ Segmentio maintained |
| `zap` | v1.28.0 | Structured logging | ✅ Uber maintained |
| `swaggo/swag` | v1.16.6 | Swagger docs generation | ⚠️ Not used — no swagger docs generated |
| `swaggo/http-swagger/v2` | v2.0.2 | Swagger UI handler | ⚠️ Wired in routes but no docs generated |
| `prometheus/client_golang` | latest | Prometheus metrics | ✅ Standard |
| `google.golang.org/grpc` | latest | gRPC framework | ✅ Standard |
| `go.opentelemetry.io/*` | latest | OTel SDK + pdata | ✅ Standard |
| `golang.org/x/sync` | latest | errgroup | ✅ Standard lib extension |
| `gopkg.in/yaml.v3` | latest | YAML parsing | ✅ Standard |
| `golang.org/x/oauth2` | latest | OAuth2 support | ⚠️ Partially unused (see SEC-08) |

## Frontend Dependencies (from package.json)

| Dependency | Purpose | Risk |
|-----------|---------|------|
| `react` 19.x | UI framework | ✅ Standard |
| `vite` | Build tool | ✅ Standard |
| `lucide-react` | Icon library | ✅ Maintained |
| `typescript` | Type checking | ✅ Standard |

## Supply Chain Risks

1. **No `go.sum` verification in CI** — `go mod verify` is not run
2. **No dependency pinning** beyond go.sum
3. **No SBOM generation** (Software Bill of Materials)
4. **`npm audit`** not run in CI for frontend dependencies
5. **No Dependabot/Renovate** configuration for automated dependency updates
6. **`govulncheck`** is run in security-scan CI but only on push/PR to main

## Recommendations

1. Add `go mod verify` to CI pipeline
2. Configure Dependabot or Renovate for Go + npm
3. Generate SBOM on release (use `syft` or `trivy`)
4. Pin transitive dependencies via go.sum (already done implicitly)
5. Remove unused swagger dependencies or generate the docs
