import React from 'react';
import { useSSE } from '../../hooks/useSSE';
import { useReplay } from '../../hooks/useReplay';
import { AgentPanel } from '../../components/AgentPanel';
import { ReplayControls } from '../../components/ReplayControls';
import { RoomHeader, ResultBanner } from '../../components/RoomHeader';
import { ShimmerCard } from '../../components/effects/ShimmerLoader';
import { useI18n } from '../../i18n';
import type { Room, GameEvent, ClawedWolfPlayer } from '../../types';

import Board from './Board';
import ActionLog from './ActionLog';

// ─── Event-sourced ActionLog wrapper ────────────────────────────────────────

function EventActionLog({
  events,
  currentStep,
  isReplay,
  players,
}: {
  events: GameEvent[];
  currentStep?: number;
  isReplay: boolean;
  players: Array<{ agent_id: number; name: string }>;
}) {
  const { t } = useI18n();
  const bottomRef = React.useRef<HTMLDivElement>(null);

  React.useEffect(() => {
    if (!isReplay) {
      bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
    }
  }, [events, isReplay]);

  return (
    <div className="glass rounded-xl border-white/8 flex flex-col h-64 overflow-hidden">
      <div className="px-3 py-2 border-b border-white/6 flex items-center gap-2">
        <span className="text-xs font-mono font-semibold text-text-muted uppercase tracking-widest">
          {t('action_log.title')}
        </span>
        <span className="flex h-1.5 w-1.5 rounded-full bg-accent-cyan/60" />
      </div>

      <div className="flex-1 overflow-y-auto p-2 flex flex-col gap-1">
        {events.length > 0 ? (
          events.map((entry, idx) => {
            const isCurrent = isReplay && idx === currentStep;
            return (
              <div
                key={entry.seq}
                className="text-sm rounded px-2 py-1.5 animate-slide-in"
                style={{
                  background: isCurrent ? 'rgba(0,229,255,0.08)' : 'transparent',
                  borderLeft: isCurrent ? '2px solid rgba(0,229,255,0.6)' : '2px solid transparent',
                }}
              >
                <span className="text-text-muted/50 font-mono mr-2">#{entry.seq}</span>
                <ActionLog entry={entry} players={players} />
              </div>
            );
          })
        ) : (
          <div className="flex-1 flex items-center justify-center">
            <span className="text-text-muted/30 text-xs font-mono italic">{t('action_log.empty')}</span>
          </div>
        )}
        <div ref={bottomRef} />
      </div>
    </div>
  );
}

// ─── Live observer ──────────────────────────────────────────────────────────

function LiveObserver({ room }: { room: Room }) {
  const { t } = useI18n();
  const { events, latestEvent, isConnected } = useSSE(room.id);
  const currentState = latestEvent?.state;

  const livePlayers = (latestEvent?.agents ?? room.agents ?? []).map((a, i) => ({
    agent_id: a.agent_id,
    name: a.name,
    seat: i,
    alive: true,
    id: a.agent_id,
  }));

  return (
    <>
      <RoomHeader room={room} isReplayMode={false} isConnected={isConnected} />

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* Board -- 2/3 width */}
        <div className="lg:col-span-2">
          {currentState ? (
            <Board
              state={currentState}
              players={livePlayers.map(a => ({
                seat: a.seat, name: a.name, alive: true, id: a.agent_id,
              })) as ClawedWolfPlayer[]}
              isReplay={false}
            />
          ) : (
            <div
              className="rounded-xl flex items-center justify-center h-64"
              style={{ background: 'rgba(255,255,255,0.02)', border: '1px solid rgba(255,255,255,0.06)' }}
            >
              <ShimmerCard />
            </div>
          )}

          {/* Seq counter */}
          {latestEvent && (
            <div className="mt-2 flex items-center gap-2">
              <span className="text-[10px] font-mono text-text-muted/30 uppercase tracking-widest">{t('observer.turn')}</span>
              <span className="text-xs font-mono text-accent-cyan/70">#{latestEvent.seq}</span>
            </div>
          )}
        </div>

        {/* Side panel -- 1/3 width */}
        <div className="flex flex-col gap-3">
          <AgentPanel
            agents={latestEvent?.agents ?? room.agents ?? []}
            pendingAction={latestEvent?.pending_action ?? null}
          />
          <EventActionLog
            events={events}
            isReplay={false}
            players={livePlayers}
          />
        </div>
      </div>
    </>
  );
}

// ─── Replay observer ────────────────────────────────────────────────────────

function ReplayObserver({ room, gameId }: { room: Room; gameId?: number }) {
  const { t } = useI18n();
  const { history, step, total, isPlaying, speed, setSpeed, isLoading, goNext, goPrev, goTo, togglePlay } = useReplay(room.id, gameId, !gameId);
  const currentEvent = history?.events[step];

  if (isLoading) {
    return (
      <>
        <RoomHeader room={room} isReplayMode={true} />
        <ShimmerCard />
      </>
    );
  }

  if (!history) {
    return (
      <>
        <RoomHeader room={room} isReplayMode={true} />
        <div className="text-accent-mag text-sm font-mono">{t('observer.error_history')}</div>
      </>
    );
  }

  const replayAgents = room.agents.length > 0
    ? room.agents.map(ra => ({ ...ra }))
    : (history?.players ?? []).map((p, i) => ({
        id: i,
        name: p.name,
        agent_id: p.agent_id,
        slot: p.slot ?? p.seat ?? i,
        score: 0,
        ready: false,
      }));

  const replayPlayers = (history?.players ?? []).map((p) => ({
    agent_id: p.agent_id,
    name: p.name,
  }));

  return (
    <>
      <RoomHeader room={room} isReplayMode={true} />

      {history.result && (
        <ResultBanner winner_team={history.result.winner_team} />
      )}

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4 mb-4">
        {/* Board */}
        <div className="lg:col-span-2">
          {currentEvent ? (
            <Board
              state={currentEvent.state}
              players={history.players.map(p => ({
                seat: p.seat ?? p.slot ?? 0,
                name: p.name,
                alive: true,
                role: p.role,
                id: p.agent_id,
              })) as ClawedWolfPlayer[]}
              isReplay={true}
            />
          ) : (
            <div
              className="rounded-xl flex items-center justify-center h-64"
              style={{ background: 'rgba(255,255,255,0.02)', border: '1px solid rgba(255,255,255,0.06)' }}
            >
              <span className="text-text-muted/30 text-xs font-mono italic">
                {t('observer.no_history')}
              </span>
            </div>
          )}
        </div>

        {/* Side panel */}
        <div className="flex flex-col gap-3">
          <AgentPanel
            agents={replayAgents}
            pendingAction={null}
            replayPlayers={history.players}
          />
          <EventActionLog
            events={history.events.slice(0, step + 1)}
            currentStep={step}
            isReplay={true}
            players={replayPlayers}
          />
        </div>
      </div>

      <ReplayControls
        step={step}
        total={total}
        isPlaying={isPlaying}
        speed={speed}
        onPrev={goPrev}
        onNext={goNext}
        onPlay={togglePlay}
        onJump={goTo}
        onSpeedChange={setSpeed}
      />
    </>
  );
}

// ─── Root observer ──────────────────────────────────────────────────────────

export default function Observer({ room, gameId }: { room: Room; gameId?: number }) {
  const isReplayMode = room.status === 'intermission' || room.status === 'closed';
  return (
    <div className="max-w-6xl mx-auto px-4 py-6">
      {isReplayMode
        ? <ReplayObserver room={room} gameId={gameId} />
        : <LiveObserver room={room} />
      }
    </div>
  );
}
