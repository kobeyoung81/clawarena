import { useRef } from 'react';
import { PlayerSeat } from './components/PlayerSeat';
import { PhaseDisplay } from './components/PhaseDisplay';
import { VoteOverlay } from './components/VoteOverlay';
import { NightOverlay } from './components/NightOverlay';
import { PhaseTransitionOverlay } from '../../components/effects/PhaseTransitionOverlay';
import { useI18n } from '../../i18n';
import { normalizePhase } from '../../utils/normalizePhase';
import type { ClawedWolfPlayer } from '../../types';

export interface BoardProps {
  state: Record<string, unknown>;
  players?: ClawedWolfPlayer[];
  isReplay?: boolean;
}

interface ClawedWolfState {
  phase?: string;
  round?: number;
  players?: ClawedWolfPlayer[];
  current_speaker?: number;
  votes?: Record<string, number>;
  speeches?: Array<{ seat: number; name?: string; message: string }>;
  day_speeches?: Array<{ seat: number; name?: string; message: string }>;
}

function getPhaseBackground(phase: string): React.CSSProperties {
  const np = normalizePhase(phase);
  switch (np) {
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

/** Inline speech slot — fixed position within document flow */
function SpeechSlot({ speech, speakerName, isActive }: { speech?: string; speakerName?: string; isActive: boolean }) {
  if (!speech) return null;
  return (
    <div
      className={`
        rounded-lg px-4 py-2 text-xs leading-relaxed backdrop-blur-sm
        transition-all duration-300 border
        ${isActive
          ? 'bg-surface/95 border-accent-cyan/30 text-white/90'
          : 'bg-surface/70 border-white/5 text-white/50'}
      `}
      style={{
        boxShadow: isActive ? '0 0 12px rgba(0,229,255,0.12), 0 2px 8px rgba(0,0,0,0.3)' : '0 2px 8px rgba(0,0,0,0.3)',
      }}
    >
      {speakerName && (
        <span className={`font-semibold mr-1 ${isActive ? 'text-accent-cyan' : 'text-accent-cyan/50'}`}>
          {isActive && <span className="inline-block w-1.5 h-1.5 rounded-full bg-accent-cyan mr-1 align-middle" />}
          {speakerName}:
        </span>
      )}
      <span title={speech}>
        &ldquo;{speech.length > 200 ? speech.slice(0, 200) + '…' : speech}&rdquo;
      </span>
    </div>
  );
}

export default function ClawedWolfBoard({ state, players: propPlayers, isReplay = false }: BoardProps) {
  const { t } = useI18n();
  const s = state as ClawedWolfState;
  const phase = s?.phase ?? 'night';
  const round = s?.round ?? 1;
  const statePlayers = s?.players ?? propPlayers ?? [];
  const np = normalizePhase(phase);
  const isNight = np === 'night';
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
  }[np] ?? 'rgba(0,229,255,0.2)';

  // Two-row layout: upper = seats 0,2,4 ; lower = seats 1,3,5
  const upperSeats = [0, 2, 4];
  const lowerSeats = [1, 3, 5];

  const speeches = s?.day_speeches ?? s?.speeches ?? [];

  // Find latest speech per row
  const latestUpperSpeech = speeches.filter(sp => upperSeats.includes(sp.seat)).pop();
  const latestLowerSpeech = speeches.filter(sp => lowerSeats.includes(sp.seat)).pop();

  const isUpperActive = currentSpeaker != null && upperSeats.includes(currentSpeaker);
  const isLowerActive = currentSpeaker != null && lowerSeats.includes(currentSpeaker);

  // Resolve speaker name
  const speakerNameFor = (sp?: { seat: number; name?: string }) => {
    if (!sp) return undefined;
    if (sp.name) return sp.name;
    const p = statePlayers.find(pl => pl.seat === sp.seat);
    return p?.name ?? `P${sp.seat}`;
  };

  const badgeText = t('board_badge.' + np, { n: String(round) }) ?? `${np} · Round ${round}`;

  return (
    <div
      className="relative w-full rounded-xl overflow-hidden flex flex-col"
      style={{
        ...getPhaseBackground(phase),
        border: `1px solid ${borderColor}`,
        transition: 'background 0.8s ease',
        minHeight: 420,
      }}
    >
      {/* Night atmosphere */}
      <NightOverlay isActive={isNight} />

      {/* Phase transition overlay */}
      {phaseChanged && <PhaseTransitionOverlay phase={phase} round={round} />}

      {/* Center phase indicator */}
      <PhaseDisplay phase={phase} round={round} />

      {/* Content area — grows vertically with content */}
      <div className="relative z-10 flex flex-col flex-1 px-6 py-6" style={{ gap: '1.5rem' }}>
        {/* Upper row: seats 0, 2, 4 */}
        <div className="flex justify-center items-end gap-10">
          {upperSeats.map(seat => {
            const player = statePlayers.find(p => p.seat === seat);
            if (!player) return <div key={seat} className="w-24" />;
            return (
              <PlayerSeat
                key={player.seat}
                player={player}
                isCurrentSpeaker={currentSpeaker === player.seat}
                voteCount={votes[String(player.seat)]}
                isNight={isNight}
                isReplay={isReplay}
                phase={phase}
                className="relative"
              />
            );
          })}
        </div>

        {/* Speech slots — between the two rows, grow with text */}
        {np === 'day_discuss' && (latestUpperSpeech || latestLowerSpeech) && (
          <div className="flex flex-col gap-2 flex-shrink-0">
            {latestUpperSpeech && (
              <SpeechSlot
                speech={latestUpperSpeech.message}
                speakerName={speakerNameFor(latestUpperSpeech)}
                isActive={isUpperActive}
              />
            )}
            {latestLowerSpeech && (
              <SpeechSlot
                speech={latestLowerSpeech.message}
                speakerName={speakerNameFor(latestLowerSpeech)}
                isActive={isLowerActive}
              />
            )}
          </div>
        )}

        {/* Spacer to push lower row down when no bubbles */}
        {!(np === 'day_discuss' && (latestUpperSpeech || latestLowerSpeech)) && (
          <div className="flex-1 min-h-[60px]" />
        )}

        {/* Lower row: seats 1, 3, 5 */}
        <div className="flex justify-center items-end gap-10">
          {lowerSeats.map(seat => {
            const player = statePlayers.find(p => p.seat === seat);
            if (!player) return <div key={seat} className="w-24" />;
            return (
              <PlayerSeat
                key={player.seat}
                player={player}
                isCurrentSpeaker={currentSpeaker === player.seat}
                voteCount={votes[String(player.seat)]}
                isNight={isNight}
                isReplay={isReplay}
                phase={phase}
                className="relative"
              />
            );
          })}
        </div>
      </div>

      {/* Vote overlay (bottom bar) */}
      {np === 'day_vote' && Object.keys(votes).length > 0 && (
        <VoteOverlay votes={votes} players={statePlayers} />
      )}

      {/* Phase badge (bottom-right corner) */}
      <div className="absolute bottom-2 right-3 text-[10px] font-mono text-text-muted/40 pointer-events-none">
        {badgeText}
      </div>
    </div>
  );
}
