import { useEffect, useState } from 'react';

interface PhaseTransitionOverlayProps {
  phase: string;
  round?: number;
}

const PHASE_CONFIG: Record<string, { label: string; icon: string; bg: string; color: string }> = {
  night: {
    label: 'Night Falls',
    icon: '🌙',
    bg: 'radial-gradient(ellipse at center, rgba(0,20,60,0.95) 0%, rgba(10,14,26,0.98) 100%)',
    color: '#00e5ff',
  },
  day_discuss: {
    label: 'Dawn Breaks',
    icon: '🌅',
    bg: 'radial-gradient(ellipse at center, rgba(60,40,0,0.95) 0%, rgba(10,14,26,0.98) 100%)',
    color: '#ffc107',
  },
  day_vote: {
    label: 'Judgement Comes',
    icon: '⚖️',
    bg: 'radial-gradient(ellipse at center, rgba(60,0,20,0.95) 0%, rgba(10,14,26,0.98) 100%)',
    color: '#ff2d6b',
  },
  game_over: {
    label: 'Game Over',
    icon: '🏁',
    bg: 'radial-gradient(ellipse at center, rgba(20,20,20,0.98) 0%, rgba(10,14,26,0.99) 100%)',
    color: '#888',
  },
};

export function PhaseTransitionOverlay({ phase, round }: PhaseTransitionOverlayProps) {
  const [show, setShow] = useState(true);
  const cfg = PHASE_CONFIG[phase] ?? PHASE_CONFIG.night;

  useEffect(() => {
    setShow(true);
    const t = setTimeout(() => setShow(false), 2000);
    return () => clearTimeout(t);
  }, [phase, round]);

  if (!show) return null;

  return (
    <div
      className="absolute inset-0 z-50 flex flex-col items-center justify-center pointer-events-none"
      style={{
        background: cfg.bg,
        animation: 'phaseTransition 2s ease both',
      }}
    >
      <div className="text-6xl mb-3">{cfg.icon}</div>
      <div
        className="font-display text-2xl font-bold tracking-widest uppercase"
        style={{ color: cfg.color, textShadow: `0 0 20px ${cfg.color}` }}
      >
        {cfg.label}
      </div>
      {round !== undefined && (
        <div className="text-text-muted font-mono text-sm mt-2">Round {round}</div>
      )}
    </div>
  );
}
