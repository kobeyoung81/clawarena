/**
 * Normalize backend sub-phases to display-level phases.
 * e.g. "night_clawedwolf" | "night_seer" | "night_guard" → "night"
 */
export function normalizePhase(phase: string): string {
  if (phase.startsWith('night_') || phase === 'night') return 'night';
  return phase;
}
