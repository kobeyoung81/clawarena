import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { getGamesHistory, getGameTypes } from '../api/client';
import { GlassPanel } from '../components/effects/GlassPanel';
import { ShimmerCard } from '../components/effects/ShimmerLoader';
import { StatusPulse } from '../components/effects/StatusPulse';
import { useI18n } from '../i18n';
import type { GameListItem, GameType } from '../types';

const GAME_ACCENT: Record<string, string> = {
  clawedwolf:  '#00e5ff',
  tic_tac_toe: '#b388ff',
  clawed_roulette: '#ff9800',
};

const PER_PAGE = 12;

function formatGameName(name: string): string {
  return name.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase());
}

function formatTimestamp(dateStr: string): string {
  const d = new Date(dateStr);
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
    + ', '
    + d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' });
}

export function Replays() {
  const { t } = useI18n();
  const [gameTypeId, setGameTypeId] = useState<string>('');
  const [page, setPage] = useState(1);

  const queryParams = {
    status: 'finished' as const,
    ...(gameTypeId ? { game_type_id: Number(gameTypeId) } : {}),
    page,
    per_page: PER_PAGE,
  };

  const { data, isLoading, error } = useQuery({
    queryKey: ['gamesHistory', queryParams],
    queryFn: () => getGamesHistory(queryParams),
  });

  const games = data?.games;

  const { data: gameTypes } = useQuery<GameType[]>({
    queryKey: ['games'],
    queryFn: getGameTypes,
  });

  const hasMore = games?.length === PER_PAGE;

  return (
    <div className="max-w-5xl mx-auto px-4 py-10">
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-white flex items-center gap-3">
          <span>{t('replays.icon')}</span>
          <span>{t('replays.title')}</span>
        </h1>
      </div>

      {/* Filters */}
      <GlassPanel className="p-4 mb-6">
        <div className="flex flex-wrap gap-3">
          <select
            value={gameTypeId}
            onChange={e => { setGameTypeId(e.target.value); setPage(1); }}
            className="bg-gray-700 text-white text-sm px-3 py-2 rounded border border-gray-600 focus:border-blue-500 focus:outline-none"
          >
            <option value="">{t('replays.all_games')}</option>
            {gameTypes?.map(g => (
              <option key={g.id} value={String(g.id)}>{t('game_names.' + g.name) ?? g.name}</option>
            ))}
          </select>
        </div>
      </GlassPanel>

      {/* Loading */}
      {isLoading && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <ShimmerCard key={i} />
          ))}
        </div>
      )}

      {/* Error */}
      {error && <div className="text-red-400">{t('replays.error')}</div>}

      {/* Game list */}
      {games && games.length > 0 ? (
        <>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {games.map(game => (
              <GameCard key={game.id} game={game} />
            ))}
          </div>

          {/* Pagination */}
          <div className="flex items-center justify-center gap-4 mt-8">
            {page > 1 && (
              <button
                onClick={() => setPage(p => p - 1)}
                className="text-sm font-mono px-4 py-2 rounded border border-white/10 text-text-muted hover:text-white hover:border-white/20 transition-colors"
              >
                ← Prev
              </button>
            )}
            <span className="text-xs font-mono text-text-muted">
              Page {page}
            </span>
            {hasMore && (
              <button
                onClick={() => setPage(p => p + 1)}
                className="text-sm font-mono px-4 py-2 rounded border border-accent-cyan/20 text-accent-cyan hover:bg-accent-cyan/10 transition-colors"
              >
                {t('replays.load_more')} →
              </button>
            )}
          </div>
        </>
      ) : !isLoading ? (
        <div className="text-center py-16">
          <p className="text-text-muted text-lg">{t('replays.no_games')}</p>
          <p className="text-text-muted/50 text-sm mt-2">{t('replays.no_games_hint')}</p>
        </div>
      ) : null}
    </div>
  );
}

function GameCard({ game }: { game: GameListItem }) {
  const { t } = useI18n();
  const gameName = game.game_type?.name ?? 'unknown';
  const accent = GAME_ACCENT[gameName] ?? '#00e5ff';

  const winnerIds = game.result?.winner_ids ?? (game.winner_id ? [game.winner_id] : []);
  const winners = game.players.filter(p => winnerIds.includes(p.agent_id));

  return (
    <GlassPanel className="flex flex-col gap-0 overflow-hidden transition-all duration-200 hover:-translate-y-0.5 h-full">
      {/* Accent bar */}
      <div
        className="h-0.5 w-full"
        style={{ background: `linear-gradient(90deg, ${accent}80, transparent)` }}
      />

      <div className="p-4 flex flex-col gap-3 flex-1">
        {/* Header */}
        <div className="flex items-center justify-between">
          <span
            className="text-xs font-mono font-semibold px-2 py-0.5 rounded-full"
            style={{
              background: `${accent}15`,
              border: `1px solid ${accent}30`,
              color: accent,
            }}
          >
            {t('game_names.' + gameName) ?? formatGameName(gameName)}
          </span>
          <StatusPulse status="idle" label="FINISHED" />
        </div>

        {/* Room ID */}
        <div className="flex items-center gap-2">
          <span className="text-text-muted text-xs font-mono">{t('replays.room')}</span>
          <span className="text-white font-mono font-bold text-lg">#{game.room_id}</span>
        </div>

        {/* Players */}
        <div className="flex flex-wrap gap-1.5">
          {game.players.map(p => {
            const isWinner = winnerIds.includes(p.agent_id);
            return (
              <span
                key={p.agent_id}
                className="text-xs font-mono px-2 py-0.5 rounded"
                style={{
                  background: isWinner ? 'rgba(0,229,255,0.1)' : 'rgba(255,255,255,0.05)',
                  color: isWinner ? '#00e5ff' : '#7a8ba8',
                  border: isWinner ? '1px solid rgba(0,229,255,0.3)' : '1px solid rgba(255,255,255,0.06)',
                }}
              >
                {isWinner ? '👑 ' : ''}{p.name}
              </span>
            );
          })}
        </div>

        {/* Winner line */}
        {winners.length > 0 && game.result?.winner_team && (
          <div className="text-xs font-mono text-accent-cyan/80">
            {t('replays.winner')}: {game.result.winner_team}
          </div>
        )}

        {/* Timestamps & action */}
        <div className="flex items-center justify-between mt-auto pt-2 border-t border-white/5">
          <div className="flex flex-col gap-0.5">
            {game.started_at && (
              <span className="text-xs text-text-muted/60 font-mono">
                {formatTimestamp(game.started_at)}
              </span>
            )}
          </div>
          <Link
            to={`/rooms/${game.room_id}?game=${game.id}`}
            className="text-xs font-mono font-semibold transition-colors"
            style={{ color: accent }}
          >
            {t('replays.watch_replay')}
          </Link>
        </div>
      </div>
    </GlassPanel>
  );
}
