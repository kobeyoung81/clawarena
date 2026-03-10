import type { RoomAgent, PendingAction, HistoryPlayer } from '../types';

const ROLE_EMOJI: Record<string, string> = {
  werewolf: '🐺',
  seer: '👁',
  guard: '🛡',
  villager: '👤',
  witch: '🧙',
};

interface AgentPanelProps {
  agents: RoomAgent[];
  pendingAction: PendingAction | null;
  replayPlayers?: HistoryPlayer[];
}

export function AgentPanel({ agents, pendingAction, replayPlayers }: AgentPanelProps) {
  const roleMap = replayPlayers
    ? Object.fromEntries(replayPlayers.map(p => [p.agent_id, p.role]))
    : {};

  return (
    <div className="bg-gray-800 rounded-lg p-4 border border-gray-700">
      <h3 className="text-sm font-semibold text-gray-400 uppercase tracking-wide mb-3">Players</h3>
      <div className="flex flex-col gap-2">
        {agents.map(ra => {
          const isCurrentTurn = pendingAction?.player_id === ra.agent?.id;
          const role = roleMap[ra.agent?.id];

          return (
            <div
              key={ra.id}
              className={`flex items-center justify-between p-2 rounded ${
                isCurrentTurn ? 'bg-yellow-900 border border-yellow-500' : 'bg-gray-700'
              }`}
            >
              <div className="flex items-center gap-2">
                <span className="text-xs text-gray-400">#{ra.slot}</span>
                <span className="text-white text-sm font-medium">
                  {ra.agent?.name ?? `Agent ${ra.agent?.id}`}
                </span>
                {role && (
                  <span className="text-sm" title={role}>
                    {ROLE_EMOJI[role] ?? '❓'} <span className="text-xs text-gray-400">{role}</span>
                  </span>
                )}
              </div>
              <div className="flex items-center gap-3 text-xs text-gray-400">
                <span>{ra.agent?.elo_rating ?? '—'} ELO</span>
                <span>Score: {ra.score}</span>
                {isCurrentTurn && (
                  <span className="text-yellow-400 font-semibold animate-pulse">⚡ Turn</span>
                )}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
