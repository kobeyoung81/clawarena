/**
 * narrativeFormatter.ts
 * Maps raw game action strings and events to atmospheric story text.
 */

interface FormatOptions {
  agents?: Record<number, string>; // agentId -> name
}

// Map action command patterns to narrative text
const ACTION_PATTERNS: Array<[RegExp, (m: RegExpMatchArray, opts: FormatOptions) => string]> = [
  [
    /kill\s+target:(\d+)/i,
    (m, { agents } = {}) => {
      const name = agents?.[Number(m[1])] ?? `Agent ${m[1]}`;
      return `🐺 The wolves chose their prey... ${name} is targeted.`;
    },
  ],
  [
    /investigate\s+target:(\d+)/i,
    (m, { agents } = {}) => {
      const name = agents?.[Number(m[1])] ?? `Agent ${m[1]}`;
      return `👁 The seer peers into the darkness, seeking the truth about ${name}...`;
    },
  ],
  [
    /protect\s+target:(\d+)/i,
    (m, { agents } = {}) => {
      const name = agents?.[Number(m[1])] ?? `Agent ${m[1]}`;
      return `🛡 A guardian stands watch over ${name} through the night.`;
    },
  ],
  [
    /vote\s+target:(\d+)/i,
    (m, { agents } = {}) => {
      const name = agents?.[Number(m[1])] ?? `Agent ${m[1]}`;
      return `⚖️ A finger is pointed at ${name}.`;
    },
  ],
  [
    /speak\s+(.+)/i,
    (m) => `💬 "${m[1]}"`,
  ],
  [
    /poison\s+target:(\d+)/i,
    (m, { agents } = {}) => {
      const name = agents?.[Number(m[1])] ?? `Agent ${m[1]}`;
      return `🧪 The witch's poison finds its mark — ${name}.`;
    },
  ],
  [
    /antidote/i,
    () => '✨ The witch uses the antidote. A life is saved.',
  ],
  [
    /skip/i,
    () => '— (action skipped)',
  ],
];

// Map event type/message patterns to narrative text
const EVENT_PATTERNS: Array<[RegExp | string, (msg: string) => string]> = [
  [/eliminated|killed|died/i,     (msg) => `💀 ${msg}`],
  [/won|victory|wins/i,           (msg) => `🏆 ${msg}`],
  [/phase.*night/i,               () => '🌙 Night descends upon the village...'],
  [/phase.*day/i,                 () => '🌅 The sun rises. The village gathers.'],
  [/phase.*vote/i,                () => '⚖️ The time for judgment is at hand.'],
  [/game.*over/i,                 (msg) => `🏁 ${msg}`],
  [/seer.*result/i,               (msg) => `🔮 The visions reveal: ${msg}`],
  [/protected/i,                  (msg) => `🛡 ${msg}`],
];

export function formatAction(rawAction: unknown, opts: FormatOptions = {}): string {
  if (!rawAction) return '';
  const str = typeof rawAction === 'string' ? rawAction : JSON.stringify(rawAction);

  for (const [pattern, fn] of ACTION_PATTERNS) {
    const m = str.match(pattern);
    if (m) return fn(m, opts);
  }

  // Fallback: clean up JSON to be readable
  if (str.startsWith('{') || str.startsWith('[')) {
    try {
      const obj = JSON.parse(str);
      if (obj.action) return `→ ${obj.action}${obj.target !== undefined ? ` (target: ${obj.target})` : ''}`;
    } catch { /* ignore */ }
  }

  return `→ ${str}`;
}

export function formatEventMessage(message: string): string {
  for (const [pattern, fn] of EVENT_PATTERNS) {
    if (typeof pattern === 'string' ? message.includes(pattern) : pattern.test(message)) {
      return fn(message);
    }
  }
  return message;
}

export function isDeathEvent(message: string): boolean {
  return /eliminat|killed|died|dead/i.test(message);
}

export function isPhaseChange(message: string): boolean {
  return /phase|night|morning|dawn|dusk/i.test(message);
}
