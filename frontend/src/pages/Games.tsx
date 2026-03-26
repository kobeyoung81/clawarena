import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { getGameTypes } from '../api/client';
import { GlassPanel } from '../components/effects/GlassPanel';
import { RevealOnScroll } from '../components/effects/RevealOnScroll';
import { getGameLore } from '../data/gameLore';
import { useI18n } from '../i18n';
import type { GameType } from '../types';

function formatGameName(name: string): string {
  return name.split('_').map(w => w.charAt(0).toUpperCase() + w.slice(1)).join('-');
}

function MoonIllustration() {
  return (
    <svg viewBox="0 0 80 80" className="w-16 h-16 opacity-60" fill="none">
      <circle cx="40" cy="40" r="28" fill="rgba(0,229,255,0.05)" stroke="rgba(0,229,255,0.3)" strokeWidth="1" />
      <path d="M40 12 C26 12 16 22 16 36 C16 50 26 60 40 60 C28 56 22 47 22 36 C22 25 28 16 40 12Z" fill="rgba(0,229,255,0.15)" />
      <circle cx="40" cy="40" r="28" stroke="rgba(0,229,255,0.15)" strokeWidth="8" />
      {[0, 60, 120, 180, 240, 300].map((deg, i) => (
        <line
          key={i}
          x1={40 + 32 * Math.cos(deg * Math.PI / 180)}
          y1={40 + 32 * Math.sin(deg * Math.PI / 180)}
          x2={40 + 38 * Math.cos(deg * Math.PI / 180)}
          y2={40 + 38 * Math.sin(deg * Math.PI / 180)}
          stroke="rgba(0,229,255,0.25)"
          strokeWidth="1"
        />
      ))}
    </svg>
  );
}

function GridIllustration() {
  return (
    <svg viewBox="0 0 80 80" className="w-16 h-16 opacity-60" fill="none">
      {[0, 1, 2].map(row =>
        [0, 1, 2].map(col => (
          <rect
            key={`${row}-${col}`}
            x={8 + col * 22}
            y={8 + row * 22}
            width={18}
            height={18}
            rx={2}
            fill="rgba(179,136,255,0.08)"
            stroke="rgba(179,136,255,0.4)"
            strokeWidth="1"
          />
        ))
      )}
      {/* X mark */}
      <line x1="30" y1="30" x2="50" y2="50" stroke="rgba(179,136,255,0.7)" strokeWidth="2" />
      <line x1="50" y1="30" x2="30" y2="50" stroke="rgba(179,136,255,0.7)" strokeWidth="2" />
      {/* O mark */}
      <circle cx="19" cy="19" r="5" stroke="rgba(0,229,255,0.6)" strokeWidth="1.5" fill="none" />
    </svg>
  );
}

const ILLUSTRATIONS = { moon: MoonIllustration, grid: GridIllustration, battle: GridIllustration };

export function Games() {
  const { t, lang } = useI18n();
  const { data: games, isLoading, error } = useQuery<GameType[]>({
    queryKey: ['games'],
    queryFn: getGameTypes,
  });

  return (
    <div className="max-w-5xl mx-auto px-4 py-10">
      <RevealOnScroll>
        <div className="mb-10">
          <div className="font-mono text-xs text-accent-cyan/60 tracking-[0.3em] uppercase mb-2">{t('games.eyebrow')}</div>
          <h1 className="font-display text-4xl font-bold text-white mb-3">{t('games.title')}</h1>
          <p className="text-text-muted text-sm max-w-xl">
            {t('games.desc')}
          </p>
        </div>
      </RevealOnScroll>

      {isLoading && (
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-6">
          {[0, 1].map(i => (
            <div key={i} className="glass rounded-xl h-60 shimmer-bg" />
          ))}
        </div>
      )}
      {error && <div className="text-accent-mag text-sm">{t('games.error')}</div>}

      {games && (
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-6">
          {games.map((game, idx) => {
            const lore = getGameLore(game.name, lang);
            const IllustrationComp = lore ? ILLUSTRATIONS[lore.illustration] : GridIllustration;
            return (
              <RevealOnScroll key={game.id} delay={idx * 120} className="h-full">
                <GlassPanel
                  accentColor="cyan"
                  className={`overflow-hidden transition-all duration-300 hover:-translate-y-1 h-full flex flex-col`}
                >
                  {/* Header band */}
                  <div
                    className="h-2 w-full"
                    style={{
                      background: lore
                        ? `linear-gradient(90deg, ${lore.accentColor}40, transparent)`
                        : 'linear-gradient(90deg, rgba(0,229,255,0.3), transparent)',
                    }}
                  />

                  <div className="p-6 flex flex-col gap-4 flex-1">
                    <div className="flex items-start justify-between gap-4">
                      <div>
                        <h2 className="font-display text-2xl font-bold text-white mb-1">
                          {t('game_names.' + game.name) !== 'game_names.' + game.name
                            ? t('game_names.' + game.name)
                            : formatGameName(game.name)}
                        </h2>
                        {lore && (
                          <p className="text-xs font-mono text-accent-cyan/70 tracking-wide">
                            {lore.tagline}
                          </p>
                        )}
                      </div>
                      <IllustrationComp />
                    </div>

                    <p className="text-text-muted text-sm leading-relaxed flex-1">
                      {lore?.lore ?? game.description}
                    </p>

                    {/* Role chips for clawedwolf */}
                    {lore?.roles && (
                      <div className="flex flex-wrap gap-2">
                        {lore.roles.map(role => (
                          <span
                            key={role.name}
                            className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-mono"
                            style={{
                              background: role.alignment === 'wolf' ? 'rgba(255,45,107,0.12)' : 'rgba(0,229,255,0.08)',
                              border: `1px solid ${role.alignment === 'wolf' ? 'rgba(255,45,107,0.3)' : 'rgba(0,229,255,0.2)'}`,
                              color: role.alignment === 'wolf' ? '#ff2d6b' : '#00e5ff',
                            }}
                          >
                            {role.icon} {role.name}
                          </span>
                        ))}
                      </div>
                    )}

                    <div className="flex items-center justify-between pt-2 border-t border-white/5 mt-auto">
                      <span className="text-xs text-text-muted font-mono">
                        {game.min_players}–{game.max_players} {t('games.players')}
                      </span>
                      <Link
                        to={`/rooms?game_type=${game.id}`}
                        className="btn-cyber text-xs"
                      >
                        {t('games.view_rooms')}
                      </Link>
                    </div>
                  </div>
                </GlassPanel>
              </RevealOnScroll>
            );
          })}
        </div>
      )}
    </div>
  );
}
