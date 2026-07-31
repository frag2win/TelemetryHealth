# 07 — Frontend Audit

### FE-01: Hardcoded Auth Token in Frontend
- **Severity**: 🔴 Critical (see SEC-02)
- **File**: `dashboard/src/App.tsx` L179
- **Explanation**: `Authorization: 'Bearer health-demo-key-2026'` is hardcoded in the fetch call. This token is visible in the browser's network tab and the compiled JS bundle.

### FE-02: No Error Boundary Fallback UI
- **Severity**: 🟡 Medium
- **Explanation**: `ErrorBoundary` components wrap views but there's no visible fallback UI defined — a component crash may render a blank panel.

### FE-03: Client-Side Metric Simulation Overrides Real Data
- **Severity**: 🟠 High
- **File**: `dashboard/src/App.tsx` L236-275
- **Explanation**: `simulateTimeRangeMetrics` overwrites real API data with hardcoded values when time range is changed. Real data from the API (cardinality, orphans) is replaced with fake strings like `"1.1M"`.
- **How to fix**: Time-range filtering should be done server-side via query parameters.

### FE-04: 20-Second Polling Interval with No Backoff
- **Severity**: 🟡 Medium
- **File**: `dashboard/src/App.tsx` L223-225
- **Explanation**: Fixed 20s polling interval even when the tab is not visible. Should use `document.visibilityState` to pause polling when hidden.

### FE-05: No Loading Skeleton
- **Severity**: 🟢 Low
- **Explanation**: Loading state shows a spinning icon but no skeleton/placeholder layout — causes layout shift when data arrives.

### FE-06: Vite Proxy Configuration Missing
- **Severity**: 🟡 Medium
- **File**: `dashboard/vite.config.ts`
- **Explanation**: Frontend fetches `/api/v1/...` (relative URL) but vite.config.ts likely doesn't have a proxy configured, meaning dev mode will fail with CORS errors unless the backend is on the same port.

### FE-07: No Accessibility (a11y) Considerations
- **Severity**: 🟡 Medium
- **Explanation**: Navigation buttons use `<button>` (good) but there are no `aria-label` attributes on icon-only buttons, no focus management for view transitions, and no skip-to-content links.

### FE-08: Tenant List is Hardcoded in Frontend
- **Severity**: 🟡 Medium
- **File**: `dashboard/src/App.tsx` L55-61
- **Explanation**: The tenant selector dropdown contains hardcoded tenant IDs. Should be fetched from the API.

### FE-09: No TypeScript Strict Mode
- **Severity**: 🟢 Low
- **File**: `dashboard/tsconfig.json`
- **Explanation**: Should verify that `strict: true` is enabled for maximum type safety.

### FE-10: `Promise.all().catch()` Returns Wrong Shape
- **Severity**: 🟡 Medium
- **File**: `dashboard/src/App.tsx` L283-287
- **Explanation**: The `.catch(() => [null, null, null])` returns an array, but the destructured `[agentsRes, orphansRes, coverageRes]` from `Promise.all` expects individual promise results. If the catch fires, all three will be `undefined` because `Promise.all().catch()` replaces the entire result.
- **How to fix**: Use `Promise.allSettled` or wrap each fetch individually.
