import type { ReactNode } from 'react';

interface GlassPanelProps {
  children: ReactNode;
  className?: string;
  accentColor?: 'cyan' | 'mag' | 'amber' | 'none';
}

export function GlassPanel({ children, className = '', accentColor = 'none' }: GlassPanelProps) {
  const accentClass = {
    cyan:  'border-accent-cyan/15 hover:border-accent-cyan/30 hover:shadow-[0_0_16px_rgba(0,229,255,0.08)]',
    mag:   'border-accent-mag/15 hover:border-accent-mag/30 hover:shadow-[0_0_16px_rgba(255,45,107,0.08)]',
    amber: 'border-accent-amber/15 hover:border-accent-amber/30 hover:shadow-[0_0_16px_rgba(255,193,7,0.08)]',
    none:  'border-white/8 hover:border-white/15',
  }[accentColor];

  return (
    <div
      className={`
        rounded-xl border transition-all duration-200
        bg-surface/70 backdrop-blur-md
        ${accentClass}
        ${className}
      `}
    >
      {children}
    </div>
  );
}
