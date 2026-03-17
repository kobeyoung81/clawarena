import { useI18n } from '../i18n';

interface ReplayControlsProps {
  step: number;
  total: number;
  isPlaying: boolean;
  speed: number;
  onPrev: () => void;
  onNext: () => void;
  onPlay: () => void;
  onJump: (step: number) => void;
  onSpeedChange: (speed: number) => void;
}

const SPEEDS = [0.5, 1, 2, 4];

export function ReplayControls({ step, total, isPlaying, speed, onPrev, onNext, onPlay, onJump, onSpeedChange }: ReplayControlsProps) {
  const { t } = useI18n();
  const progress = total > 1 ? (step / (total - 1)) * 100 : 0;
  const speedIdx = SPEEDS.indexOf(speed) === -1 ? 1 : SPEEDS.indexOf(speed);

  function cycleSpeed() {
    const next = SPEEDS[(speedIdx + 1) % SPEEDS.length];
    onSpeedChange(next);
  }

  return (
    <div
      className="glass rounded-xl border-white/8 p-4 flex flex-col gap-3"
      style={{ borderColor: 'rgba(0,229,255,0.12)' }}
    >
      {/* Header row */}
      <div className="flex items-center justify-between">
        <span className="text-xs font-mono font-semibold text-text-muted uppercase tracking-widest">
          {t('replay.title')}
        </span>
        <span className="text-xs font-mono text-text-muted/60">
          <span className="text-accent-cyan">{step + 1}</span>
          <span className="text-white/20"> / </span>
          <span className="text-white/60">{total}</span>
        </span>
      </div>

      {/* Timeline scrubber */}
      {total > 1 && (
        <div className="relative h-6 flex items-center group">
          {/* Track */}
          <div
            className="absolute inset-x-0 h-1 rounded-full"
            style={{ background: 'rgba(255,255,255,0.08)' }}
          />
          {/* Progress fill */}
          <div
            className="absolute left-0 h-1 rounded-full transition-all duration-150"
            style={{
              width: `${progress}%`,
              background: 'linear-gradient(90deg, rgba(0,229,255,0.6) 0%, rgba(0,229,255,0.9) 100%)',
              boxShadow: '0 0 6px rgba(0,229,255,0.5)',
            }}
          />
          {/* Invisible range input on top */}
          <input
            type="range"
            min={0}
            max={total - 1}
            value={step}
            onChange={e => onJump(Number(e.target.value))}
            className="absolute inset-0 w-full opacity-0 cursor-pointer h-full"
          />
          {/* Thumb indicator */}
          <div
            className="absolute w-3 h-3 rounded-full border border-accent-cyan/80 bg-surface pointer-events-none transition-all duration-150"
            style={{
              left: `calc(${progress}% - 6px)`,
              boxShadow: '0 0 8px rgba(0,229,255,0.6)',
            }}
          />
        </div>
      )}

      {/* Controls row */}
      <div className="flex items-center gap-2">
        {/* Prev */}
        <button
          onClick={onPrev}
          disabled={step <= 0}
          className="flex items-center justify-center w-8 h-8 rounded font-mono text-sm transition-all duration-150 disabled:opacity-25 disabled:cursor-not-allowed"
          style={{
            background: 'rgba(255,255,255,0.05)',
            border: '1px solid rgba(255,255,255,0.08)',
            color: 'rgba(255,255,255,0.7)',
          }}
          title={t('replay.prev')}
        >
          ◀
        </button>

        {/* Play / Pause — primary action */}
        <button
          onClick={onPlay}
          className="flex-1 flex items-center justify-center gap-2 h-8 rounded font-mono text-xs font-semibold transition-all duration-150"
          style={{
            background: isPlaying
              ? 'rgba(255,193,7,0.12)'
              : 'rgba(0,229,255,0.12)',
            border: isPlaying
              ? '1px solid rgba(255,193,7,0.4)'
              : '1px solid rgba(0,229,255,0.4)',
            color: isPlaying ? '#ffc107' : '#00e5ff',
            boxShadow: isPlaying
              ? '0 0 10px rgba(255,193,7,0.15)'
              : '0 0 10px rgba(0,229,255,0.15)',
          }}
          title={isPlaying ? t('replay.pause') : t('replay.play')}
        >
          {isPlaying ? `⏸ ${t('replay.pause')}` : `⏯ ${t('replay.play')}`}
        </button>

        {/* Next */}
        <button
          onClick={onNext}
          disabled={step >= total - 1}
          className="flex items-center justify-center w-8 h-8 rounded font-mono text-sm transition-all duration-150 disabled:opacity-25 disabled:cursor-not-allowed"
          style={{
            background: 'rgba(255,255,255,0.05)',
            border: '1px solid rgba(255,255,255,0.08)',
            color: 'rgba(255,255,255,0.7)',
          }}
          title={t('replay.next')}
        >
          ▶
        </button>

        {/* Speed toggle */}
        <button
          onClick={cycleSpeed}
          className="flex items-center justify-center w-12 h-8 rounded font-mono text-xs transition-all duration-150"
          style={{
            background: 'rgba(255,255,255,0.04)',
            border: '1px solid rgba(255,255,255,0.08)',
            color: speedIdx !== 1 ? '#ffc107' : 'rgba(255,255,255,0.4)',
          }}
          title={t('replay.speed')}
        >
          {SPEEDS[speedIdx]}×
        </button>
      </div>
    </div>
  );
}
