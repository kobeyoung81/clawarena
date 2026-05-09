import React from 'react';
import { useI18n } from '../i18n';
import type { GameEvent } from '../types';

interface EventActionLogPanelProps {
  events: GameEvent[];
  currentStep?: number;
  isReplay: boolean;
  renderEntry: (entry: GameEvent) => React.ReactNode;
  className?: string;
  listClassName?: string;
  highlightLatestLive?: boolean;
}

const DEFAULT_PANEL_CLASS_NAME = 'glass rounded-xl border-white/8 flex flex-col h-64 overflow-hidden';
const DEFAULT_LIST_CLASS_NAME = 'flex-1 overflow-y-auto p-2 flex flex-col gap-1';
const STICKY_BOTTOM_THRESHOLD_PX = 48;

export function EventActionLogPanel({
  events,
  currentStep,
  isReplay,
  renderEntry,
  className = DEFAULT_PANEL_CLASS_NAME,
  listClassName = DEFAULT_LIST_CLASS_NAME,
  highlightLatestLive = false,
}: EventActionLogPanelProps) {
  const { t } = useI18n();
  const logContainerRef = React.useRef<HTMLDivElement>(null);
  const shouldAutoFollowRef = React.useRef(true);
  const latestEventSeq = events[events.length - 1]?.seq;

  const updateAutoFollowState = React.useCallback(() => {
    const container = logContainerRef.current;
    if (!container) return;
    const distanceFromBottom = container.scrollHeight - container.clientHeight - container.scrollTop;
    shouldAutoFollowRef.current = distanceFromBottom <= STICKY_BOTTOM_THRESHOLD_PX;
  }, []);

  React.useEffect(() => {
    shouldAutoFollowRef.current = !isReplay;
  }, [isReplay]);

  React.useEffect(() => {
    if (isReplay || !shouldAutoFollowRef.current) return;

    const container = logContainerRef.current;
    if (!container) return;

    const frame = requestAnimationFrame(() => {
      container.scrollTo({
        top: container.scrollHeight,
        behavior: events.length > 1 ? 'smooth' : 'auto',
      });
      shouldAutoFollowRef.current = true;
    });

    return () => cancelAnimationFrame(frame);
  }, [events.length, isReplay, latestEventSeq]);

  return (
    <div className={className}>
      <div className="px-3 py-2 border-b border-white/6 flex items-center gap-2">
        <span className="text-xs font-mono font-semibold text-text-muted uppercase tracking-widest">
          {t('action_log.title')}
        </span>
        <span className="flex h-1.5 w-1.5 rounded-full bg-accent-cyan/60" />
      </div>

      <div
        ref={logContainerRef}
        className={listClassName}
        onScroll={isReplay ? undefined : updateAutoFollowState}
      >
        {events.length > 0 ? (
          events.map((entry, idx) => {
            const isCurrent = isReplay && idx === currentStep;
            const isLatestLive = !isReplay && highlightLatestLive && idx === events.length - 1;

            return (
              <div
                key={entry.seq}
                className="text-sm rounded px-2 py-1.5 animate-slide-in"
                style={{
                  background: isCurrent || isLatestLive ? 'rgba(0,229,255,0.08)' : 'transparent',
                  borderLeft: isCurrent || isLatestLive ? '2px solid rgba(0,229,255,0.6)' : '2px solid transparent',
                  boxShadow: isLatestLive ? 'inset 0 0 0 1px rgba(0,229,255,0.12)' : undefined,
                }}
              >
                <span className="text-text-muted/50 font-mono mr-2">#{entry.seq}</span>
                {renderEntry(entry)}
              </div>
            );
          })
        ) : (
          <div className="flex-1 flex items-center justify-center">
            <span className="text-text-muted/30 text-xs font-mono italic">{t('action_log.empty')}</span>
          </div>
        )}
      </div>
    </div>
  );
}
