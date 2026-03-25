import { useRef } from 'react';
import { PlayerSeat } from './clawedwolf/PlayerSeat';
import { PhaseDisplay } from './clawedwolf/PhaseDisplay';
import { VoteOverlay } from './clawedwolf/VoteOverlay';
import { NightOverlay } from './clawedwolf/NightOverlay';
import { PhaseTransitionOverlay } from '../effects/PhaseTransitionOverlay';
import type { ClawedWolfPlayer } from '../../types';

export interface BoardProps {
  state: Record<string, unknown>;
  players?: ClawedWolfPlayer[];
  isReplay?: boolean;
}

const SEAT_POSITIONS: React.CSSProperties[] = [
  { top: '5%',  left: '50%', transform: 'translateX(-50%)' },
  { top: '20%', left: '82%', transform: 'translate(-50%, -50%)' },
  { top: '65%', left: '90%', transform: 'translate(-50%, -50%)' },
  { top: '88%', left: '65%', transform: 'translate(-50%, -50%)' },
  { top: '88%', left: '35%', transform: 'translate(-50%, -50%)' },
  { top: '65%', left: '10%', transform: 'translate(-50%, -50%)' },
  { top: '20%', left: '18%', transform: 'translate(-50%, -50%)' },
];

interface ClawedWolfState {
  phase?: string;
  round?: number;
  players?: ClawedWolfPlayer[];
  current_speaker?: number;
  votes?: Record<string, number>;
}

function getPhaseBackground(phase: string): React.CSSProperties {
  switch (phase) {
    case 'night':
      return { background: 'radial-gradient(ellipse at 50% 20%, #0d1a3a 0%, #0a0e1a 100%)' };
    case 'day_discuss':
      return { background: 'radial-gradient(ellipse at 50% 80%, #1a0f00 0%, #0a0e1a 100%)' };
    case 'day_vote':
      return { background: 'radial-gradient(ellipse at 50% 50%, #1a0008 0%, #0a0e1a 100%)' };
    case 'game_over':
      return { background: '#0a0a0a' };
    default:
      return { background: '#0a0e1a' };
  }
}

export function ClawedWolfBoard({ state, players: propPlayers, isReplay = false }: BoardProps) {
  const s = state as ClawedWolfState;
  const phase = s?.phase ?? 'night';
  const round = s?.round ?? 1;
  const statePlayers = s?.players ?? propPlayers ?? [];
  const isNight = phase === 'night';
  const currentSpeaker = s?.current_speaker;
  const votes = s?.votes ?? {};

  const prevPhaseRef = useRef(phase);
  const phaseChanged = prevPhaseRef.current !== phase;
  if (phaseChanged) prevPhaseRef.current = phase;

  const borderColor = {
    night:      'rgba(0,229,255,0.25)',
    day_discuss:'rgba(255,193,7,0.25)',
    day_vote:   'rgba(255,45,107,0.25)',
    game_over:  'rgba(100,100,100,0.2)',
  }[phase] ?? 'rgba(0,229,255,0.2)';

  return (
    <div
      className="relative w-full h-96 rounded-xl overflow-hidden"
      style={{
        ...getPhaseBackground(phase),
        border: `1px solid ${borderColor}`,
        transition: 'background 0.8s ease',
      }}
    >
      {/* Night atmosphere */}
      <NightOverlay isActive={isNight} />

      {/* Phase transition overlay */}
      {phaseChanged && <PhaseTransitionOverlay phase={phase} round={round} />}

      {/* Center phase indicator */}
      <PhaseDisplay phase={phase} round={round} />

      {/* Player seats arranged in ellipse */}
      {statePlayers.slice(0, 7).map((player, idx) => {
        const pos = SEAT_POSITIONS[idx];
        const isSpeaker = currentSpeaker === player.seat;
        const voteCount = votes[String(player.seat)];

        return (
          <PlayerSeat
            key={player.seat ?? idx}
            player={player}
            isCurrentSpeaker={isSpeaker}
            voteCount={voteCount}
            isNight={isNight}
            isReplay={isReplay}
            phase={phase}
            style={pos}
          />
        );
      })}

      {/* Vote overlay (bottom bar) */}
      {phase === 'day_vote' && Object.keys(votes).length > 0 && (
        <VoteOverlay votes={votes} players={statePlayers} />
      )}

      {/* Phase badge (bottom-right corner) */}
      <div className="absolute bottom-2 right-3 text-[10px] font-mono text-text-muted/40 pointer-events-none">
        {phase.replace('_', ' ')} · Round {round}
      </div>
    </div>
  );
}
