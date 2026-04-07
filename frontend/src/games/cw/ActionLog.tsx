import React from 'react';
import { useI18n } from '../../i18n';
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
  const { t } = useI18n();
  const { event_type, actor, target, details } = entry;
  const statePlayers = Array.isArray((entry.state as { players?: unknown }).players)
    ? (entry.state as { players: Array<{ id?: number; seat?: number; name?: string }> }).players
    : [];

  const content = typeof details?.content === 'string' ? details.content : '';
  const message = content || (typeof details?.message === 'string' ? details.message : '');

  /** Resolve a seat number to a player name using state (id↔seat) + players prop (agent_id↔name) */
  const nameForSeat = (seat: number | undefined): string => {
    if (seat === undefined) return 'unknown';
    const statePlayer = statePlayers.find(p => p.seat === seat);
    if (statePlayer?.name) return statePlayer.name;
    if (statePlayer?.id && players?.length) {
      const match = players.find(p => p.agent_id === statePlayer.id);
      if (match?.name) return match.name;
    }
    return `seat ${seat}`;
  };

  const targetSeat = target?.seat;
  const targetName = nameForSeat(targetSeat);

  /** Map phase names to narrative descriptions */
  const phaseNarrative = (phase: string): string => {
    switch (phase) {
      case 'night_clawedwolf': return t('cw_events.phase_wolves');
      case 'night_seer':       return t('cw_events.phase_seer');
      case 'night_guard':      return t('cw_events.phase_guard');
      case 'day_discuss':      return t('cw_events.phase_discuss');
      case 'day_vote':         return t('cw_events.phase_vote');
      default:                 return '';
    }
  };

  const renderPublicAction = () => {
    switch (event_type) {
      case 'game_start':
        return <span className="text-accent-cyan font-semibold text-xs">{t('cw_events.game_started')}</span>;

      case 'speak':
        return message
          ? <span className="text-[11px] text-text-muted/85">{boldAgentNames(message, players)}</span>
          : <span className="text-[11px] text-text-muted/40 italic">{t('cw_events.no_message')}</span>;

      case 'vote':
        return <span className="text-[11px] text-text-muted/85">{t('cw_events.voted')} <strong className="text-white">{targetName}</strong></span>;

      case 'protect':
        return <span className="text-[11px] text-text-muted/85">{t('cw_events.protected')} <strong className="text-white">{targetName}</strong></span>;

      case 'kill':
        return <span className="text-[11px] text-text-muted/85">{t('cw_events.targeted')} <strong className="text-white">{targetName}</strong></span>;

      case 'investigate':
        return <span className="text-[11px] text-text-muted/85">{t('cw_events.investigated')} <strong className="text-white">{targetName}</strong></span>;

      case 'phase_change': {
        const phase = typeof details?.phase === 'string' ? details.phase : '';
        const narrative = phaseNarrative(phase);
        if (narrative) {
          return <span className="text-yellow-400 font-semibold text-xs">{narrative}</span>;
        }
        return <span className="text-yellow-400 font-semibold text-xs">{t('cw_events.phase_label', { phase: phase.replace(/_/g, ' ') })}</span>;
      }

      case 'night_resolve': {
        const killedSeat = details?.killed_seat;
        const guarded = details?.guarded;
        if (killedSeat != null && killedSeat !== false) {
          const victimName = nameForSeat(killedSeat as number);
          return <span className="text-accent-mag font-semibold text-xs">{t('cw_events.killed_in_night', { name: victimName })}</span>;
        }
        if (guarded) {
          return <span className="text-green-400 font-semibold text-xs">{t('cw_events.peaceful_night_guarded')}</span>;
        }
        return <span className="text-green-400 font-semibold text-xs">{t('cw_events.peaceful_night')}</span>;
      }

      case 'guard_save': {
        return <span className="text-green-400 font-semibold text-xs">{t('cw_events.guard_saved_someone')}</span>;
      }

      case 'vote_result': {
        const eliminated = details?.eliminated;
        const tally = details?.tally as Record<string, number> | undefined;
        if (!eliminated) {
          return <span className="text-yellow-400 font-semibold text-xs">{t('cw_events.no_consensus')}</span>;
        }
        const tallyStr = tally
          ? Object.entries(tally).map(([seat, count]) => `${nameForSeat(Number(seat))}: ${count}`).join(', ')
          : '';
        return (
          <span className="text-yellow-400 font-semibold text-xs">
            {tallyStr
              ? t('cw_events.vote_result_tally', { name: targetName, tally: tallyStr })
              : t('cw_events.vote_result', { name: targetName })
            }
          </span>
        );
      }

      case 'elimination':
      case 'death': {
        const cause = typeof details?.cause === 'string' ? details.cause : '';
        const roleReveal = typeof details?.role_reveal === 'string' ? details.role_reveal : '';
        const causeLabel = cause === 'night_kill' ? t('cw_events.killed_by_wolves') : cause === 'vote_elimination' ? t('cw_events.voted_out') : t('cw_events.eliminated');
        return (
          <span className="text-accent-mag font-semibold text-xs">
            {roleReveal
              ? t('cw_events.death_message_role', { name: targetName, cause: causeLabel, role: t(`role_names.${roleReveal}`) })
              : t('cw_events.death_message', { name: targetName, cause: causeLabel })
            }
          </span>
        );
      }

      case 'game_over':
        return (
          <span className="text-accent-cyan font-semibold text-xs">
            {entry.result?.winner_team
              ? (entry.result.winner_team === 'evil' ? t('cw_events.evil_wins') : t('cw_events.good_wins'))
              : t('cw_events.game_over_generic')}
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
