import { useEffect, useState } from 'react';
import { useI18n } from '../../i18n';

interface PhaseTransitionOverlayProps {
  phase: string;
  round?: number;
}

const PHASE_STYLES: Record<string, { icon: string; bg: string; color: string }> = {
  night: {
    icon: '🌙',
    bg: 'radial-gradient(ellipse at center, rgba(0,20,60,0.95) 0%, rgba(10,14,26,0.98) 100%)',
    color: '#00e5ff',
  },
  day_discuss: {
    icon: '🌅',
    bg: 'radial-gradient(ellipse at center, rgba(60,40,0,0.95) 0%, rgba(10,14,26,0.98) 100%)',
    color: '#ffc107',
  },
  day_vote: {
    icon: '⚖️',
    bg: 'radial-gradient(ellipse at center, rgba(60,0,20,0.95) 0%, rgba(10,14,26,0.98) 100%)',
    color: '#ff2d6b',
  },
  game_over: {
    icon: '🏁',
    bg: 'radial-gradient(ellipse at center, rgba(20,20,20,0.98) 0%, rgba(10,14,26,0.99) 100%)',
    color: '#888',
  },
};

export function PhaseTransitionOverlay({ phase, round }: PhaseTransitionOverlayProps) {
  const { t } = useI18n();
  const [show, setShow] = useState(true);
  const cfg = PHASE_STYLES[phase] ?? PHASE_STYLES.night;
  const label = t(`phase.${phase}`) !== `phase.${phase}` ? t(`phase.${phase}`) : phase;

  useEffect(() => {
    setShow(true);
    const timer = setTimeout(() => setShow(false), 2000);
    return () => clearTimeout(timer);
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
        {label}
      </div>
      {round !== undefined && (
        <div className="text-text-muted font-mono text-sm mt-2">{t('phase.round', { n: String(round) })}</div>
      )}
    </div>
  );
}
