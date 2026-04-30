import { useI18n } from '../../i18n';
import type { GameEvent } from '../../types';

export interface CrPlayer {
  seat: number;
  name: string;
  alive: boolean;
  id: number;
}

export interface BoardProps {
  state: Record<string, unknown>;
  players?: CrPlayer[];
  currentEvent?: GameEvent;
  isReplay?: boolean;
}

interface CrStatePlayer {
  id: number;
  seat: number;
  hits: number;
  alive: boolean;
  gadget_count?: number;
}

interface CrState {
  players: CrStatePlayer[];
  bullet_index: number;
  total_bullets: number;
  current_turn: number;
  phase: 'playing' | 'finished';
  winner?: number;
  is_draw: boolean;
}

interface CrActionDetails {
  bullet?: string;
  self_shot?: boolean;
  target_hits?: number;
  hits_after?: number;
  gadget?: string;
}

interface PlayerCardProps {
  player?: CrStatePlayer;
  name: string;
  isTurn: boolean;
  isActor: boolean;
  isTarget: boolean;
  align: 'left' | 'right';
  aliveLabel: string;
  eliminatedLabel: string;
}

const MAX_HITS = 3;

function PlayerCard({
  player,
  name,
  isTurn,
  isActor,
  isTarget,
  align,
  aliveLabel,
  eliminatedLabel,
}: PlayerCardProps) {
  if (!player) {
    return <div className="hidden md:block" />;
  }

  const textAlign = align === 'right' ? 'md:text-right md:items-end' : 'items-start';
  const headerAlign = align === 'right' ? 'md:flex-row-reverse' : '';

  let border = '1px solid rgba(255,255,255,0.08)';
  let glow = '0 12px 30px rgba(0,0,0,0.18)';

  if (isTarget) {
    border = '1px solid rgba(239,68,68,0.6)';
    glow = '0 0 24px rgba(239,68,68,0.18)';
  } else if (isActor) {
    border = '1px solid rgba(0,229,255,0.55)';
    glow = '0 0 24px rgba(0,229,255,0.14)';
  } else if (isTurn) {
    border = '1px solid rgba(255,152,0,0.7)';
    glow = '0 0 24px rgba(255,152,0,0.14)';
  }

  return (
    <div
      className={`relative overflow-hidden rounded-2xl p-4 flex min-h-[170px] flex-col gap-3 transition-all ${textAlign}`}
      style={{
        background: player.alive ? 'rgba(255,255,255,0.04)' : 'rgba(255,255,255,0.02)',
        border,
        boxShadow: glow,
        opacity: player.alive ? 1 : 0.55,
      }}
    >
      {isTarget && (
        <span className="pointer-events-none absolute inset-0 rounded-2xl border border-red-400/40 animate-ping-slow" />
      )}
      {isActor && (
        <span className="pointer-events-none absolute inset-0 rounded-2xl border border-accent-cyan/20 animate-breathe" />
      )}

      <div
        className="pointer-events-none absolute inset-0"
        style={{
          background: align === 'left'
            ? 'radial-gradient(circle at top left, rgba(0,229,255,0.08), transparent 55%)'
            : 'radial-gradient(circle at top right, rgba(255,152,0,0.08), transparent 55%)',
        }}
      />

      <div className={`relative z-10 flex items-center justify-between gap-3 ${headerAlign}`}>
        <div className="min-w-0">
          <div className="text-[11px] font-mono uppercase tracking-[0.28em] text-text-muted/45">
            #{player.seat}
          </div>
          <div className="truncate text-lg font-semibold text-white">{name}</div>
        </div>
        {(isTurn || isActor || isTarget) && (
          <span
            className={`h-2.5 w-2.5 rounded-full ${isTurn ? 'animate-pulse' : 'animate-breathe'}`}
            style={{
              background: isTarget ? '#ef4444' : isActor ? '#00e5ff' : '#ff9800',
              boxShadow: isTarget
                ? '0 0 12px rgba(239,68,68,0.55)'
                : isActor
                  ? '0 0 12px rgba(0,229,255,0.55)'
                  : '0 0 12px rgba(255,152,0,0.55)',
            }}
          />
        )}
      </div>

      <div className={`relative z-10 flex flex-col gap-2 ${textAlign}`}>
        <div className={`flex gap-1.5 ${align === 'right' ? 'md:justify-end' : ''}`}>
          {Array.from({ length: MAX_HITS }, (_, i) => (
            <span
              key={i}
              className="text-lg leading-none"
              style={{ color: i < player.hits ? '#ef4444' : 'rgba(255,255,255,0.15)' }}
            >
              ♥
            </span>
          ))}
        </div>

        <div className={`flex items-center gap-3 text-[11px] font-mono uppercase tracking-[0.18em] ${align === 'right' ? 'md:justify-end' : ''}`}>
          <span style={{ color: player.alive ? '#4ade80' : '#ef4444' }}>
            {player.alive ? aliveLabel : eliminatedLabel}
          </span>
          <span className="text-text-muted/55">🎒 {player.gadget_count ?? 0}</span>
        </div>
      </div>
    </div>
  );
}

export default function ClawedRouletteBoard({ state, players, currentEvent }: BoardProps) {
  const { t } = useI18n();
  const s = state as unknown as CrState;
  const bulletIndex = s?.bullet_index ?? 0;
  const totalBullets = s?.total_bullets ?? 0;
  const currentTurn = s?.current_turn ?? 0;
  const phase = s?.phase ?? 'playing';
  const winnerId = s?.winner;
  const isDraw = s?.is_draw ?? false;
  const statePlayers = [...(s?.players ?? [])].sort((a, b) => a.seat - b.seat);

  const getName = (seatOrId: number, bySeat = true) => {
    const player = bySeat
      ? players?.find((candidate) => candidate.seat === seatOrId)
      : players?.find((candidate) => candidate.id === seatOrId);
    return player?.name ?? `Player ${seatOrId}`;
  };

  const leftPlayer = statePlayers[0];
  const rightPlayer = statePlayers[1];
  const leftSeat = leftPlayer?.seat;
  const rightSeat = rightPlayer?.seat;

  const actionEvent = currentEvent && (currentEvent.event_type === 'fire' || currentEvent.event_type === 'gadget_use')
    ? currentEvent
    : undefined;
  const actionDetails = (actionEvent?.details ?? {}) as CrActionDetails;
  const actorSeat = actionEvent?.actor?.seat;
  const targetSeat = actionEvent?.target?.seat;
  const actorName = actorSeat != null ? getName(actorSeat) : '';
  const targetName = targetSeat != null ? getName(targetSeat) : '';

  const isFireAction = actionEvent?.event_type === 'fire';
  const isGadgetAction = actionEvent?.event_type === 'gadget_use';
  const bulletType = actionDetails.bullet;
  const gadget = actionDetails.gadget;

  let statusMsg = t('cr.status_finished');
  if (phase === 'finished') {
    if (isDraw) {
      statusMsg = t('cr.status_draw');
    } else if (winnerId != null) {
      statusMsg = t('cr.status_winner', { name: getName(winnerId, false) });
    }
  } else {
    statusMsg = t('cr.status_turn', { name: getName(currentTurn) });
  }

  let actionSummary = statusMsg;
  if (isFireAction && actorName && targetName) {
    if (bulletType === 'live') {
      actionSummary = t('cr_events.fire_live', {
        name: actorName,
        target: targetName,
        hits: actionDetails.target_hits ?? '?',
      });
    } else if (actionDetails.self_shot) {
      actionSummary = t('cr_events.fire_blank_self', { name: actorName });
    } else {
      actionSummary = t('cr_events.fire_blank', { name: actorName, target: targetName });
    }
  } else if (isGadgetAction && actorName && gadget) {
    if (gadget === 'fish_chips') {
      actionSummary = t('cr_events.gadget_fish_chips', {
        name: actorName,
        hits: actionDetails.hits_after ?? 0,
      });
    } else if (gadget === 'goggles') {
      actionSummary = t('cr_events.gadget_goggles', { name: actorName });
    }
  }

  const direction = actorSeat === leftSeat && targetSeat === rightSeat
    ? 'right'
    : actorSeat === rightSeat && targetSeat === leftSeat
      ? 'left'
      : 'self';

  const gadgetIcon = gadget === 'fish_chips' ? '🐟' : gadget === 'goggles' ? '🔍' : '🎒';
  const gadgetName = gadget === 'fish_chips'
    ? t('cr.gadget_fish_chips')
    : gadget === 'goggles'
      ? t('cr.gadget_goggles')
      : gadget ?? '';

  return (
    <div
      className="rounded-xl p-4 md:p-5"
      style={{
        background: 'rgba(255,255,255,0.03)',
        border: '1px solid rgba(255,255,255,0.06)',
      }}
    >
      <div className="grid gap-4 md:grid-cols-[minmax(0,220px)_1fr_minmax(0,220px)] md:items-stretch">
        <PlayerCard
          player={leftPlayer}
          name={leftPlayer ? getName(leftPlayer.seat) : ''}
          isTurn={phase === 'playing' && leftPlayer?.seat === currentTurn}
          isActor={actorSeat != null && leftPlayer?.seat === actorSeat}
          isTarget={targetSeat != null && leftPlayer?.seat === targetSeat}
          align="left"
          aliveLabel={t('cr.alive')}
          eliminatedLabel={t('cr.eliminated')}
        />

        <div
          className="relative overflow-hidden rounded-2xl px-4 py-5 md:px-6"
          style={{
            border: '1px solid rgba(0,229,255,0.15)',
            background: 'linear-gradient(180deg, rgba(6,18,32,0.92), rgba(10,14,26,0.94))',
            minHeight: 300,
          }}
        >
          <div
            className="pointer-events-none absolute inset-0"
            style={{
              background: 'radial-gradient(circle at center, rgba(0,229,255,0.08), transparent 60%)',
            }}
          />

          <div className="relative z-10 flex h-full flex-col gap-5">
            <div className="flex flex-col gap-2">
              <div className="flex items-center justify-between gap-3">
                <span className="text-[10px] font-mono uppercase tracking-[0.28em] text-text-muted/45">
                  {t('cr.chamber')}
                </span>
                <span className="text-[10px] font-mono uppercase tracking-[0.22em] text-text-muted/45">
                  {bulletIndex}/{totalBullets}
                </span>
              </div>
              <div className="flex flex-wrap gap-1.5">
                {Array.from({ length: totalBullets }, (_, index) => {
                  const used = index < bulletIndex;
                  return (
                    <span
                      key={index}
                      className="inline-block h-3 w-3 rounded-full transition-all"
                      style={{
                        background: used ? 'rgba(255,255,255,0.08)' : 'rgba(255,152,0,0.72)',
                        border: used ? '1px solid rgba(255,255,255,0.06)' : '1px solid rgba(255,152,0,0.92)',
                        boxShadow: used ? 'none' : '0 0 10px rgba(255,152,0,0.2)',
                      }}
                    />
                  );
                })}
              </div>
            </div>

            <div className="flex flex-1 items-center justify-center">
              <div
                className="w-full max-w-2xl rounded-2xl border border-white/8 bg-[#09111f]/80 px-4 py-5 shadow-[0_0_30px_rgba(0,229,255,0.08)] backdrop-blur-sm animate-slide-in"
              >
                <div className="mb-3 flex items-center justify-center gap-2 text-[10px] font-mono uppercase tracking-[0.28em] text-accent-cyan/70">
                  <span className="h-1.5 w-1.5 rounded-full bg-accent-cyan/70" />
                  {t('cr.action_indicator')}
                </div>

                {isFireAction && leftPlayer && rightPlayer ? (
                  <div className="flex flex-col gap-4">
                    <div className="grid grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] items-center gap-3">
                      <div className={`truncate text-sm font-semibold ${leftSeat === actorSeat || leftSeat === targetSeat ? 'text-white' : 'text-text-muted/50'}`}>
                        {getName(leftSeat ?? 0)}
                      </div>
                      <div className="flex items-center justify-center gap-2 text-xl font-semibold text-accent-cyan">
                        {direction === 'right' && (
                          <>
                            <span className="animate-breathe">🔫</span>
                            <span>━━▶</span>
                          </>
                        )}
                        {direction === 'left' && (
                          <>
                            <span>◀━━</span>
                            <span className="animate-breathe">🔫</span>
                          </>
                        )}
                        {direction === 'self' && (
                          <span className="animate-breathe">🔫 ↺</span>
                        )}
                      </div>
                      <div className={`truncate text-right text-sm font-semibold ${rightSeat === actorSeat || rightSeat === targetSeat ? 'text-white' : 'text-text-muted/50'}`}>
                        {getName(rightSeat ?? 1)}
                      </div>
                    </div>
                    <div className="text-center text-xs text-text-muted/80">
                      {actionSummary}
                    </div>
                    <div className="flex justify-center">
                      <span
                        className="rounded-full px-3 py-1 text-[10px] font-mono uppercase tracking-[0.22em]"
                        style={{
                          color: bulletType === 'live' ? '#fecaca' : '#d1d5db',
                          background: bulletType === 'live' ? 'rgba(239,68,68,0.14)' : 'rgba(255,255,255,0.08)',
                          border: bulletType === 'live'
                            ? '1px solid rgba(239,68,68,0.28)'
                            : '1px solid rgba(255,255,255,0.08)',
                        }}
                      >
                        {bulletType === 'live' ? t('cr.round_live') : t('cr.round_blank')}
                      </span>
                    </div>
                  </div>
                ) : isGadgetAction && (leftPlayer || rightPlayer) ? (
                  <div className="flex flex-col gap-4">
                    <div className="grid grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] items-center gap-3">
                      <div className={`truncate text-sm font-semibold ${leftSeat === actorSeat ? 'text-white' : 'text-text-muted/50'}`}>
                        {leftPlayer ? getName(leftSeat ?? 0) : ''}
                      </div>
                      <div className="flex items-center justify-center gap-2 rounded-full border border-accent-cyan/15 bg-accent-cyan/5 px-4 py-2">
                        <span className="text-2xl animate-breathe">{gadgetIcon}</span>
                        <span className="text-xs font-mono uppercase tracking-[0.18em] text-accent-cyan/80">
                          {gadgetName}
                        </span>
                      </div>
                      <div className={`truncate text-right text-sm font-semibold ${rightSeat === actorSeat ? 'text-white' : 'text-text-muted/50'}`}>
                        {rightPlayer ? getName(rightSeat ?? 1) : ''}
                      </div>
                    </div>
                    <div className="text-center text-xs text-text-muted/80">
                      {actionSummary}
                    </div>
                  </div>
                ) : (
                  <div className="flex flex-col items-center gap-3 py-4 text-center">
                    <span className="h-2.5 w-2.5 rounded-full bg-[#ff9800] animate-pulse" />
                    <div className="text-lg font-semibold text-white">{statusMsg}</div>
                  </div>
                )}
              </div>
            </div>

            <div className="text-center text-sm font-semibold text-white">
              {statusMsg}
            </div>
          </div>
        </div>

        <PlayerCard
          player={rightPlayer}
          name={rightPlayer ? getName(rightPlayer.seat) : ''}
          isTurn={phase === 'playing' && rightPlayer?.seat === currentTurn}
          isActor={actorSeat != null && rightPlayer?.seat === actorSeat}
          isTarget={targetSeat != null && rightPlayer?.seat === targetSeat}
          align="right"
          aliveLabel={t('cr.alive')}
          eliminatedLabel={t('cr.eliminated')}
        />
      </div>
    </div>
  );
}
