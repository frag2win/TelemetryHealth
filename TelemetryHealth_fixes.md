# TelemetryHealth — Fix List & Hardcoded Code Rewrite

Analysis based on a full clone + read of `frag2win/TelemetryHealth` (main branch).
Two places return hardcoded/fabricated data instead of real query results. Everything
else checked (cardinality tracker, mTLS/SPIFFE authz, remediation validator, Kafka
wiring) is genuinely implemented — this doc covers only what needs fixing.

---

## 1. Where "hardcoded" was found

### 1a. `control-plane/internal/mcp/tools.go` (fully hardcoded)

```go
func GetTelemetryHealth(tenantID string) (*HealthResponse, error) {
	// Query SigNoz Query Builder API (mocked/simulated for AI agent context)
	return &HealthResponse{
		HealthScore: 85,
		Metrics: MetricsPayload{
			Cardinality: MetricValue{Value: "1.2M", Change: 14.5},
			Orphans:     MetricValue{Value: "432", Change: -5.2},
			Coverage:    MetricValue{Value: "14", Change: 0},
		},
		Remediation: RemediationPayload{
			IssueType: "cardinality_explosion",
			Yaml: `processors:
  attributes/remediation:
    actions:
      - key: "user_id"
        action: "delete"`,
			Validated: true,
		},
		TenantId: tenantID,
		Version:  "v1.1.0-mcp",
	}, nil
}
```

- `tenantID` is accepted but never used — every tenant gets identical output.
- Comment above it: `// Exposing exact functions expected by the judges/tests in tools.go` —
  this function exists to satisfy a rubric, not to do work.
- Your **real** REST handler for the same data
  (`control-plane/internal/api/rest/server.go:335-396`, `GetTenantHealth`) already does
  this correctly — it calls `s.healthRepo.QueryHealthMetrics(ctx, tenantID)`, generates
  remediation YAML only when an issue is found, validates it, and returns a proper 503
  if ClickHouse isn't configured (no mock fallback). `tools.go` just isn't using any of
  that.

### 1b. `control-plane/internal/storage/clickhouse/health_repository.go:161` (partially hardcoded)

```go
Tokens:            4120, // default placeholder
```

Inside `QueryAgentTraces()` — the query itself is real, but every scanned row gets a
fixed token count regardless of what the query returned. Worse: if the query returns
zero rows (true on any DB without seeded `gen_ai.*` spans — i.e. almost always), the
function falls through to a fully fabricated array of fake traces with invented IDs,
costs, and "decision steps." This is lower priority than 1a since it's a UI demo panel,
not a core scoring path, but it's the same pattern.

---

## 2. Fix list, in priority order

**Must-fix before demo (a judge reading code will hit these):**

1. **Rewrite `mcp.GetTelemetryHealth`** to call the real `HealthRepository` instead of
   returning a static struct. See rewrite below — it mirrors your own working
   `GetTenantHealth` REST handler.
2. **README roadmap table**: mark M5 (Kafka) as done. `kafka.EnsureTopics`,
   `kafka.NewProducer`, and `kafka.NewWorkerSet` are already wired into
   `cmd/ingest-gateway/main.go` and `cmd/worker/main.go`. The README currently says
   "🔜 Planned" — it's understating what you've built.
3. **README / architecture diagram wording**: change "validated through a
   shadow-Collector dry-run" to something accurate, e.g. "validated via YAML
   structural checks and an OTel component allowlist." `remediation/validator.go` only
   does Phase 1 (parse + allowlist) — no collector process is ever started. The
   in-code comment already says Phase 2 (real otelcol dry-run) is future work; the
   README should say the same.
4. **`validator.go` log line**: `"Running shadow-collector validation (Phase 1:
   structural check)"` → drop "shadow-collector" from the message. Keep the "(Phase 1:
   structural check)" part — that's accurate.

**Should-fix if you have time:**

5. **`QueryAgentTraces` fallback** (1b above): either seed ClickHouse with real
   `gen_ai.*` demo spans before presenting, or relabel that dashboard panel clearly as
   "simulated" so it doesn't implicitly claim to be live pipeline data.
6. **Verify test coverage numbers** — I couldn't run `go test ./... -cover` myself
   (sandbox network can't reach `proxy.golang.org`). Run it locally and confirm
   93.3% / 93.9% / 100% are still current before quoting them to judges.

**Polish:**

7. Delete any comment phrased like "expected by the judges/tests" before the repo is
   presented publicly — even after the function is made real, that comment reads as an
   admission if anyone scrolls up.

---

## 3. Rewrite: `mcp/tools.go` → `GetTelemetryHealth`

This mirrors the real logic already working in
`control-plane/internal/api/rest/server.go:335-396`, adapted to the MCP tool's
signature. It needs the same three dependencies the REST server already has:
`*clickhouse.HealthRepository`, `*remediation.Generator`, `*remediation.Validator`.

```go
package mcp

import (
	"context"
	"fmt"

	"github.com/frag2win/TelemetryHealth/control-plane/internal/remediation"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/storage/clickhouse"
	"go.uber.org/zap"
)

type MetricValue struct {
	Value  string  `json:"value"`
	Change float64 `json:"change"`
}

type MetricsPayload struct {
	Cardinality MetricValue `json:"cardinality"`
	Orphans     MetricValue `json:"orphans"`
	Coverage    MetricValue `json:"coverage"`
}

type RemediationPayload struct {
	IssueType string `json:"issueType"`
	Yaml      string `json:"yaml"`
	Validated bool   `json:"validated"`
}

type HealthResponse struct {
	HealthScore float64            `json:"healthScore"`
	Metrics     MetricsPayload     `json:"metrics"`
	Remediation RemediationPayload `json:"remediation"`
	TenantId    string             `json:"tenantId"`
	Version     string             `json:"version"`
}

// Toolset holds the real dependencies the MCP tools need. Construct once at
// startup (same repo/generator/validator instances the REST server uses) and
// pass it into the tool functions — this replaces the old package-level
// hardcoded stub and the init-time `defaultGen` singleton.
type Toolset struct {
	HealthRepo *clickhouse.HealthRepository
	Generator  *remediation.Generator
	Validator  *remediation.Validator
	Logger     *zap.Logger
}

func NewToolset(repo *clickhouse.HealthRepository, gen *remediation.Generator, val *remediation.Validator, logger *zap.Logger) *Toolset {
	return &Toolset{HealthRepo: repo, Generator: gen, Validator: val, Logger: logger}
}

// GetTelemetryHealth queries real tenant health data instead of returning a
// static payload. Returns an error (not fabricated data) if ClickHouse isn't
// configured, matching the REST endpoint's 503 behavior.
func (t *Toolset) GetTelemetryHealth(ctx context.Context, tenantID string) (*HealthResponse, error) {
	if t.HealthRepo == nil {
		return nil, fmt.Errorf("health repository not configured — ClickHouse unavailable")
	}

	metrics, err := t.HealthRepo.QueryHealthMetrics(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("querying health metrics: %w", err)
	}

	issueType := metrics.RemediationIssue
	remediationYaml := ""
	validated := false
	if issueType != "" && t.Generator != nil {
		remediationYaml, err = t.Generator.Generate(ctx, issueType)
		if err != nil {
			t.Logger.Error("failed to generate remediation yaml", zap.Error(err))
		}
		if t.Validator != nil && remediationYaml != "" {
			validated, _ = t.Validator.Validate(ctx, remediationYaml)
		}
	}

	return &HealthResponse{
		HealthScore: metrics.CompositeScore,
		Metrics: MetricsPayload{
			Cardinality: MetricValue{Value: fmt.Sprintf("%d", metrics.CardinalityMax)},
			Orphans:     MetricValue{Value: fmt.Sprintf("%d", metrics.OrphanCount)},
			Coverage:    MetricValue{Value: fmt.Sprintf("%d", metrics.ActiveServices)},
		},
		Remediation: RemediationPayload{
			IssueType: issueType,
			Yaml:      remediationYaml,
			Validated: validated,
		},
		TenantId: tenantID,
		Version:  "v1.1.0-mcp",
	}, nil
}

// GenerateRemediation is unchanged in shape but now hangs off the injected
// Toolset instead of a package-level singleton, so it shares the same
// generator instance as the REST server (one source of truth).
func (t *Toolset) GenerateRemediation(ctx context.Context, issueType string) (string, error) {
	if t.Generator == nil {
		return "", fmt.Errorf("remediation generator not configured")
	}
	return t.Generator.Generate(ctx, issueType)
}
```

**Wiring change needed at the call site** (wherever your MCP server registers these
tools — likely `cmd/api-server/main.go` near where `NewServer(logger, healthRepo)` is
already called): construct one `mcp.NewToolset(healthRepo, generator, validator,
logger)` using the same instances passed into `rest.NewServer`, and register
`toolset.GetTelemetryHealth` / `toolset.GenerateRemediation` as the MCP tool handlers
instead of the old package-level functions. This also lets you delete the
`defaultGen` package-level singleton in the old file, since the generator now comes
from the same place the REST API gets it.

Cardinality/Orphans/Coverage `Change` fields are left at zero above since the old
stub's `14.5` / `-5.2` were fabricated deltas with no real computation behind them —
if you want real deltas, reuse `cardChange()` / `calculateDelta()` from
`server.go`, which already do this for the REST path.
