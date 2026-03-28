import type { ClawedWolfPlayer } from '../../../types';

interface PlayerSeatProps {
  player: ClawedWolfPlayer;
  isCurrentSpeaker: boolean;
  voteCount?: number;
  isNight: boolean;
  isReplay: boolean;
  phase: string;
  style: React.CSSProperties;
  speech?: string;
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

export function PlayerSeat({ player, isCurrentSpeaker, voteCount, isNight, isReplay, phase, style, speech }: PlayerSeatProps) {
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
      className="absolute flex flex-col items-center gap-1 cursor-default group"
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
          relative w-14 h-14 rounded-full flex items-center justify-center text-xl
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
          <span className="text-white text-xs font-mono font-bold">{player.seat ?? '?'}</span>
        )}
      </div>

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

      {/* Name label */}
      <div className="text-center max-w-[70px]">
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

      {/* Speech balloon */}
      {speech && isCurrentSpeaker && (
        <div
          className="absolute top-full mt-1 max-w-[130px] text-[9px] text-white/80 bg-surface/90 border border-white/10 rounded-lg px-2 py-1 pointer-events-none backdrop-blur-sm"
          style={{ left: '50%', transform: 'translateX(-50%)' }}
        >
          <div className="truncate" title={speech}>
            &ldquo;{speech.length > 60 ? speech.slice(0, 60) + '...' : speech}&rdquo;
          </div>
        </div>
      )}
    </div>
  );
}
