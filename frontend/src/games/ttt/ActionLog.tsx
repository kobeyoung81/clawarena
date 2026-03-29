import type { GameEvent } from '../../types';

export interface ActionLogEntryProps {
  entry: GameEvent;
  players?: Array<{ agent_id: number; name: string }>;
}

const POSITION_NAMES = [
  'top-left', 'top-center', 'top-right',
  'mid-left', 'center', 'mid-right',
  'bottom-left', 'bottom-center', 'bottom-right',
] as const;

const MARKERS = ['X', 'O'] as const;

/** Tiny 3x3 grid showing the latest move highlighted. */
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
          {i === position ? marker : '.'}
        </span>
      ))}
    </span>
  );
}

export default function TicTacToeActionLog({ entry, players }: ActionLogEntryProps) {
  const { event_type, actor, details } = entry;

  // Determine marker from actor seat (seat 0 = X, seat 1 = O)
  const seatIdx = actor?.seat ?? 0;
  const marker = MARKERS[seatIdx] ?? 'X';
  const agentName = actor?.agent_id != null
    ? players?.find(p => p.agent_id === actor.agent_id)?.name
    : undefined;

  // Handle game over events
  if (event_type === 'game_over' || entry.game_over) {
    const winnerTeam = entry.result?.winner_team;
    if (winnerTeam) {
      return <span className="text-accent-cyan font-semibold">Game over - {winnerTeam} wins!</span>;
    }
    return <span className="text-yellow-400 font-semibold">Game over - Draw!</span>;
  }

  // Handle move events
  if (event_type === 'move' || event_type === 'action') {
    const pos = typeof details?.position === 'number'
      ? details.position
      : typeof details?.pos === 'number'
        ? details.pos
        : null;

    const label = agentName ? `${agentName} (${marker})` : marker;

    if (pos !== null && pos >= 0 && pos <= 8) {
      const name = POSITION_NAMES[pos];
      return (
        <div className="text-[10px] text-accent-cyan/60 mt-0.5 font-mono flex items-center gap-1">
          <strong className="text-text-primary/80">{label}</strong>
          {' plays '}
          <strong className="text-text-primary/70">{name}</strong>
          <span className="text-text-muted/40"> (pos {pos})</span>
          <MiniGrid position={pos} marker={marker} />
        </div>
      );
    }

    // Fallback for action with no recognized position
    return (
      <div className="text-[10px] text-accent-cyan/60 mt-0.5 font-mono">
        <strong className="text-text-primary/80">{label}</strong> {JSON.stringify(details)}
      </div>
    );
  }

  // Generic event fallback
  const message = typeof details?.message === 'string' ? details.message : event_type;
  return <span className="text-text-muted/70 text-xs">{message}</span>;
}
