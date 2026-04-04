import { useI18n } from '../i18n';
import { StatusPulse } from './effects/StatusPulse';
import { ParticleCanvas } from './effects/ParticleCanvas';
import type { Room } from '../types';

function formatGameName(name: string): string {
  return name.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase());
}

/** Map a raw game_type name to an i18n key, falling back to formatted name */
function localizedGameName(name: string, t: (key: string) => string): string {
  const key = `game_names.${name}`;
  const translated = t(key);
  return translated !== key ? translated : formatGameName(name);
}

/** Translate team name (evil/good) via i18n, falling back to raw value */
function localizedTeamName(team: string, t: (key: string) => string): string {
  const key = `teams.${team}`;
  const translated = t(key);
  return translated !== key ? translated : team;
}

const STATUS_COLOR: Record<string, string> = {
  waiting:      'rgba(255,255,255,0.3)',
  ready_check:  '#ffc107',
  playing:      '#00e676',
  intermission: '#ffc107',
  closed:       'rgba(255,255,255,0.2)',
};

export function RoomHeader({ room, isReplayMode, isConnected }: {
  room: Room;
  isReplayMode: boolean;
  isConnected?: boolean;
}) {
  const { t } = useI18n();
  const statusColor = STATUS_COLOR[room.status] ?? 'rgba(255,255,255,0.3)';
  const statusLabel = t(`status.${room.status}`) !== `status.${room.status}`
    ? t(`status.${room.status}`)
    : room.status;

  return (
    <div
      className="relative rounded-xl overflow-hidden mb-5"
      style={{
        background: 'rgba(10,14,26,0.7)',
        border: `1px solid ${statusColor}22`,
        backdropFilter: 'blur(10px)',
      }}
    >
      {/* Subtle accent line at top */}
      <div className="h-px w-full" style={{ background: `linear-gradient(90deg, transparent, ${statusColor}60, transparent)` }} />

      <div className="px-5 py-4 flex flex-wrap items-center gap-4">
        {/* Game type + room number */}
        <div>
          <h1 className="text-xl font-bold tracking-tight text-text-primary">
            {localizedGameName(room.game_type?.name ?? '', t)}
            <span className="ml-2 text-text-muted/40 font-mono text-base font-normal">
              #{room.id}
            </span>
          </h1>
          <div className="flex items-center gap-3 mt-1">
            {/* Status pill */}
            <div
              className="flex items-center gap-1.5 text-[10px] font-mono font-semibold uppercase tracking-widest px-2 py-0.5 rounded"
              style={{ background: `${statusColor}14`, color: statusColor, border: `1px solid ${statusColor}30` }}
            >
              {room.status === 'playing' && (
                <span className="w-1.5 h-1.5 rounded-full animate-ping-slow" style={{ background: statusColor }} />
              )}
              {statusLabel}
            </div>

            {/* Replay badge */}
            {isReplayMode && (
              <span className="text-[10px] font-mono text-text-muted/50 bg-white/4 px-2 py-0.5 rounded border border-white/8">
                {t('observer.replay_badge')}
              </span>
            )}

            {/* Live connection indicator */}
            {!isReplayMode && (
              <StatusPulse
                status={isConnected ? 'live' : 'waiting'}
                label={isConnected ? t('observer.connected') : t('observer.reconnecting')}
              />
            )}
          </div>
        </div>

        {/* Spacer */}
        <div className="flex-1" />

        {/* Player count */}
        <div className="text-right">
          <div className="text-xs font-mono text-text-muted/40">{t('observer.players_label')}</div>
          <div className="text-lg font-mono font-bold text-text-primary">{room.agents.length}</div>
        </div>
      </div>
    </div>
  );
}

export function ResultBanner({ winner_team }: { winner_team?: string }) {
  const { t } = useI18n();
  return (
    <div
      className="relative rounded-xl overflow-hidden mb-4"
      style={{
        background: 'rgba(0,229,255,0.05)',
        border: '1px solid rgba(0,229,255,0.25)',
      }}
    >
      <ParticleCanvas density={15} speed={0.2} color="#00e5ff" className="opacity-20 rounded-xl" />
      <div className="relative z-10 py-5 text-center">
        <div className="text-2xl font-bold tracking-tight text-text-primary">
          {winner_team ? t('observer.victory', { team: localizedTeamName(winner_team, t) }) : t('observer.game_over')}
        </div>
        <div className="text-xs font-mono text-accent-cyan/60 mt-1 uppercase tracking-widest">
          {winner_team ? t('observer.winner_declared') : t('observer.match_concluded')}
        </div>
      </div>
    </div>
  );
}
