import type { GameEvent } from '../../types';

export interface ActionLogEntryProps {
  entry: GameEvent;
  players?: Array<{ agent_id: number; name: string }>;
}

// i18n TODO: replace hardcoded strings with t('cr_events.*') keys

export default function ClawedRouletteActionLog({ entry, players }: ActionLogEntryProps) {
  const { event_type, actor, target, details } = entry;

  const resolveName = (agentId?: number): string => {
    if (agentId == null) return 'Unknown';
    return players?.find(p => p.agent_id === agentId)?.name ?? `Agent ${agentId}`;
  };

  const actorName = resolveName(actor?.agent_id);
  const targetName = resolveName(target?.agent_id);

  if (event_type === 'game_start') {
    const liveCount = (details?.live_count as number) ?? '?';
    const blankCount = (details?.blank_count as number) ?? '?';
    return (
      <span className="text-accent-cyan font-semibold">
        🎮 Game Started — {liveCount} live, {blankCount} blank bullets
      </span>
    );
  }

  if (event_type === 'fire') {
    const bullet = details?.bullet as string;
    const selfShot = details?.self_shot as boolean;
    const targetHits = details?.target_hits as number | undefined;

    if (bullet === 'live') {
      return (
        <span style={{ color: '#ef4444' }}>
          🔴 {actorName} fires at {targetName} — HIT!
          {targetHits != null && ` (${targetHits} hits)`}
        </span>
      );
    }

    return (
      <span className="text-text-muted/60">
        ⚪ {actorName} fires at {targetName} — blank
        {selfShot && <span className="text-accent-cyan ml-1">(extra turn!)</span>}
      </span>
    );
  }

  if (event_type === 'gadget_use') {
    const gadget = details?.gadget as string;
    if (gadget === 'fish_chips') {
      const hitsAfter = details?.hits_after as number | undefined;
      return (
        <span className="text-green-400">
          🐟 {actorName} uses Fish &amp; Chips
          {hitsAfter != null && ` — heals to ${hitsAfter} hits`}
        </span>
      );
    }
    if (gadget === 'goggles') {
      return (
        <span className="text-yellow-400">
          🔍 {actorName} uses Goggles — peeked at next bullet
        </span>
      );
    }
    return (
      <span className="text-text-muted/70">
        🎒 {actorName} uses {gadget}
      </span>
    );
  }

  if (event_type === 'elimination') {
    return (
      <span style={{ color: '#ef4444' }} className="font-semibold">
        💀 {targetName} eliminated!
      </span>
    );
  }

  if (event_type === 'game_over' || entry.game_over) {
    const winnerId = details?.winner as number | undefined;
    const isDraw = details?.is_draw as boolean;
    const winnerTeam = entry.result?.winner_team;

    if (isDraw) {
      return <span className="text-yellow-400 font-semibold">🏁 Game Over — Draw!</span>;
    }
    if (winnerTeam) {
      return <span className="text-accent-cyan font-semibold">🏆 Game Over — {winnerTeam} wins!</span>;
    }
    if (winnerId != null) {
      const winnerName = resolveName(winnerId);
      return <span className="text-accent-cyan font-semibold">🏆 Game Over — {winnerName} wins!</span>;
    }
    return <span className="text-accent-cyan font-semibold">🏁 Game Over</span>;
  }

  // Fallback for unknown events
  const message = typeof details?.message === 'string' ? details.message : event_type;
  return <span className="text-text-muted/70 text-xs">{message}</span>;
}
