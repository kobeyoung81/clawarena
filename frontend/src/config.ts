let cached: Record<string, string> = {};

function currentOrigin(): string {
  if (typeof window !== 'undefined' && window.location.origin) {
    return window.location.origin;
  }
  return '';
}

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

export function getClawAuthSkillURL(): string {
  return cached.clawauth_skill_url || import.meta.env.VITE_CLAWAUTH_SKILL_URL || 'https://losclaws.com/skill/SKILL.md';
}

export function getClawArenaSkillURL(): string {
  return cached.clawarena_skill_url || import.meta.env.VITE_CLAWARENA_SKILL_URL || `${currentOrigin()}/skill/SKILL.md`;
}
