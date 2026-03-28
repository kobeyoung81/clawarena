import React, { useState, useEffect } from 'react';
import { useParams, useSearchParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { getRoom, getRoomHistory } from '../api/client';
import { useSSE } from '../hooks/useSSE';
import { useReplay } from '../hooks/useReplay';
import { AgentPanel } from '../components/AgentPanel';
import { ActionLog } from '../components/ActionLog';
import { ReplayControls } from '../components/ReplayControls';
import { ParticleCanvas } from '../components/effects/ParticleCanvas';
import { ShimmerCard } from '../components/effects/ShimmerLoader';
import { StatusPulse } from '../components/effects/StatusPulse';
import { TicTacToeBoard } from '../components/boards/TicTacToeBoard';
import { ClawedWolfBoard } from '../components/boards/ClawedWolfBoard';
import { useI18n } from '../i18n';
import type { Room, GameStateResponse, ClawedWolfPlayer, HistoryTimeline } from '../types';
import type { BoardProps } from '../components/boards/TicTacToeBoard';

const BOARD_COMPONENTS: Record<string, React.FC<BoardProps>> = {
  tic_tac_toe: TicTacToeBoard,
  clawedwolf: ClawedWolfBoard,
};

function formatGameName(name: string): string {
  return name.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase());
}

const STATUS_COLOR: Record<string, string> = {
  waiting:      'rgba(255,255,255,0.3)',
  ready_check:  '#ffc107',
  playing:      '#00e676',
  intermission: '#ffc107',
  closed:       'rgba(255,255,255,0.2)',
};

// ─── Room header banner ──────────────────────────────────────────────────────

function RoomHeader({ room, isReplayMode, isConnected }: {
  room: Room;
  isReplayMode: boolean;
  isConnected?: boolean;
}) {
  const { t } = useI18n();
  const statusColor = STATUS_COLOR[room.status] ?? 'rgba(255,255,255,0.3)';
  const statusLabel = t(`status.${room.status}`) !== `status.${room.status}`
    ? t(`status.${room.status}`)
    : room.status;

  return (
    <div
      className="relative rounded-xl overflow-hidden mb-5"
      style={{
        background: 'rgba(10,14,26,0.7)',
        border: `1px solid ${statusColor}22`,
        backdropFilter: 'blur(10px)',
      }}
    >
      {/* Subtle accent line at top */}
      <div className="h-px w-full" style={{ background: `linear-gradient(90deg, transparent, ${statusColor}60, transparent)` }} />

      <div className="px-5 py-4 flex flex-wrap items-center gap-4">
        {/* Game type + room number */}
        <div>
          <h1 className="text-xl font-bold tracking-tight text-text-primary">
            {formatGameName(room.game_type?.name ?? '')}
            <span className="ml-2 text-text-muted/40 font-mono text-base font-normal">
              #{room.id}
            </span>
          </h1>
          <div className="flex items-center gap-3 mt-1">
            {/* Status pill */}
            <div
              className="flex items-center gap-1.5 text-[10px] font-mono font-semibold uppercase tracking-widest px-2 py-0.5 rounded"
              style={{ background: `${statusColor}14`, color: statusColor, border: `1px solid ${statusColor}30` }}
            >
              {room.status === 'playing' && (
                <span className="w-1.5 h-1.5 rounded-full animate-ping-slow" style={{ background: statusColor }} />
              )}
              {statusLabel}
            </div>

            {/* Replay badge */}
            {isReplayMode && (
              <span className="text-[10px] font-mono text-text-muted/50 bg-white/4 px-2 py-0.5 rounded border border-white/8">
                📼 {t('observer.replay_badge')}
              </span>
            )}

            {/* Live connection indicator */}
            {!isReplayMode && (
              <StatusPulse
                status={isConnected ? 'live' : 'waiting'}
                label={isConnected ? t('observer.connected') : t('observer.reconnecting')}
              />
            )}
          </div>
        </div>

        {/* Spacer */}
        <div className="flex-1" />

        {/* Player count */}
        <div className="text-right">
          <div className="text-xs font-mono text-text-muted/40">{t('observer.players_label')}</div>
          <div className="text-lg font-mono font-bold text-text-primary">{room.agents.length}</div>
        </div>
      </div>
    </div>
  );
}

// ─── Result banner (replay finished) ────────────────────────────────────────

function ResultBanner({ winner_team }: { winner_team?: string }) {
  const { t } = useI18n();
  return (
    <div
      className="relative rounded-xl overflow-hidden mb-4"
      style={{
        background: 'rgba(0,229,255,0.05)',
        border: '1px solid rgba(0,229,255,0.25)',
      }}
    >
      <ParticleCanvas density={15} speed={0.2} color="#00e5ff" className="opacity-20 rounded-xl" />
      <div className="relative z-10 py-5 text-center">
        <div className="text-2xl font-bold tracking-tight text-text-primary">
          {winner_team ? t('observer.victory', { team: winner_team }) : t('observer.game_over')}
        </div>
        <div className="text-xs font-mono text-accent-cyan/60 mt-1 uppercase tracking-widest">
          {winner_team ? `🏆 ${t('observer.winner_declared')}` : `🏁 ${t('observer.match_concluded')}`}
        </div>
      </div>
    </div>
  );
}

// ─── Live observer ───────────────────────────────────────────────────────────

function LiveObserver({ roomId, room }: { roomId: number; room: Room }) {
  const { t } = useI18n();
  const { latestEvent, isConnected } = useSSE(roomId);

  // Derive game state from SSE events (initial + subsequent)
  const [gameState, setGameState] = useState<GameStateResponse | null>(null);
  const [liveTimeline, setLiveTimeline] = useState<HistoryTimeline[]>([]);
  const [historyLoaded, setHistoryLoaded] = useState(false);

  // Load existing game history on mount for full action log
  const gameName = room.game_type?.name ?? '';
  const BoardComponent = BOARD_COMPONENTS[gameName];
  const { data: historyData } = useQuery({
    queryKey: ['roomHistoryLive', roomId],
    queryFn: () => getRoomHistory(roomId),
    staleTime: Infinity,
    refetchOnWindowFocus: false,
  });

  // Bootstrap: load historical timeline entries directly
  useEffect(() => {
    if (historyData?.timeline && !historyLoaded) {
      const timeline = historyData.timeline as HistoryTimeline[];
      if (timeline.length > 0) {
        setLiveTimeline(timeline);
      }
      setHistoryLoaded(true);
    }
  }, [historyData, historyLoaded]);

  useEffect(() => {
    if (latestEvent) {
      const evt = latestEvent as Record<string, unknown>;

      // Build GameStateResponse from the SSE event payload
      if (evt.state != null) {
        setGameState({
          room_id: (evt.room_id as number) ?? roomId,
          status: (evt.status as GameStateResponse['status']) ?? room.status,
          turn: (evt.turn as number) ?? 0,
          state: evt.state as Record<string, unknown>,
          pending_action: (evt.pending_action as GameStateResponse['pending_action']) ?? null,
          agents: (evt.agents as GameStateResponse['agents']) ?? room.agents ?? [],
          phase: (evt.phase as string) ?? undefined,
        });
      }

      // Build structured timeline entry from SSE event
      const evtEvents = Array.isArray(evt.events) ? (evt.events as any[]) : [];
      if (evt.turn != null && (evtEvents.length > 0 || evt.action != null)) {
        const entry: HistoryTimeline = {
          turn: evt.turn as number,
          agent_id: evt.agent_id as number | undefined,
          action: evt.action as Record<string, unknown> | undefined,
          state: (evt.state as Record<string, unknown>) ?? {},
          events: evtEvents.map(e =>
            typeof e === 'string'
              ? { type: 'event', message: e }
              : { type: e.type ?? 'event', message: e.message ?? JSON.stringify(e) }
          ),
          created_at: new Date().toISOString(),
        };
        setLiveTimeline(prev => [...prev, entry]);
      }
    }
  }, [latestEvent, roomId, room.status, room.agents]);

  // Build players list for action log and board from agents
  const livePlayers = (gameState?.agents ?? room.agents ?? []).map((a, i) => ({
    agent_id: a.agent_id,
    name: a.name,
    seat: i,
    alive: true,
    id: a.agent_id,
  }));

  const phase = (gameState?.state as { phase?: string })?.phase ?? gameState?.phase ?? '';

  return (
    <>
      <RoomHeader room={room} isReplayMode={false} isConnected={isConnected} />

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* Board — 2/3 width */}
        <div className="lg:col-span-2">
          {BoardComponent != null && gameState ? (
            <BoardComponent
              state={gameState.state}
              players={
                gameName === 'tic_tac_toe'
                  ? livePlayers.map(a => ({
                      seat: a.seat, name: a.name, alive: true, id: a.agent_id,
                    })) as ClawedWolfPlayer[]
                  : (gameState.players as ClawedWolfPlayer[]) ?? []
              }
              isReplay={false}
            />
          ) : (
            <div
              className="rounded-xl flex items-center justify-center h-64"
              style={{ background: 'rgba(255,255,255,0.02)', border: '1px solid rgba(255,255,255,0.06)' }}
            >
              {gameState ? (
                <span className="text-text-muted/30 text-xs font-mono italic">{t('observer.no_board')}</span>
              ) : (
                <ShimmerCard />
              )}
            </div>
          )}

          {/* Turn counter */}
          {gameState?.turn !== undefined && (
            <div className="mt-2 flex items-center gap-2">
              <span className="text-[10px] font-mono text-text-muted/30 uppercase tracking-widest">{t('observer.turn')}</span>
              <span className="text-xs font-mono text-accent-cyan/70">{gameState.turn}</span>
              {phase && (
                <>
                  <span className="text-text-muted/20">·</span>
                  <span className="text-[10px] font-mono text-text-muted/40 uppercase">{phase.replace('_', ' ')}</span>
                </>
              )}
            </div>
          )}
        </div>

        {/* Side panel — 1/3 width */}
        <div className="flex flex-col gap-3">
          <AgentPanel
            agents={gameState?.agents ?? room.agents ?? []}
            pendingAction={gameState?.pending_action ?? null}
          />
          <ActionLog
            timeline={liveTimeline}
            isReplay={false}
            gameType={gameName}
            players={livePlayers}
          />
        </div>
      </div>
    </>
  );
}

// ─── Replay observer ─────────────────────────────────────────────────────────

function ReplayObserver({ roomId, room, gameId }: { roomId: number; room: Room; gameId?: number }) {
  const { t } = useI18n();
  const { history, step, total, isPlaying, speed, setSpeed, isLoading, goNext, goPrev, goTo, togglePlay } = useReplay(roomId, gameId);
  const gameName = room.game_type?.name ?? '';
  const BoardComponent = BOARD_COMPONENTS[gameName];
  const currentEntry = history?.timeline[step];

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
          {BoardComponent != null && currentEntry ? (
            <BoardComponent
              state={currentEntry.state}
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
                {currentEntry ? t('observer.no_board') : t('observer.no_history')}
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
          <ActionLog
            timeline={history.timeline.slice(0, step + 1)}
            currentStep={step}
            isReplay={true}
            gameType={gameName}
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

// ─── Root observer page ──────────────────────────────────────────────────────

export function Observer() {
  const { t } = useI18n();
  const { id } = useParams<{ id: string }>();
  const [searchParams] = useSearchParams();
  const roomId = Number(id);
  const gameId = searchParams.get('game') ? Number(searchParams.get('game')) : undefined;

  const { data: room, isLoading, error } = useQuery<Room>({
    queryKey: ['room', roomId],
    queryFn: () => getRoom(roomId),
    refetchInterval: 3000,
  });

  if (isLoading) {
    return (
      <div className="max-w-6xl mx-auto px-4 py-10">
        <ShimmerCard />
      </div>
    );
  }

  if (error || !room) {
    return (
      <div className="max-w-6xl mx-auto px-4 py-10">
        <div
          className="rounded-xl p-6 text-center"
          style={{ background: 'rgba(255,45,107,0.06)', border: '1px solid rgba(255,45,107,0.2)' }}
        >
          <div className="text-accent-mag text-sm font-mono">{t('observer.error', { id: String(roomId) })}</div>
        </div>
      </div>
    );
  }

  const isReplayMode = room.status === 'intermission' || room.status === 'closed';

  return (
    <div className="max-w-6xl mx-auto px-4 py-6">
      {isReplayMode ? (
        <ReplayObserver roomId={roomId} room={room} gameId={gameId} />
      ) : (
        <LiveObserver roomId={roomId} room={room} />
      )}
    </div>
  );
}
