import { formatEventMessage, isDeathEvent, isPhaseChange } from '../../utils/narrativeFormatter';
import type { ActionLogEntryProps } from './types';

export function DefaultActionLog({ entry }: ActionLogEntryProps) {
  const { events, action } = entry;

  return (
    <>
      {events.map((ev, ei) => {
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

      {action && (
        <div className="text-[10px] text-accent-cyan/60 mt-0.5 font-mono whitespace-pre-wrap break-all">
          → {JSON.stringify(action, null, 2)}
        </div>
      )}
    </>
  );
}
