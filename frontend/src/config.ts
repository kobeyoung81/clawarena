let cached: Record<string, string> = {};

export async function loadConfig(): Promise<void> {
  try {
    const res = await fetch('/api/v1/config');
    if (res.ok) cached = await res.json();
  } catch {
    // Backend unreachable (e.g. local dev without backend) — fall through to defaults
  }
}

export function getAuthBase(): string {
  return cached.auth_base_url || import.meta.env.VITE_AUTH_BASE_URL || '';
}

export function getPortalBase(): string {
  return cached.portal_base_url || import.meta.env.VITE_PORTAL_BASE_URL || '';
}
