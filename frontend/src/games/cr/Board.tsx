export interface CrPlayer {
  seat: number;
  name: string;
  alive: boolean;
  id: number;
}

export interface BoardProps {
  state: Record<string, unknown>;
  players?: CrPlayer[];
  isReplay?: boolean;
}

interface CrState {
  players: Array<{
    id: number;
    seat: number;
    hits: number;
    alive: boolean;
    gadget_count?: number;
  }>;
  bullet_index: number;
  total_bullets: number;
  current_turn: number;
  phase: 'playing' | 'finished';
  winner?: number;
  is_draw: boolean;
}

export default function ClawedRouletteBoard({ state, players }: BoardProps) {
  const s = state as unknown as CrState;
  const bulletIndex = s?.bullet_index ?? 0;
  const totalBullets = s?.total_bullets ?? 0;
  const currentTurn = s?.current_turn ?? 0;
  const phase = s?.phase ?? 'playing';
  const winnerId = s?.winner;
  const isDraw = s?.is_draw ?? false;
  const statePlayers = s?.players ?? [];

  const getName = (seatOrId: number, bySeat = true) => {
    const p = bySeat
      ? players?.find(pl => pl.seat === seatOrId)
      : players?.find(pl => pl.id === seatOrId);
    return p?.name ?? `Player ${seatOrId}`;
  };

  // Status line
  let statusMsg: string;
  if (phase === 'finished') {
    if (isDraw) {
      statusMsg = 'Game Over — Draw!';
    } else if (winnerId != null) {
      statusMsg = `Game Over — ${getName(winnerId, false)} wins!`;
    } else {
      statusMsg = 'Game Over';
    }
  } else {
    statusMsg = `${getName(currentTurn)}'s turn`;
  }

  return (
    <div
      className="rounded-xl p-5 flex flex-col gap-5"
      style={{ background: 'rgba(255,255,255,0.03)', border: '1px solid rgba(255,255,255,0.06)' }}
    >
      {/* Bullet chamber progress */}
      <div className="flex flex-col gap-1.5">
        <span className="text-[10px] font-mono text-text-muted/50 uppercase tracking-widest">
          Chamber ({bulletIndex}/{totalBullets})
        </span>
        <div className="flex gap-1.5 flex-wrap">
          {Array.from({ length: totalBullets }, (_, i) => {
            const used = i < bulletIndex;
            return (
              <span
                key={i}
                className="inline-block w-3 h-3 rounded-full"
                style={{
                  background: used ? 'rgba(255,255,255,0.08)' : 'rgba(255,152,0,0.7)',
                  border: used ? '1px solid rgba(255,255,255,0.06)' : '1px solid rgba(255,152,0,0.9)',
                }}
              />
            );
          })}
        </div>
      </div>

      {/* Player cards */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        {statePlayers.map((sp) => {
          const isTurn = phase === 'playing' && sp.seat === currentTurn;
          const name = getName(sp.seat);
          const maxHits = 2;

          return (
            <div
              key={sp.seat}
              className="rounded-lg p-3 flex flex-col gap-2 transition-all"
              style={{
                background: sp.alive ? 'rgba(255,255,255,0.04)' : 'rgba(255,255,255,0.01)',
                border: isTurn
                  ? '2px solid rgba(255,152,0,0.8)'
                  : '1px solid rgba(255,255,255,0.06)',
                boxShadow: isTurn ? '0 0 12px rgba(255,152,0,0.25)' : 'none',
                opacity: sp.alive ? 1 : 0.45,
              }}
            >
              {/* Name */}
              <div className="flex items-center justify-between">
                <span className="text-sm font-semibold text-white truncate">{name}</span>
                {isTurn && (
                  <span
                    className="w-2 h-2 rounded-full animate-pulse"
                    style={{ background: '#ff9800' }}
                  />
                )}
              </div>

              {/* Hit indicators */}
              <div className="flex gap-1">
                {Array.from({ length: maxHits }, (_, i) => (
                  <span
                    key={i}
                    className="text-xs"
                    style={{ color: i < sp.hits ? '#ef4444' : 'rgba(255,255,255,0.15)' }}
                  >
                    ♥
                  </span>
                ))}
              </div>

              {/* Status + gadgets */}
              <div className="flex items-center justify-between">
                <span
                  className="text-[10px] font-mono uppercase tracking-wider"
                  style={{ color: sp.alive ? '#4ade80' : '#ef4444' }}
                >
                  {sp.alive ? 'Alive' : 'Eliminated'}
                </span>
                {sp.gadget_count != null && sp.gadget_count > 0 && (
                  <span className="text-[10px] font-mono text-text-muted/50">
                    🎒 {sp.gadget_count}
                  </span>
                )}
              </div>
            </div>
          );
        })}
      </div>

      {/* Status line */}
      <div className="text-center text-sm font-semibold text-white">{statusMsg}</div>
    </div>
  );
}
