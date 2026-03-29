import React from 'react';
import { formatEventMessage, isDeathEvent, isPhaseChange } from '../../utils/narrativeFormatter';
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

export default function ClawedWolfActionLog({ entry, players }: ActionLogEntryProps) {
  const { event_type, actor, target, details } = entry;
  const statePlayers = Array.isArray((entry.state as { players?: unknown }).players)
    ? (entry.state as { players: Array<{ seat?: number; name?: string }> }).players
    : [];

  // Extract message from details if present
  const message = typeof details?.message === 'string' ? details.message : '';

  const renderPublicAction = () => {
    const targetSeat = target?.seat;
    const targetName = targetSeat !== undefined
      ? statePlayers.find(p => p.seat === targetSeat)?.name ?? `seat ${targetSeat}`
      : undefined;

    switch (event_type) {
      case 'speak':
        return message
          ? <span className="text-[11px] text-text-muted/85">{boldAgentNames(message, players)}</span>
          : null;
      case 'vote':
        return <span className="text-[11px] text-text-muted/85">voted {targetName ?? 'unknown target'}</span>;
      case 'protect':
        return <span className="text-[11px] text-text-muted/85">protected {targetName ?? 'unknown target'}</span>;
      case 'kill':
        return <span className="text-[11px] text-text-muted/85">targeted {targetName ?? 'unknown target'}</span>;
      case 'investigate':
        return <span className="text-[11px] text-text-muted/85">investigated {targetName ?? 'unknown target'}</span>;
      case 'phase_change': {
        const phase = typeof details?.phase === 'string' ? details.phase : event_type;
        const formattedMsg = formatEventMessage(`phase ${phase}`);
        return <span className="text-yellow-400 font-semibold text-xs">{formattedMsg}</span>;
      }
      case 'elimination':
      case 'death': {
        const victimName = targetSeat !== undefined
          ? statePlayers.find(p => p.seat === targetSeat)?.name ?? `seat ${targetSeat}`
          : 'unknown';
        return <span className="text-accent-mag font-semibold text-xs">{victimName} was eliminated</span>;
      }
      case 'game_over':
        return (
          <span className="text-accent-cyan font-semibold text-xs">
            {entry.result?.winner_team ? `${entry.result.winner_team} wins!` : 'Game over'}
          </span>
        );
      default:
        return null;
    }
  };

  const actionLine = renderPublicAction();

  // If we have a specific action rendering, show it
  if (actionLine) {
    // Build actor label
    const actorName = actor?.agent_id != null
      ? players?.find(p => p.agent_id === actor.agent_id)?.name
      : undefined;
    const actorLabel = actorName ?? (actor?.role ?? '');

    return (
      <>
        {actorLabel && (
          <span className="text-text-muted/50 font-mono mr-1 text-[10px]">[{actorLabel}]</span>
        )}
        {actionLine}
      </>
    );
  }

  // Fallback: show event_type + message
  const fallbackMsg = message || event_type;
  const formattedMsg = formatEventMessage(fallbackMsg);
  const isDeath = isDeathEvent(fallbackMsg);
  const isPhase = isPhaseChange(fallbackMsg);

  return (
    <span
      className="leading-relaxed"
      style={{
        color: isDeath ? '#ff2d6b' : isPhase ? '#ffc107' : '#7a8ba8',
        fontWeight: isDeath || isPhase ? 600 : 400,
      }}
    >
      {boldAgentNames(formattedMsg, players)}
    </span>
  );
}
