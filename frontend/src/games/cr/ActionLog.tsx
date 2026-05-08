import type { GameEvent } from '../../types';
import { useI18n } from '../../i18n';

export interface ActionLogEntryProps {
  entry: GameEvent;
  players?: Array<{ agent_id: number; name: string }>;
}

export default function ClawedRouletteActionLog({ entry, players }: ActionLogEntryProps) {
  const { t } = useI18n();
  const { event_type, actor, target, details } = entry;

  const resolveName = (agentId?: number): string => {
    if (agentId == null) return t('cr.unknown_player');
    return players?.find((player) => player.agent_id === agentId)?.name
      ?? t('cr.player_fallback', { id: agentId });
  };

  const actorName = resolveName(actor?.agent_id);
  const targetName = resolveName(target?.agent_id);

  if (event_type === 'game_start') {
    const liveCount = (details?.live_count as number) ?? '?';
    const blankCount = (details?.blank_count as number) ?? '?';
    return (
      <span className="text-accent-cyan font-semibold">
        {t('cr_events.game_started')} — {t('cr_events.game_started_detail', { live: liveCount, blank: blankCount })}
      </span>
    );
  }

  if (event_type === 'fire') {
    const bullet = details?.bullet as string | undefined;
    const selfShot = details?.self_shot as boolean | undefined;
    const targetHits = details?.target_hits as number | undefined;

    if (bullet === 'live') {
      return (
        <span style={{ color: '#ef4444' }}>
          {t('cr_events.fire_live', {
            name: actorName,
            target: targetName,
            hits: targetHits ?? '?',
          })}
        </span>
      );
    }

    if (selfShot) {
      return <span className="text-text-muted/70">{t('cr_events.fire_blank_self', { name: actorName })}</span>;
    }

    return (
      <span className="text-text-muted/70">
        {t('cr_events.fire_blank', { name: actorName, target: targetName })}
      </span>
    );
  }

  if (event_type === 'gadget_use') {
    const gadget = details?.gadget as string | undefined;
    if (gadget === 'fish_chips') {
      const hitsAfter = details?.hits_after as number | undefined;
      return (
        <span className="text-green-400">
          {t('cr_events.gadget_fish_chips', { name: actorName, hits: hitsAfter ?? 0 })}
        </span>
      );
    }
    if (gadget === 'goggles') {
      return (
        <span className="text-yellow-400">
          {t('cr_events.gadget_goggles', { name: actorName })}
        </span>
      );
    }
    return (
      <span className="text-text-muted/70">
        {t('cr_events.gadget_generic', { name: actorName, gadget: gadget ?? '?' })}
      </span>
    );
  }

  if (event_type === 'elimination') {
    return (
      <span style={{ color: '#ef4444' }} className="font-semibold">
        {t('cr_events.elimination', { name: targetName })}
      </span>
    );
  }

  if (event_type === 'game_over' || entry.game_over) {
    const winnerId = details?.winner as number | undefined;
    const isDraw = details?.is_draw as boolean | undefined;
    const winnerTeam = entry.result?.winner_team;

    if (isDraw) {
      return <span className="text-yellow-400 font-semibold">{t('cr_events.game_over_draw')}</span>;
    }
    if (winnerTeam) {
      return <span className="text-accent-cyan font-semibold">{t('cr_events.game_over_winner', { name: winnerTeam })}</span>;
    }
    if (winnerId != null) {
      return <span className="text-accent-cyan font-semibold">{t('cr_events.game_over_winner', { name: resolveName(winnerId) })}</span>;
    }
    return <span className="text-accent-cyan font-semibold">{t('cr.status_finished')}</span>;
  }

  const message = typeof details?.message === 'string' ? details.message : event_type;
  return <span className="text-text-muted/70 text-xs">{message}</span>;
}
