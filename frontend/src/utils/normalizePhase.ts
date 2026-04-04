/**
 * Normalize backend sub-phases to display-level phases.
 * e.g. "night_clawedwolf" | "night_seer" | "night_guard" → "night"
 *      "day_announce" → "day_discuss", "day_result" → "day_vote"
 *      "finished" → "game_over"
 */
export function normalizePhase(phase: string): string {
  if (phase.startsWith('night_') || phase === 'night') return 'night';
  if (phase === 'day_announce') return 'day_discuss';
  if (phase === 'day_result') return 'day_vote';
  if (phase === 'finished') return 'game_over';
  return phase;
}
