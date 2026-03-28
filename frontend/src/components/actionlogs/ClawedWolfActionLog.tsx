import React from 'react';
import { formatEventMessage, isDeathEvent, isPhaseChange } from '../../utils/narrativeFormatter';
import type { ActionLogEntryProps } from './types';

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

export function ClawedWolfActionLog({ entry, players }: ActionLogEntryProps) {
  const { events, action } = entry;
  const statePlayers = Array.isArray((entry.state as { players?: unknown }).players)
    ? (entry.state as { players: Array<{ seat?: number; name?: string }> }).players
    : [];

  const renderPublicAction = () => {
    if (!action || typeof action !== 'object') return null;
    const type = typeof action.type === 'string' ? action.type : '';
    const targetSeat = typeof action.target_seat === 'number' ? action.target_seat : undefined;
    const targetName = targetSeat !== undefined
      ? statePlayers.find(p => p.seat === targetSeat)?.name ?? `seat ${targetSeat}`
      : undefined;

    switch (type) {
      case 'speak':
        return typeof action.message === 'string'
          ? <span className="text-[11px] text-text-muted/85">💬 "{action.message}"</span>
          : null;
      case 'vote':
        return <span className="text-[11px] text-text-muted/85">⚖️ voted {targetName ?? 'unknown target'}</span>;
      case 'protect':
        return <span className="text-[11px] text-text-muted/85">🛡 protected {targetName ?? 'unknown target'}</span>;
      default:
        return null;
    }
  };

  const actionLine = renderPublicAction();

  return (
    <>
      {events.map((ev, ei) => {
        const msg = formatEventMessage(ev.message);
        const isDeath = isDeathEvent(ev.message);
        const isPhase = isPhaseChange(ev.message);
        return (
          <span
            key={ei}
            className="mr-2 leading-relaxed"
            style={{
              color: isDeath ? '#ff2d6b' : isPhase ? '#ffc107' : '#7a8ba8',
              fontWeight: isDeath || isPhase ? 600 : 400,
            }}
          >
            {boldAgentNames(msg, players)}
          </span>
        );
      })}

      {actionLine && <div className="mt-0.5 font-mono">{actionLine}</div>}
    </>
  );
}
