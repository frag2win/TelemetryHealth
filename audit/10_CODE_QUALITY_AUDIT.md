# 10 — Code Quality Audit

### CQ-01: Magic Numbers
- **Severity**: 🟡 Medium
- **Locations**:
  - `100 * time.Millisecond` burst rate, `20` burst size, `5 * time.Minute` cleanup interval — all hardcoded with no named constants in the rate limiter
  - `1_000_000` cardinality threshold, `1000` orphan threshold, `10` active services minimum — hardcoded in health score calculation
  - `64 * 1024` max YAML size — should be a named constant
  - `100` max cardinality keys per service — duplicated in both `grpc_server.go` and `tracker.go`

### CQ-02: Error Handling Inconsistencies
- Some handlers silently swallow errors (e.g., `GetTenantIssues` returns hardcoded data, no error possible)
- Some handlers return 503 on DB errors, others return 501, others return 500
- `encodeResponse` logs encode errors but the HTTP status has already been sent

### CQ-03: Inconsistent Naming
- `GetTenantHealth` vs `handleBehaviorGraph` vs `HandleTenantConfigGet` — three different naming conventions for handlers
- `GetCoverage` has a misleading godoc: appears above the wrong function

### CQ-04: Comments Referencing Non-existent "Finding" Numbers
- **Severity**: 🟢 Low
- **Explanation**: Code comments reference "Finding 7.2", "Finding 8.2", "Finding 10.2", "Improvement #2.1" etc. These appear to reference an earlier code review, but the review document isn't in the repo, making the references opaque.

### CQ-05: Godoc Comments Missing or Incorrect
- **Severity**: 🟢 Low
- **Explanation**: Several exported functions lack godoc comments (`Named`, `DB`). Some comments are on the wrong function (see CQ-03).

### CQ-06: Inline CSS in React Components
- **Severity**: 🟢 Low
- **File**: `dashboard/src/App.tsx` L368-394
- **Explanation**: Extensive inline `style={{}}` objects instead of CSS classes. This hurts readability and prevents style reuse.
