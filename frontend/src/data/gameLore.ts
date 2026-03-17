import { en, type TranslationKeys } from '../i18n/en';
import { zh } from '../i18n/zh';

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

type Lang = 'en' | 'zh';
const translations: Record<Lang, TranslationKeys> = { en, zh };

function buildGameLore(lang: Lang): Record<string, GameLore> {
  const t = translations[lang];
  return {
    werewolf: {
      slug: 'werewolf',
      tagline: t.game_lore.werewolf.tagline,
      lore: t.game_lore.werewolf.lore,
      phases: [
        {
          key: 'night',
          label: t.phase_label.night,
          icon: '🌙',
          flavor: [...t.game_lore.flavor.night],
        },
        {
          key: 'day_discuss',
          label: t.phase_label.day_discuss,
          icon: '💬',
          flavor: [...t.game_lore.flavor.day_discuss],
        },
        {
          key: 'day_vote',
          label: t.phase_label.day_vote,
          icon: '⚖️',
          flavor: [...t.game_lore.flavor.day_vote],
        },
      ],
      roles: [
        { name: 'werewolf', icon: '🐺', alignment: 'wolf',    description: t.game_lore.roles.werewolf },
        { name: 'seer',     icon: '👁',  alignment: 'village', description: t.game_lore.roles.seer },
        { name: 'guard',    icon: '🛡',  alignment: 'village', description: t.game_lore.roles.guard },
        { name: 'witch',    icon: '🧙',  alignment: 'village', description: t.game_lore.roles.witch },
        { name: 'villager', icon: '👤',  alignment: 'village', description: t.game_lore.roles.villager },
      ],
      bgGradient: 'from-slate-900 via-blue-950 to-bg',
      accentColor: '#00e5ff',
      illustration: 'moon',
    },
    tic_tac_toe: {
      slug: 'tic_tac_toe',
      tagline: t.game_lore.tic_tac_toe.tagline,
      lore: t.game_lore.tic_tac_toe.lore,
      bgGradient: 'from-purple-950 via-bg to-bg',
      accentColor: '#b388ff',
      illustration: 'grid',
    },
  };
}

// Default English lore for non-React contexts
const GAME_LORE = buildGameLore('en');

export function getGameLore(name: string, lang?: Lang): GameLore | undefined {
  if (lang) {
    return buildGameLore(lang)[name];
  }
  return GAME_LORE[name];
}

export function getPhaseFlavorText(phase: string, gameName: string, lang?: Lang): string {
  const lore = lang ? buildGameLore(lang) : GAME_LORE;
  const gameLore = lore[gameName];
  const phaseInfo = gameLore?.phases?.find(p => p.key === phase);
  if (!phaseInfo || phaseInfo.flavor.length === 0) return '';
  return phaseInfo.flavor[Math.floor(Date.now() / 10000) % phaseInfo.flavor.length];
}
