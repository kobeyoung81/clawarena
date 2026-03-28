import type { ActionLogEntryProps } from './types';

const POSITION_NAMES = [
  'top-left', 'top-center', 'top-right',
  'mid-left', 'center', 'mid-right',
  'bottom-left', 'bottom-center', 'bottom-right',
] as const;

const MARKERS = ['X', 'O'] as const;

/** Tiny 3×3 grid showing the latest move highlighted. */
function MiniGrid({ position, marker }: { position: number; marker: string }) {
  return (
    <span className="inline-grid grid-cols-3 gap-px ml-1.5 align-middle" style={{ width: 24, height: 24 }}>
      {Array.from({ length: 9 }, (_, i) => (
        <span
          key={i}
          className="flex items-center justify-center text-[6px] font-mono leading-none rounded-sm"
          style={{
            width: 8,
            height: 8,
            background: i === position ? 'rgba(0,229,255,0.25)' : 'rgba(255,255,255,0.04)',
            color: i === position ? '#00e5ff' : 'transparent',
            border: i === position ? '1px solid rgba(0,229,255,0.5)' : '1px solid rgba(255,255,255,0.06)',
          }}
        >
          {i === position ? marker : '·'}
        </span>
      ))}
    </span>
  );
}

export function TicTacToeActionLog({ entry }: ActionLogEntryProps) {
  const { turn, action, events } = entry;

  // Determine marker from turn (turn 0 = X, turn 1 = O, …)
  const marker = MARKERS[turn % 2] ?? 'X';

  return (
    <>
      {/* Events (win / draw / etc.) */}
      {events.map((ev, ei) => {
        const isWin = /won|wins|victory/i.test(ev.message);
        const isDraw = /draw|tie/i.test(ev.message);
        const isEnd = /game.*over|finished/i.test(ev.message);
        return (
          <span
            key={ei}
            className="mr-2 leading-relaxed"
            style={{
              color: isWin ? '#00e5ff' : isDraw ? '#ffc107' : isEnd ? '#ffc107' : '#7a8ba8',
              fontWeight: isWin || isDraw || isEnd ? 600 : 400,
            }}
          >
            {isWin && '🏆 '}{isDraw && '🤝 '}{ev.message}
          </span>
        );
      })}

      {/* Action */}
      {action && renderAction(action, marker)}
    </>
  );
}

function renderAction(action: Record<string, unknown>, marker: string) {
  const pos = typeof action.position === 'number'
    ? action.position
    : typeof action.pos === 'number'
      ? action.pos
      : null;

  if (pos !== null && pos >= 0 && pos <= 8) {
    const name = POSITION_NAMES[pos];
    return (
      <div className="text-[10px] text-accent-cyan/60 mt-0.5 font-mono flex items-center gap-1">
        <strong className="text-text-primary/80">{marker}</strong>
        {' plays '}
        <strong className="text-text-primary/70">{name}</strong>
        <span className="text-text-muted/40"> (pos {pos})</span>
        <MiniGrid position={pos} marker={marker} />
      </div>
    );
  }

  // Fallback
  return (
    <div className="text-[10px] text-accent-cyan/60 mt-0.5 font-mono">
      → {JSON.stringify(action)}
    </div>
  );
}
