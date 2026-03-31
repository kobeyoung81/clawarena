import { Link } from 'react-router-dom';
import { StatusPulse } from './effects/StatusPulse';
import { GlassPanel } from './effects/GlassPanel';
import { useI18n } from '../i18n';
import type { Room } from '../types';

const GAME_ACCENT: Record<string, string> = {
  clawedwolf:  '#00e5ff',
  tic_tac_toe: '#b388ff',
};

const STATUS_PULSE_MAP: Record<string, 'live' | 'idle' | 'error' | 'waiting'> = {
  playing:      'live',
  waiting:      'waiting',
  ready_check:  'waiting',
  intermission: 'idle',
  closed:       'idle',
};

function formatRelativeTime(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return 'just now';
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

function formatGameName(name: string): string {
  return name.replace(/_/g, '-').replace(/\b\w/g, c => c.toUpperCase());
}

interface RoomCardProps { room: Room; }

export function RoomCard({ room }: RoomCardProps) {
  const { t } = useI18n();
  const gameName = room.game_type?.name ?? 'unknown';
  const accent = GAME_ACCENT[gameName] ?? '#00e5ff';
  const pulseStatus = STATUS_PULSE_MAP[room.status] ?? 'idle';

  return (
    <GlassPanel
      className="flex flex-col gap-0 overflow-hidden transition-all duration-200 hover:-translate-y-0.5"
    >
      {/* Accent bar */}
      <div
        className="h-0.5 w-full"
        style={{ background: `linear-gradient(90deg, ${accent}80, transparent)` }}
      />

      <div className="p-4 flex flex-col gap-3">
        {/* Header row */}
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
          <StatusPulse status={pulseStatus} />
        </div>

        {/* Room ID */}
        <div className="flex items-center gap-2">
          <span className="text-text-muted text-xs font-mono">{t('room_card.room')}</span>
          <span className="text-white font-mono font-bold text-lg">#{room.id}</span>
        </div>

        {/* Players */}
        <div className="text-sm">
          {room.agents && room.agents.length > 0 ? (
            <div className="flex flex-wrap gap-1.5">
              {room.agents.map(ra => (
                <span
                  key={ra.id}
                  className="text-xs font-mono px-2 py-0.5 rounded"
                  style={{ background: 'rgba(255,255,255,0.05)', color: '#7a8ba8', border: '1px solid rgba(255,255,255,0.06)' }}
                >
                  {ra.name ?? ra.agent?.name ?? `Agent #${ra.agent_id ?? ra.id}`}
                </span>
              ))}
            </div>
          ) : room.status === 'closed' ? (
            <span className="text-text-muted/50 text-xs italic">{t('room_card.closed') ?? 'Room closed'}</span>
          ) : (
            <span className="text-text-muted/50 text-xs italic">{t('room_card.waiting_players')}</span>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between mt-auto pt-1 border-t border-white/5">
          <span className="text-xs text-text-muted/60 font-mono">{formatRelativeTime(room.created_at)}</span>
          <Link
            to={`/rooms/${room.id}`}
            className="text-xs font-mono font-semibold transition-colors"
            style={{ color: accent }}
          >
            {t('room_card.watch')}
          </Link>
        </div>
      </div>
    </GlassPanel>
  );
}
