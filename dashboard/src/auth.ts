/**
 * Centralized authentication helper for TelemetryHealth dashboard.
 *
 * In development (INSECURE_DEV_MODE on backend): no auth header is sent —
 * the backend middleware auto-injects dev-user context.
 *
 * In production: this should be replaced with a real OIDC token acquisition
 * flow (e.g., via an auth provider SDK).
 */

export function getAuthHeaders(): Record<string, string> {
  // When a real OIDC provider is configured, acquire a token here.
  // For now, return empty headers — the backend's INSECURE_DEV_MODE
  // handles authentication bypass in development.
  return {};
}

/**
 * Convenience wrapper for authenticated fetch calls.
 */
export async function authFetch(url: string, options: RequestInit = {}): Promise<Response> {
  const authHeaders = getAuthHeaders();
  const headers = {
    ...authHeaders,
    ...(options.headers || {}),
  };
  return fetch(url, { ...options, headers });
}
