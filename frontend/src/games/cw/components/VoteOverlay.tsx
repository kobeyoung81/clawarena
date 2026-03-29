import type { ClawedWolfPlayer } from '../../../types';

interface VoteOverlayProps {
  votes: Record<string, number>;
  players: ClawedWolfPlayer[];
}

export function VoteOverlay({ votes, players }: VoteOverlayProps) {
  const totalVotes = Object.values(votes).reduce((sum, v) => sum + v, 0);
  if (totalVotes === 0) return null;

  const maxVotes = Math.max(...Object.values(votes));

  return (
    <div className="absolute bottom-2 left-2 right-2 flex flex-wrap gap-2 justify-center pointer-events-none">
      {Object.entries(votes)
        .filter(([, count]) => count > 0)
        .sort(([, a], [, b]) => b - a)
        .map(([seat, count]) => {
          const player = players.find(p => String(p.seat) === seat);
          const isLeader = count === maxVotes;
          return (
            <div
              key={seat}
              className="flex items-center gap-1.5 px-2 py-1 rounded-full text-xs font-mono"
              style={{
                background: isLeader ? 'rgba(255,45,107,0.25)' : 'rgba(255,45,107,0.1)',
                border: `1px solid ${isLeader ? 'rgba(255,45,107,0.6)' : 'rgba(255,45,107,0.25)'}`,
                animation: 'voteReveal 0.4s ease both',
                boxShadow: isLeader ? '0 0 8px rgba(255,45,107,0.3)' : 'none',
              }}
            >
              <span className="text-white/80">{player?.name ?? `P${seat}`}</span>
              <span className="text-accent-mag font-bold">{count}</span>
              <span className="text-red-400">▲</span>
            </div>
          );
        })}
    </div>
  );
}
