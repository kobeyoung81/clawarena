interface StatusPulseProps {
  status: 'live' | 'idle' | 'error' | 'waiting';
  label?: string;
  className?: string;
}

const STATUS_CONFIG = {
  live:    { dot: 'bg-accent-cyan', label: 'LIVE',        glow: 'rgba(0, 229, 255, 0.5)' },
  idle:    { dot: 'bg-text-muted',  label: 'IDLE',        glow: 'rgba(122, 139, 168, 0.3)' },
  error:   { dot: 'bg-accent-mag',  label: 'ERROR',       glow: 'rgba(255, 45, 107, 0.5)' },
  waiting: { dot: 'bg-accent-amber',label: 'WAITING',     glow: 'rgba(255, 193, 7, 0.5)' },
};

export function StatusPulse({ status, label, className = '' }: StatusPulseProps) {
  const cfg = STATUS_CONFIG[status];

  return (
    <div className={`inline-flex items-center gap-1.5 ${className}`}>
      <span className="relative flex h-2 w-2">
        <span
          className={`absolute inline-flex h-full w-full rounded-full ${cfg.dot} opacity-75`}
          style={{ animation: status !== 'idle' ? 'ping-slow 2s cubic-bezier(0,0,0.2,1) infinite' : 'none' }}
        />
        <span className={`relative inline-flex h-2 w-2 rounded-full ${cfg.dot}`} />
      </span>
      <span className="text-xs font-mono font-semibold tracking-widest" style={{ color: cfg.glow }}>
        {label ?? cfg.label}
      </span>
    </div>
  );
}
