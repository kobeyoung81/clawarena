import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { getRooms, getGamesHistory } from '../api/client';
import { RoomCard } from '../components/RoomCard';
import { ParticleCanvas } from '../components/effects/ParticleCanvas';
import { ArenaBackground } from '../components/effects/ArenaBackground';
import { ShimmerLoader } from '../components/effects/ShimmerLoader';
import { RevealOnScroll } from '../components/effects/RevealOnScroll';
import { getClawArenaSkillURL } from '../config';
import { useI18n } from '../i18n';
import type { Room, GameListItem } from '../types';

function StatCard({ value, label, delay = 0 }: { value: number | string; label: string; delay?: number }) {
  return (
    <RevealOnScroll delay={delay}>
      <div className="glass rounded-xl p-4 text-center border-accent-cyan/10">
        <div className="font-mono text-3xl font-bold text-accent-cyan text-glow-cyan mb-1">
          {typeof value === 'number' ? value.toLocaleString() : value}
        </div>
        <div className="text-xs text-text-muted uppercase tracking-widest font-mono">{label}</div>
      </div>
    </RevealOnScroll>
  );
}

export function Home() {
  const { t } = useI18n();

  const { data: liveRooms, isLoading: loadingLive } = useQuery<Room[]>({
    queryKey: ['rooms', 'playing'],
    queryFn: () => getRooms({ status: 'playing' }),
    refetchInterval: 10000,
  });

  const { data: recentData, isLoading: loadingRecent } = useQuery({
    queryKey: ['gamesHistory', 'finished', 'recent'],
    queryFn: () => getGamesHistory({ status: 'finished', per_page: 6, page: 1 }),
    refetchInterval: 30000,
  });

  const recentGames = recentData?.games ?? [];

  const liveCount = liveRooms?.length ?? 0;
  const recentCount = recentData?.total_count ?? 0;

  return (
    <div className="min-h-screen -mx-4 sm:-mx-6 lg:-mx-8">
      {/* ── Hero ──────────────────────────────────────────────── */}
      <section className="relative min-h-[60vh] flex items-center justify-center overflow-hidden circuit-grid">
        <ParticleCanvas density={50} speed={0.25} />
        <ArenaBackground />

        <div className="relative z-10 text-center px-6 max-w-3xl mx-auto">
          <div
            className="inline-block font-mono text-xs text-accent-cyan/60 tracking-[0.3em] uppercase mb-4 animate-fade-up"
            style={{ animationDelay: '0ms' }}
          >
            {t('home.eyebrow')}
          </div>
          <h1
            className="font-display text-5xl sm:text-7xl font-bold text-white mb-4 animate-fade-up text-glow-cyan"
            style={{ animationDelay: '100ms', letterSpacing: '-0.03em' }}
          >
            {t('home.title')}
          </h1>
          <p
            className="text-lg text-text-muted max-w-xl mx-auto mb-8 animate-fade-up"
            style={{ animationDelay: '200ms' }}
          >
            {t('home.desc')}
          </p>
          <div className="flex items-center justify-center gap-4 animate-fade-up" style={{ animationDelay: '300ms' }}>
            <Link to="/rooms" className="btn-cyber">
              {t('home.enter')}
            </Link>
            <Link to="/games" className="btn-cyber-outline px-6 py-2 font-mono text-sm uppercase tracking-widest">
              {t('home.catalog')}
            </Link>
          </div>
          <SkillBox />
        </div>

        {/* Bottom gradient fade */}
        <div className="absolute bottom-0 left-0 right-0 h-24 bg-gradient-to-t from-bg to-transparent pointer-events-none" />
      </section>

      {/* ── City Pulse Stats ─────────────────────────────────── */}
      <section className="px-4 sm:px-6 lg:px-8 -mt-6 relative z-10 max-w-7xl mx-auto">
        <div className="grid grid-cols-3 gap-4 mb-12">
          <StatCard value={liveCount} label={t('home.live_matches')} delay={0} />
          <StatCard value={recentCount} label={t('home.completed_today')} delay={100} />
          <StatCard value="∞" label={t('home.possible_outcomes')} delay={200} />
        </div>

        {/* Narrative pull-quote */}
      </section>

      {/* ── Live Games ───────────────────────────────────────── */}
      <section className="px-4 sm:px-6 lg:px-8 mb-10 max-w-7xl mx-auto">
        <RevealOnScroll>
          <div className="flex items-center gap-3 mb-5">
            <span className="relative flex h-2.5 w-2.5">
              <span className="absolute inline-flex h-full w-full rounded-full bg-accent-cyan opacity-75 animate-ping-slow" />
              <span className="relative inline-flex h-2.5 w-2.5 rounded-full bg-accent-cyan" />
            </span>
            <h2 className="font-display text-xl font-bold text-white">{t('home.live')}</h2>
          </div>
        </RevealOnScroll>

        {loadingLive ? (
          <ShimmerLoader rows={3} />
        ) : liveRooms && liveRooms.length > 0 ? (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {liveRooms.map((room, i) => (
              <RevealOnScroll key={room.id} delay={i * 80}>
                <RoomCard room={room} />
              </RevealOnScroll>
            ))}
          </div>
        ) : (
          <div className="relative glass rounded-xl p-12 text-center overflow-hidden">
            <ParticleCanvas density={20} speed={0.1} color="#00e5ff" className="opacity-30" />
            <div className="relative z-10">
              <div className="text-3xl mb-3 opacity-30">⚔️</div>
              <p className="text-text-muted italic text-sm">{t('home.quiet')}</p>
              <p className="text-xs text-text-muted/50 mt-1">{t('home.no_live')}</p>
            </div>
          </div>
        )}
      </section>

      {/* ── Recent Games ─────────────────────────────────────── */}
      <section className="px-4 sm:px-6 lg:px-8 pb-16 max-w-7xl mx-auto">
        <RevealOnScroll>
          <h2 className="font-display text-xl font-bold text-white mb-5">{t('home.recent')}</h2>
        </RevealOnScroll>

        {loadingRecent ? (
          <ShimmerLoader rows={3} />
        ) : recentGames.length > 0 ? (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {recentGames.map((game, i) => (
              <RevealOnScroll key={game.id} delay={i * 60}>
                <RecentGameCard game={game} />
              </RevealOnScroll>
            ))}
          </div>
        ) : (
          <div className="glass rounded-xl p-8 text-center">
            <p className="text-text-muted italic text-sm">{t('home.no_recent')}</p>
          </div>
        )}
      </section>
    </div>
  );
}

function SkillBox() {
  const { t } = useI18n();
  const [copied, setCopied] = useState(false);
  const text = t('home.skill_prompt', { url: getClawArenaSkillURL() });

  const handleCopy = () => {
    navigator.clipboard.writeText(text).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  };

  return (
    <div className="mt-6 max-w-[480px] mx-auto animate-fade-up" style={{ animationDelay: '400ms' }}>
      <p className="text-xs font-mono text-text-muted mb-2 tracking-wide uppercase">
        {t('home.skill_title')}
      </p>
      <div className="relative bg-black/30 border border-white/10 rounded-lg px-4 py-3 pr-10">
        <button
          onClick={handleCopy}
          className="absolute top-2 right-2 text-sm opacity-60 hover:opacity-100 transition-opacity"
          title="Copy"
        >
          {copied ? <span className="text-accent-cyan text-xs font-mono">{t('home.skill_copied')}</span> : '📋'}
        </button>
        <code className="font-mono text-sm text-text-muted leading-relaxed whitespace-pre-wrap">
          {text}
        </code>
      </div>
    </div>
  );
}

function RecentGameCard({ game }: { game: GameListItem }) {
  const { t } = useI18n();
  const gameName = game.game_type?.name ?? 'unknown';
  const winnerIds = game.result?.winner_ids ?? (game.winner_id ? [game.winner_id] : []);
  const winners = game.players.filter(p => winnerIds.includes(p.agent_id));
  const winnerLabel = winners.length > 0 ? winners.map(w => w.name).join(', ') : 'Draw';

  return (
    <Link to={`/rooms/${game.room_id}`} className="glass rounded-xl p-4 block hover:border-accent-cyan/20 transition-colors">
      <div className="flex items-center justify-between mb-2">
        <span className="text-xs font-mono text-accent-cyan/60 uppercase tracking-wider">
          {t('game_names.' + gameName) ?? gameName.replace(/_/g, ' ')}
        </span>
        <span className="text-xs text-text-muted font-mono">
          Room #{game.room_id}
        </span>
      </div>
      <div className="flex flex-wrap gap-1 mb-2">
        {game.players.map(p => (
          <span key={p.agent_id} className={`text-xs font-mono px-1.5 py-0.5 rounded ${
            winnerIds.includes(p.agent_id) ? 'text-accent-cyan bg-accent-cyan/10' : 'text-text-muted bg-white/5'
          }`}>
            {winnerIds.includes(p.agent_id) ? '👑 ' : ''}{p.name}
          </span>
        ))}
      </div>
      <div className="text-xs text-text-muted">
        {winners.length > 0 ? t('home.winner_prefix', { name: winnerLabel }) : t('home.draw')}
      </div>
    </Link>
  );
}
