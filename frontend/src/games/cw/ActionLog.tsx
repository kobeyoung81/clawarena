import React from 'react';
import type { GameEvent } from '../../types';

export interface ActionLogEntryProps {
  entry: GameEvent;
  players?: Array<{ agent_id: number; name: string }>;
}

function boldAgentNames(message: string, players?: Array<{ name: string }>): React.ReactNode {
  if (!players?.length) return message;
  const names = players.map(p => p.name).filter(Boolean);
  if (names.length === 0) return message;
  const regex = new RegExp(`\\b(${names.map(n => n.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')).join('|')})\\b`, 'g');
  const parts = message.split(regex);
  return parts.map((part, i) =>
    names.includes(part)
      ? <strong key={i} className="text-text-primary">{part}</strong>
      : part
  );
}

/** Map phase names to narrative descriptions */
function phaseNarrative(phase: string): string {
  switch (phase) {
    case 'night_clawedwolf': return '🐺 The wolves awaken and choose their prey...';
    case 'night_seer':       return '👁 The seer opens their eyes...';
    case 'night_guard':      return '🛡 The guard takes their post...';
    case 'day_discuss':      return '🌅 The village gathers to discuss.';
    case 'day_vote':         return '⚖️ The time for judgment is at hand.';
    default:                 return '';
  }
}

export default function ClawedWolfActionLog({ entry, players }: ActionLogEntryProps) {
  const { event_type, actor, target, details } = entry;
  const statePlayers = Array.isArray((entry.state as { players?: unknown }).players)
    ? (entry.state as { players: Array<{ id?: number; seat?: number; name?: string }> }).players
    : [];

  const content = typeof details?.content === 'string' ? details.content : '';
  const message = content || (typeof details?.message === 'string' ? details.message : '');

  /** Resolve a seat number to a player name using state (id↔seat) + players prop (agent_id↔name) */
  const nameForSeat = (seat: number | undefined): string => {
    if (seat === undefined) return 'unknown';
    // Try state player name first (available during live mode)
    const statePlayer = statePlayers.find(p => p.seat === seat);
    if (statePlayer?.name) return statePlayer.name;
    // Join via agent_id: state has id+seat, players prop has agent_id+name
    if (statePlayer?.id && players?.length) {
      const match = players.find(p => p.agent_id === statePlayer.id);
      if (match?.name) return match.name;
    }
    return `seat ${seat}`;
  };

  const targetSeat = target?.seat;
  const targetName = nameForSeat(targetSeat);

  const renderPublicAction = () => {
    switch (event_type) {
      case 'game_start':
        return <span className="text-accent-cyan font-semibold text-xs">🎮 Game Started</span>;

      case 'speak':
        return message
          ? <span className="text-[11px] text-text-muted/85">{boldAgentNames(message, players)}</span>
          : <span className="text-[11px] text-text-muted/40 italic">(no message)</span>;

      case 'vote':
        return <span className="text-[11px] text-text-muted/85">voted <strong className="text-white">{targetName}</strong></span>;

      case 'protect':
        return <span className="text-[11px] text-text-muted/85">protected <strong className="text-white">{targetName}</strong></span>;

      case 'kill':
        return <span className="text-[11px] text-text-muted/85">targeted <strong className="text-white">{targetName}</strong></span>;

      case 'investigate':
        return <span className="text-[11px] text-text-muted/85">investigated <strong className="text-white">{targetName}</strong></span>;

      case 'phase_change': {
        const phase = typeof details?.phase === 'string' ? details.phase : '';
        const narrative = phaseNarrative(phase);
        if (narrative) {
          return <span className="text-yellow-400 font-semibold text-xs">{narrative}</span>;
        }
        return <span className="text-yellow-400 font-semibold text-xs">Phase: {phase.replace(/_/g, ' ')}</span>;
      }

      case 'night_resolve': {
        const killedSeat = details?.killed_seat;
        const guarded = details?.guarded;
        if (killedSeat != null && killedSeat !== false) {
          const victimName = nameForSeat(killedSeat as number);
          return <span className="text-accent-mag font-semibold text-xs">🐺 <strong className="text-white">{victimName}</strong> was killed in the night</span>;
        }
        if (guarded) {
          return <span className="text-green-400 font-semibold text-xs">☮️ A peaceful night — the guard's protection held</span>;
        }
        return <span className="text-green-400 font-semibold text-xs">☮️ A peaceful night — no one was killed</span>;
      }

      case 'guard_save': {
        const savedSeat = details?.saved_seat;
        const savedName = savedSeat != null ? nameForSeat(savedSeat as number) : 'someone';
        return <span className="text-green-400 font-semibold text-xs">🛡 The guard saved <strong className="text-white">{savedName}</strong> from the wolves!</span>;
      }

      case 'vote_result': {
        const eliminated = details?.eliminated;
        const tally = details?.tally as Record<string, number> | undefined;
        if (!eliminated) {
          return <span className="text-yellow-400 font-semibold text-xs">⚖️ No consensus reached — no one is eliminated</span>;
        }
        const tallyStr = tally
          ? Object.entries(tally).map(([seat, count]) => `${nameForSeat(Number(seat))}: ${count}`).join(', ')
          : '';
        return (
          <span className="text-yellow-400 font-semibold text-xs">
            ⚖️ Vote result: <strong className="text-white">{targetName}</strong> is eliminated{tallyStr ? ` (${tallyStr})` : ''}
          </span>
        );
      }

      case 'elimination':
      case 'death': {
        const cause = typeof details?.cause === 'string' ? details.cause : '';
        const roleReveal = typeof details?.role_reveal === 'string' ? details.role_reveal : '';
        const causeLabel = cause === 'night_kill' ? 'killed by wolves' : cause === 'vote_elimination' ? 'voted out' : 'eliminated';
        return (
          <span className="text-accent-mag font-semibold text-xs">
            💀 <strong className="text-white">{targetName}</strong> was {causeLabel}{roleReveal ? ` — revealed as ${roleReveal}` : ''}
          </span>
        );
      }

      case 'game_over':
        return (
          <span className="text-accent-cyan font-semibold text-xs">
            {entry.result?.winner_team
              ? `🏆 ${entry.result.winner_team === 'evil' ? 'Evil' : 'Good'} team wins!`
              : '🏁 Game over'}
          </span>
        );

      default:
        return null;
    }
  };

  const actionLine = renderPublicAction();

  if (actionLine) {
    const actorName = actor?.agent_id != null
      ? players?.find(p => p.agent_id === actor.agent_id)?.name
      : undefined;
    const actorLabel = actorName ?? (actor?.role ?? '');

    return (
      <>
        {actorLabel && (
          <span className="font-mono mr-1 text-[10px]">[<strong className="text-white">{actorLabel}</strong>]</span>
        )}
        {actionLine}
      </>
    );
  }

  // Fallback: show event_type
  return (
    <span className="text-text-muted/60 text-xs">
      {event_type.replace(/_/g, ' ')}
      {message ? `: ${message}` : ''}
    </span>
  );
}
