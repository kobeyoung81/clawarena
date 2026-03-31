import type { ClawedWolfPlayer } from '../../../types';

interface PlayerSeatProps {
  player: ClawedWolfPlayer;
  isCurrentSpeaker: boolean;
  voteCount?: number;
  isNight: boolean;
  isReplay: boolean;
  phase: string;
  style?: React.CSSProperties;
  className?: string;
}

const ROLE_EMOJI: Record<string, string> = {
  clawedwolf: '🐺',
  seer:       '👁',
  guard:      '🛡',
  villager:   '👤',
  witch:      '🧙',
};

const ROLE_COLORS: Record<string, string> = {
  clawedwolf: '#ff2d6b',
  seer:     '#b388ff',
  guard:    '#00e676',
  villager: '#64b5f6',
  witch:    '#e040fb',
};

/** Get a short display label for the avatar circle */
function avatarLabel(player: ClawedWolfPlayer): string {
  if (player.name) {
    return player.name.slice(0, 2).toUpperCase();
  }
  return `P${player.seat}`;
}

export function PlayerSeat({ player, isCurrentSpeaker, voteCount, isNight, isReplay, phase, style, className }: PlayerSeatProps) {
  const isAlive = player.alive;
  const role = isReplay ? player.role : undefined;
  const roleColor = role ? (ROLE_COLORS[role] ?? '#00e5ff') : '#00e5ff';

  const ringStyle = (() => {
    if (!isAlive) return 'ring-1 ring-gray-600';
    if (isCurrentSpeaker) return 'ring-2 ring-accent-cyan animate-speaker';
    if (role === 'clawedwolf') return 'ring-2 ring-accent-mag/60';
    return 'ring-1 ring-white/10';
  })();

  const bgStyle = (() => {
    if (!isAlive) return 'bg-gray-900/80';
    if (isCurrentSpeaker) return 'bg-accent-cyan/10';
    if (isNight) return 'bg-blue-950/80';
    if (phase === 'day_vote') return 'bg-red-950/60';
    return 'bg-amber-950/60';
  })();

  return (
    <div
      className={`flex flex-col items-center gap-1 cursor-default group ${className ?? ''}`}
      style={style}
    >
      {/* Speaker spotlight */}
      {isCurrentSpeaker && (
        <div
          className="absolute -inset-8 rounded-full pointer-events-none"
          style={{
            background: 'radial-gradient(circle, rgba(0,229,255,0.12) 0%, transparent 70%)',
          }}
        />
      )}

      {/* Avatar circle */}
      <div
        className={`
          relative w-12 h-12 rounded-full flex items-center justify-center text-lg
          ${bgStyle} ${ringStyle}
          transition-all duration-500
          ${!isAlive ? 'opacity-40' : isAlive && isCurrentSpeaker ? 'scale-110' : 'animate-breathe'}
          ${!isAlive ? 'grayscale' : ''}
        `}
        style={isCurrentSpeaker ? { boxShadow: `0 0 16px rgba(0,229,255,0.4)` } : undefined}
        title={isReplay && role ? role : undefined}
      >
        {!isAlive ? (
          <span className="text-gray-500 text-lg">☠</span>
        ) : role ? (
          <span style={{ filter: `drop-shadow(0 0 4px ${roleColor})` }}>{ROLE_EMOJI[role] ?? '👤'}</span>
        ) : (
          <span className="text-white text-[10px] font-mono font-bold leading-tight text-center">
            {avatarLabel(player)}
          </span>
        )}

        {/* Vote count badge */}
        {voteCount !== undefined && voteCount > 0 && (
          <div
            className="absolute -top-1 -right-1 w-5 h-5 rounded-full flex items-center justify-center text-xs font-bold"
            style={{
              background: '#ff2d6b',
              boxShadow: '0 0 6px rgba(255,45,107,0.6)',
              animation: 'voteReveal 0.4s ease both',
            }}
          >
            {voteCount}
          </div>
        )}
      </div>

      {/* Name label — supports up to 20 characters */}
      <div className="text-center max-w-[140px]">
        <div
          className={`text-[11px] font-medium truncate ${isAlive ? 'text-white' : 'text-gray-600'}`}
          title={player.name}
        >
          {player.name ?? `P${player.seat}`}
        </div>
        {isReplay && role && (
          <div className="text-[9px] font-mono" style={{ color: roleColor }}>
            {role}
          </div>
        )}
      </div>
    </div>
  );
}
