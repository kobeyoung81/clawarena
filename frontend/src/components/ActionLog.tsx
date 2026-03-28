import { useEffect, useRef } from 'react';
import { useI18n } from '../i18n';
import { ACTION_LOG_COMPONENTS, DefaultActionLog } from './actionlogs';
import type { HistoryTimeline } from '../types';

interface ActionLogProps {
  timeline?: HistoryTimeline[];
  currentStep?: number;
  isReplay?: boolean;
  gameType?: string;
  players?: Array<{ agent_id: number; name: string }>;
}

export function ActionLog({ timeline, currentStep, isReplay, gameType, players }: ActionLogProps) {
  const { t } = useI18n();
  const bottomRef = useRef<HTMLDivElement>(null);
  const EntryRenderer = gameType ? (ACTION_LOG_COMPONENTS[gameType] ?? DefaultActionLog) : DefaultActionLog;

  useEffect(() => {
    if (!isReplay) {
      bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
    }
  }, [timeline, isReplay]);

  return (
    <div className="glass rounded-xl border-white/8 flex flex-col h-64 overflow-hidden">
      <div className="px-3 py-2 border-b border-white/6 flex items-center gap-2">
        <span className="text-xs font-mono font-semibold text-text-muted uppercase tracking-widest">
          {t('action_log.title')}
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
                className="text-sm rounded px-2 py-1.5 animate-slide-in"
                style={{
                  background: isCurrent ? 'rgba(0,229,255,0.08)' : 'transparent',
                  borderLeft: isCurrent ? '2px solid rgba(0,229,255,0.6)' : '2px solid transparent',
                }}
              >
                <span className="text-text-muted/50 font-mono mr-2">T{entry.turn}</span>
                <EntryRenderer entry={entry} players={players} />
              </div>
            );
          })
        ) : (
          <div className="flex-1 flex items-center justify-center">
            <span className="text-text-muted/30 text-xs font-mono italic">{t('action_log.empty')}</span>
          </div>
        )}
        <div ref={bottomRef} />
      </div>
    </div>
  );
}
