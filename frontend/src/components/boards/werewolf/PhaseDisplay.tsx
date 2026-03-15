import { getPhaseFlavorText } from '../../../data/gameLore';

interface PhaseDisplayProps {
  phase: string;
  round: number;
}

const PHASE_CONFIG: Record<string, { label: string; icon: string; color: string; glowColor: string }> = {
  night:      { label: 'Night',      icon: '🌙', color: '#00e5ff',  glowColor: 'rgba(0,229,255,0.2)' },
  day_discuss:{ label: 'Discussion', icon: '💬', color: '#ffc107',  glowColor: 'rgba(255,193,7,0.2)' },
  day_vote:   { label: 'Judgement',  icon: '⚖️', color: '#ff2d6b',  glowColor: 'rgba(255,45,107,0.2)' },
  game_over:  { label: 'Game Over',  icon: '🏁', color: '#888888',  glowColor: 'rgba(136,136,136,0.15)' },
};

export function PhaseDisplay({ phase, round }: PhaseDisplayProps) {
  const cfg = PHASE_CONFIG[phase] ?? PHASE_CONFIG.night;
  const flavor = getPhaseFlavorText(phase, 'werewolf');

  return (
    <div className="absolute inset-0 flex flex-col items-center justify-center pointer-events-none">
      {/* Atmospheric center glow */}
      <div
        className="absolute w-48 h-48 rounded-full"
        style={{
          background: `radial-gradient(circle, ${cfg.glowColor} 0%, transparent 70%)`,
        }}
      />

      {/* Phase icon */}
      <div
        className="text-4xl mb-2 relative"
        style={{ filter: `drop-shadow(0 0 8px ${cfg.color})` }}
      >
        {cfg.icon}
      </div>

      {/* Phase label */}
      <div
        className="font-display text-2xl font-bold opacity-40 uppercase tracking-widest"
        style={{ color: cfg.color }}
      >
        {cfg.label}
      </div>

      {/* Round counter */}
      <div className="font-mono text-xs text-text-muted/40 mt-1">
        Round {round}
      </div>

      {/* Flavor text */}
      {flavor && (
        <div className="font-mono text-xs italic text-text-muted/30 mt-2 max-w-[150px] text-center leading-tight">
          {flavor}
        </div>
      )}
    </div>
  );
}
