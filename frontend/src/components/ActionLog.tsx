import { useEffect, useRef } from 'react';
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
    <div className="bg-gray-800 rounded-lg border border-gray-700 flex flex-col h-64">
      <h3 className="text-sm font-semibold text-gray-400 uppercase tracking-wide p-3 border-b border-gray-700">
        Action Log
      </h3>
      <div className="flex-1 overflow-y-auto p-3 flex flex-col gap-1">
        {timeline && timeline.length > 0 ? (
          timeline.map((entry, idx) => {
            const isCurrent = isReplay && idx === currentStep;
            return (
              <div
                key={idx}
                className={`text-xs rounded px-2 py-1 ${
                  isCurrent ? 'bg-yellow-900 border border-yellow-500 text-white' : 'text-gray-300'
                }`}
              >
                <span className="text-gray-500 mr-2">T{entry.turn}</span>
                {entry.events.map((ev, ei) => (
                  <span key={ei} className="mr-2">{ev.message}</span>
                ))}
                {entry.action && (
                  <span className="text-blue-400">
                    {JSON.stringify(entry.action)}
                  </span>
                )}
              </div>
            );
          })
        ) : liveEvents && liveEvents.length > 0 ? (
          liveEvents.map((ev, idx) => (
            <div key={idx} className="text-xs text-gray-300 px-2 py-0.5">
              {ev}
            </div>
          ))
        ) : (
          <span className="text-gray-500 text-xs italic">No events yet</span>
        )}
        <div ref={bottomRef} />
      </div>
    </div>
  );
}
