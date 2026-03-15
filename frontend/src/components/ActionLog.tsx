import { useEffect, useRef } from 'react';
import { formatAction, formatEventMessage, isDeathEvent, isPhaseChange } from '../utils/narrativeFormatter';
import type { HistoryTimeline } from '../types';

interface ActionLogProps {
  timeline?: HistoryTimeline[];
  liveEvents?: string[];
  currentStep?: number;
  isReplay?: boolean;
}

export function ActionLog({ timeline, liveEvents, currentStep, isReplay }: ActionLogProps) {
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!isReplay) {
      bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
    }
  }, [timeline, liveEvents, isReplay]);

  return (
    <div className="glass rounded-xl border-white/8 flex flex-col h-64 overflow-hidden">
      <div className="px-3 py-2 border-b border-white/6 flex items-center gap-2">
        <span className="text-xs font-mono font-semibold text-text-muted uppercase tracking-widest">
          Action Log
        </span>
        <span className="flex h-1.5 w-1.5 rounded-full bg-accent-cyan/60" />
      </div>

      <div className="flex-1 overflow-y-auto p-2 flex flex-col gap-1">
        {timeline && timeline.length > 0 ? (
          timeline.map((entry, idx) => {
            const isCurrent = isReplay && idx === currentStep;
            return (
              <div
                key={idx}
                className="text-xs rounded px-2 py-1.5 animate-slide-in"
                style={{
                  background: isCurrent ? 'rgba(0,229,255,0.08)' : 'transparent',
                  borderLeft: isCurrent ? '2px solid rgba(0,229,255,0.6)' : '2px solid transparent',
                }}
              >
                <span className="text-text-muted/50 font-mono mr-2">T{entry.turn}</span>
                {entry.events.map((ev, ei) => {
                  const msg = formatEventMessage(ev.message);
                  const isDeath = isDeathEvent(ev.message);
                  const isPhase = isPhaseChange(ev.message);
                  return (
                    <span
                      key={ei}
                      className="mr-2 leading-relaxed"
                      style={{
                        color: isDeath ? '#ff2d6b' : isPhase ? '#ffc107' : '#7a8ba8',
                        fontWeight: isDeath || isPhase ? 600 : 400,
                      }}
                    >
                      {msg}
                    </span>
                  );
                })}
                {entry.action && (
                  <div className="text-[10px] text-accent-cyan/60 mt-0.5 font-mono">
                    {formatAction(entry.action)}
                  </div>
                )}
              </div>
            );
          })
        ) : liveEvents && liveEvents.length > 0 ? (
          liveEvents.map((ev, idx) => (
            <div
              key={idx}
              className="text-xs text-text-muted/70 px-2 py-0.5 animate-slide-in"
            >
              {formatEventMessage(ev)}
            </div>
          ))
        ) : (
          <div className="flex-1 flex items-center justify-center">
            <span className="text-text-muted/30 text-xs font-mono italic">No events yet...</span>
          </div>
        )}
        <div ref={bottomRef} />
      </div>
    </div>
  );
}
