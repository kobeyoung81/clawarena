import { useI18n } from '../../i18n';
import type { GameEvent } from '../../types';
import fishAndChips from '../../assets/cr-fish-and-chips.svg';
import pistolLeft from '../../assets/cr-pistol-left.svg';
import pistolRight from '../../assets/cr-pistol-right.svg';

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
  turn_gadget_used?: boolean;
  phase: 'playing' | 'finished';
  winner?: number;
  is_draw: boolean;
}

interface CrActionDetails {
  bullet?: string;
  self_shot?: boolean;
  gadget?: string;
  peek_result?: string;
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

function GogglesGraphic() {
  return (
    <svg
      viewBox="0 0 160 90"
      className="w-full max-w-[170px] select-none drop-shadow-[0_0_24px_rgba(0,229,255,0.22)] animate-breathe"
      aria-hidden="true"
    >
      <rect x="12" y="26" width="50" height="34" rx="14" fill="#09111f" stroke="#00e5ff" strokeWidth="7" />
      <rect x="98" y="26" width="50" height="34" rx="14" fill="#09111f" stroke="#00e5ff" strokeWidth="7" />
      <rect x="62" y="38" width="36" height="10" rx="5" fill="#00e5ff" />
      <path d="M12 43H2" stroke="#00e5ff" strokeWidth="7" strokeLinecap="round" />
      <path d="M158 43h-10" stroke="#00e5ff" strokeWidth="7" strokeLinecap="round" />
      <circle cx="37" cy="43" r="10" fill="#7dd3fc" fillOpacity="0.35" />
      <circle cx="123" cy="43" r="10" fill="#7dd3fc" fillOpacity="0.35" />
    </svg>
  );
}

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
  const remainingHp = Math.max(0, MAX_HITS - player.hits);

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
      className={`relative flex min-h-[170px] flex-col gap-3 overflow-hidden rounded-2xl p-4 transition-all ${textAlign}`}
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
          {Array.from({ length: MAX_HITS }, (_, index) => (
            <span
              key={index}
              className="text-lg leading-none"
              style={{ color: index < remainingHp ? '#ef4444' : 'rgba(255,255,255,0.15)' }}
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

function CenterActionVisual({
  event,
  leftSeat,
}: {
  event?: GameEvent;
  leftSeat?: number;
}) {
  const { t } = useI18n();
  const details = (event?.details ?? {}) as CrActionDetails;

  let image = (
    <img
      src={pistolRight}
      alt=""
      className="w-full max-w-[250px] select-none drop-shadow-[0_0_24px_rgba(245,158,11,0.28)] animate-breathe"
      draggable={false}
    />
  );
  let caption = t('cr.action_waiting');

  if (event?.event_type === 'fire') {
    const targetSeat = event.target?.seat;
    const direction = targetSeat === leftSeat ? 'left' : 'right';
    const pistolAsset = direction === 'left' ? pistolLeft : pistolRight;
    const liveRound = details.bullet === 'live';
    image = (
      <img
        src={pistolAsset}
        alt=""
        className={`w-full max-w-[250px] select-none drop-shadow-[0_0_24px_rgba(245,158,11,0.28)] ${liveRound ? 'animate-breathe' : 'animate-slide-in'}`}
        draggable={false}
      />
    );
    caption = liveRound ? t('cr.action_fire_hit') : t('cr.action_fire_blank');
  }

  if (event?.event_type === 'gadget_use') {
    const gadget = details.gadget;
    if (gadget === 'fish_chips') {
      image = (
        <img
          src={fishAndChips}
          alt=""
          className="w-full max-w-[170px] select-none drop-shadow-[0_0_24px_rgba(34,197,94,0.24)] animate-breathe"
          draggable={false}
        />
      );
      caption = t('cr.action_fish_and_chips');
    }
    if (gadget === 'goggles') {
      const peekResult = details.peek_result ?? (typeof event.state?.last_peek === 'string' ? event.state.last_peek : undefined);
      image = <GogglesGraphic />;
      caption = peekResult === 'live'
        ? t('cr.action_next_live')
        : peekResult === 'blank'
          ? t('cr.action_next_blank')
          : t('cr.action_next_unknown');
    }
  }

  return (
    <div className="flex h-full items-center justify-center">
      <div className="flex min-h-[150px] flex-col items-center justify-center gap-3 text-center animate-slide-in">
        <div className="flex min-h-[110px] items-center justify-center">
          {image}
        </div>
        <div className="text-base font-semibold tracking-tight text-white">
          {caption}
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
    return player?.name ?? t('cr.player_fallback', { id: seatOrId });
  };

  const leftPlayer = statePlayers[0];
  const rightPlayer = statePlayers[1];
  const leftSeat = leftPlayer?.seat;

  const actionEvent = currentEvent && (currentEvent.event_type === 'fire' || currentEvent.event_type === 'gadget_use')
    ? currentEvent
    : undefined;
  const actorSeat = actionEvent?.actor?.seat;
  const targetSeat = actionEvent?.target?.seat;

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
              <div className="w-full max-w-2xl rounded-2xl border border-white/8 bg-[#09111f]/80 px-4 py-5 shadow-[0_0_30px_rgba(0,229,255,0.08)] backdrop-blur-sm animate-slide-in">
                <div className="min-h-[150px]">
                  <CenterActionVisual event={actionEvent} leftSeat={leftSeat} />
                </div>
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
