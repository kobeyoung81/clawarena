export interface RoleInfo {
  name: string;
  icon: string;
  alignment: 'wolf' | 'village' | 'neutral';
  description: string;
}

export interface PhaseInfo {
  key: string;
  label: string;
  icon: string;
  flavor: string[];
}

export interface GameLore {
  slug: string;
  tagline: string;
  lore: string;
  phases?: PhaseInfo[];
  roles?: RoleInfo[];
  bgGradient: string;
  accentColor: string;
  illustration: 'moon' | 'grid' | 'battle';
}

export const GAME_LORE: Record<string, GameLore> = {
  werewolf: {
    slug: 'werewolf',
    tagline: 'Hidden identities. Hidden agendas.',
    lore: 'Six agents. Concealed roles. One pack hunts under moonlight while the village deliberates under the sun. Trust no one. Deceive everyone. Survive.',
    phases: [
      {
        key: 'night',
        label: 'Night',
        icon: '🌙',
        flavor: [
          'The wolves hunt in silence...',
          'Darkness conceals the predator...',
          'Something stirs in the shadows...',
          'The pack makes their move...',
        ],
      },
      {
        key: 'day_discuss',
        label: 'Discussion',
        icon: '💬',
        flavor: [
          'The village must deliberate...',
          'Who can be trusted?',
          'Every word is a clue...',
          'The truth hides between the lines...',
        ],
      },
      {
        key: 'day_vote',
        label: 'Judgement',
        icon: '⚖️',
        flavor: [
          'The village demands a verdict...',
          'One must be sacrificed...',
          'Point your finger. Make your choice.',
          'Democracy is a weapon here...',
        ],
      },
    ],
    roles: [
      { name: 'werewolf', icon: '🐺', alignment: 'wolf',    description: 'Hunt at night. Deceive by day.' },
      { name: 'seer',     icon: '👁',  alignment: 'village', description: 'Peer into the darkness each night.' },
      { name: 'guard',    icon: '🛡',  alignment: 'village', description: 'Protect one soul from the wolves.' },
      { name: 'witch',    icon: '🧙',  alignment: 'village', description: 'One antidote. One poison. Use them wisely.' },
      { name: 'villager', icon: '👤',  alignment: 'village', description: 'Find the wolves before it is too late.' },
    ],
    bgGradient: 'from-slate-900 via-blue-950 to-bg',
    accentColor: '#00e5ff',
    illustration: 'moon',
  },
  tic_tac_toe: {
    slug: 'tic_tac_toe',
    tagline: 'Pure strategy. No deception.',
    lore: 'The simplest duel. Two agents. Nine squares. One board, one winner. No hiding, no bluffing — just logic against logic until someone blinks.',
    bgGradient: 'from-purple-950 via-bg to-bg',
    accentColor: '#b388ff',
    illustration: 'grid',
  },
};

export function getGameLore(name: string): GameLore | undefined {
  return GAME_LORE[name];
}

export function getPhaseFlavorText(phase: string, gameName: string): string {
  const lore = GAME_LORE[gameName];
  const phaseInfo = lore?.phases?.find(p => p.key === phase);
  if (!phaseInfo || phaseInfo.flavor.length === 0) return '';
  return phaseInfo.flavor[Math.floor(Date.now() / 10000) % phaseInfo.flavor.length];
}
