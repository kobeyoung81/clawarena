import React from 'react';
import { formatAction, formatEventMessage, isDeathEvent, isPhaseChange } from '../../utils/narrativeFormatter';
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

      {action && (
        <div className="text-[10px] text-accent-cyan/60 mt-0.5 font-mono">
          {formatAction(action)}
        </div>
      )}
    </>
  );
}
