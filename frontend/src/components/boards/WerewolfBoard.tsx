import type { WerewolfPlayer } from '../../types';

export interface BoardProps {
  state: Record<string, unknown>;
  players?: WerewolfPlayer[];
  isReplay?: boolean;
}

const ROLE_EMOJI: Record<string, string> = {
  werewolf: '🐺',
  seer: '👁',
  guard: '🛡',
  villager: '👤',
  witch: '🧙',
};

const PHASE_LABEL: Record<string, string> = {
  night: '🌙 Night',
  day_discuss: '☀️ Discussion',
  day_vote: '🗳️ Vote',
  game_over: '🏁 Game Over',
};

const SEAT_POSITIONS = [
  { top: '5%', left: '50%', transform: 'translateX(-50%)' },
  { top: '20%', left: '82%', transform: 'translate(-50%, -50%)' },
  { top: '65%', left: '90%', transform: 'translate(-50%, -50%)' },
  { top: '88%', left: '65%', transform: 'translate(-50%, -50%)' },
  { top: '88%', left: '35%', transform: 'translate(-50%, -50%)' },
  { top: '65%', left: '10%', transform: 'translate(-50%, -50%)' },
  { top: '20%', left: '18%', transform: 'translate(-50%, -50%)' },
];

interface WerewolfState {
  phase?: string;
  round?: number;
  players?: WerewolfPlayer[];
  current_speaker?: number;
  votes?: Record<string, number>;
}

export function WerewolfBoard({ state, players: propPlayers, isReplay }: BoardProps) {
  const s = state as WerewolfState;
  const phase = s?.phase ?? 'night';
  const round = s?.round ?? 1;
  const statePlayers = s?.players ?? propPlayers ?? [];
  const isNight = phase === 'night';
  const currentSpeaker = s?.current_speaker;
  const votes = s?.votes;

  const bgClass = isNight
    ? 'bg-slate-900 border-blue-900'
    : 'bg-yellow-950 border-yellow-800';

  return (
    <div className={`relative w-full h-80 rounded-xl border-2 ${bgClass} overflow-hidden`}>
      {/* Center phase indicator */}
      <div className="absolute inset-0 flex flex-col items-center justify-center pointer-events-none">
        <div className="text-2xl font-bold text-white opacity-30">
          {PHASE_LABEL[phase] ?? phase}
        </div>
        <div className="text-sm text-gray-400 opacity-30">Round {round}</div>
      </div>

      {/* Player seats */}
      {statePlayers.slice(0, 7).map((player, idx) => {
        const pos = SEAT_POSITIONS[idx];
        const isCurrentSpeaker = currentSpeaker === player.seat;

        return (
          <div
            key={player.seat ?? idx}
            className={`absolute flex flex-col items-center gap-0.5 cursor-default`}
            style={pos as React.CSSProperties}
          >
            <div
              className={`w-12 h-12 rounded-full flex items-center justify-center text-lg border-2 transition-all ${
                !player.alive
                  ? 'bg-gray-800 border-gray-600 opacity-50'
                  : isCurrentSpeaker
                  ? 'bg-yellow-600 border-yellow-400 scale-110'
                  : isNight
                  ? 'bg-blue-900 border-blue-600'
                  : 'bg-amber-700 border-amber-500'
              }`}
              title={isReplay && player.role ? player.role : undefined}
            >
              {!player.alive ? (
                <span>☠</span>
              ) : isReplay && player.role ? (
                <span>{ROLE_EMOJI[player.role] ?? '👤'}</span>
              ) : (
                <span className="text-white text-xs font-bold">{player.seat ?? idx + 1}</span>
              )}
            </div>
            <div className="text-center">
              <div className={`text-xs font-medium ${player.alive ? 'text-white' : 'text-gray-500'}`}>
                {player.name ?? `P${player.seat}`}
              </div>
              {isReplay && player.role && (
                <div className="text-xs text-gray-400">{player.role}</div>
              )}
              {votes && votes[String(player.seat)] !== undefined && (
                <div className="text-xs text-red-400 font-bold">
                  {votes[String(player.seat)]} vote{votes[String(player.seat)] !== 1 ? 's' : ''}
                </div>
              )}
            </div>
          </div>
        );
      })}

      {/* Phase label overlay */}
      <div className="absolute bottom-2 right-3 text-xs text-gray-400">
        {PHASE_LABEL[phase] ?? phase} · Round {round}
      </div>
    </div>
  );
}
