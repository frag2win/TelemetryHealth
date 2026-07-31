# 17 — API Review

## Endpoint Inventory

| Method | Path | Auth | Data Source | Status |
|--------|------|------|-------------|--------|
| GET | `/healthz` | None | Static | ✅ Production-ready |
| GET | `/readyz` | None | Nil check | ⚠️ Should ping DB |
| GET | `/metrics` | None | Prometheus | ✅ OK |
| GET | `/swagger/*` | None | Static | ⚠️ No swagger docs generated |
| GET | `/api/v1/tenant/{id}/health` | OIDC | ClickHouse + fallback | ⚠️ Returns mock data |
| POST | `/api/v1/tenant/{id}/simulate` | OIDC | Simulator → gRPC | ⚠️ Creates outbound gRPC |
| GET | `/api/v1/tenant/{id}/issues` | OIDC | **Hardcoded** | ❌ Fake data |
| GET | `/api/v1/tenant/{id}/agents` | OIDC | ClickHouse + fallback | ⚠️ Falls back to mock |
| GET | `/api/v1/tenant/{id}/coverage` | OIDC | **Hardcoded** | ❌ Fake data |
| GET | `/api/v1/tenant/{id}/traces/orphans` | OIDC | **Hardcoded** | ❌ Fake data |
| GET | `/api/v1/tenant/{id}/config` | OIDC | ClickHouse | ✅ Real data |
| PUT | `/api/v1/tenant/{id}/config` | OIDC | ClickHouse | ✅ Real data |
| POST | `/api/v1/tenant/{id}/config` | OIDC | ClickHouse | ✅ Real data |
| GET | `/api/v1/tenant/{id}/behavior` | OIDC | Engine | ✅ Real data |
| GET | `/api/v1/tenant/{id}/root-cause` | OIDC | Engine | ✅ Real data |
| GET | `/api/v1/tenant/{id}/replay` | OIDC | SigNoz + CH + Mock | ⚠️ Triple fallback |
| POST | `/api/v1/remediation/apply` | OIDC | CH audit log | ✅ Real data |
| GET | `/api/v1/signoz/health` | OIDC | HTTP ping | ✅ OK |
| GET | `/api/v1/signoz/config` | OIDC | Env vars | ✅ OK |
| GET | `/api/agents/{id}/traces/{id}/behavior` | OIDC | CH + fallback | ⚠️ Falls back |
| GET | `/api/agents/{id}/traces/{id}/decisions` | OIDC | CH + fallback | ⚠️ Falls back |
| GET | `/api/agents/{id}/traces/{id}/root-cause` | OIDC | CH + fallback | ⚠️ Falls back |

## Summary

- **3 endpoints** return completely hardcoded data (never query a database)
- **7 endpoints** fall back to mock/fake data when DB is empty
- **Only 9 endpoints** are fully connected to real data sources
- **No pagination** on any list endpoint
- **No rate limiting granularity** — all endpoints share the same rate limiter
- **No request validation** — POST bodies are decoded without Content-Type checks
- **No response schema documentation** — Swagger route exists but serves nothing

## API Design Recommendations

1. Remove hardcoded endpoints or clearly mark them as `v0`/stub
2. Add OpenAPI spec generation via `swaggo` annotations
3. Add pagination (`?limit=50&offset=0`) to all list endpoints
4. Standardize error response format: `{"error": {"code": "...", "message": "..."}}`
5. Move `/api/v1/remediation/apply` to `/api/v1/tenant/{id}/remediation/apply` for consistency
6. Add API versioning headers (`X-API-Version`, `Deprecation`)
