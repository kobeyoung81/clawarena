import React from 'react';
import { useParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { getRoom } from '../api/client';
import { useSSE } from '../hooks/useSSE';
import { useGameState } from '../hooks/useGameState';
import { useReplay } from '../hooks/useReplay';
import { AgentPanel } from '../components/AgentPanel';
import { ActionLog } from '../components/ActionLog';
import { ReplayControls } from '../components/ReplayControls';
import { ParticleCanvas } from '../components/effects/ParticleCanvas';
import { ShimmerCard } from '../components/effects/ShimmerLoader';
import { StatusPulse } from '../components/effects/StatusPulse';
import { TicTacToeBoard } from '../components/boards/TicTacToeBoard';
import { WerewolfBoard } from '../components/boards/WerewolfBoard';
import type { Room, GameStateResponse, WerewolfPlayer } from '../types';
import type { BoardProps } from '../components/boards/TicTacToeBoard';

const BOARD_COMPONENTS: Record<string, React.FC<BoardProps>> = {
  tic_tac_toe: TicTacToeBoard,
  werewolf: WerewolfBoard,
};

function formatGameName(name: string): string {
  return name.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase());
}

const STATUS_LABEL: Record<string, string> = {
  waiting: 'Waiting',
  ready_check: 'Ready Check',
  playing: 'Live',
  finished: 'Finished',
  cancelled: 'Cancelled',
};

const STATUS_COLOR: Record<string, string> = {
  waiting:     'rgba(255,255,255,0.3)',
  ready_check: '#ffc107',
  playing:     '#00e676',
  finished:    '#00e5ff',
  cancelled:   '#ff2d6b',
};

// ─── Room header banner ──────────────────────────────────────────────────────

function RoomHeader({ room, isReplayMode, isConnected }: {
  room: Room;
  isReplayMode: boolean;
  isConnected?: boolean;
}) {
  const statusColor = STATUS_COLOR[room.status] ?? 'rgba(255,255,255,0.3)';

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
              {STATUS_LABEL[room.status] ?? room.status}
            </div>

            {/* Replay badge */}
            {isReplayMode && (
              <span className="text-[10px] font-mono text-text-muted/50 bg-white/4 px-2 py-0.5 rounded border border-white/8">
                📼 REPLAY
              </span>
            )}

            {/* Live connection indicator */}
            {!isReplayMode && (
              <StatusPulse
                status={isConnected ? 'live' : 'waiting'}
                label={isConnected ? 'Connected' : 'Reconnecting'}
              />
            )}
          </div>
        </div>

        {/* Spacer */}
        <div className="flex-1" />

        {/* Player count */}
        <div className="text-right">
          <div className="text-xs font-mono text-text-muted/40">players</div>
          <div className="text-lg font-mono font-bold text-text-primary">{room.agents.length}</div>
        </div>
      </div>
    </div>
  );
}

// ─── Result banner (replay finished) ────────────────────────────────────────

function ResultBanner({ winner_team }: { winner_team?: string }) {
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
          {winner_team ? `${winner_team} Victory` : 'Game Over'}
        </div>
        <div className="text-xs font-mono text-accent-cyan/60 mt-1 uppercase tracking-widest">
          {winner_team ? '🏆 Winner Declared' : '🏁 Match Concluded'}
        </div>
      </div>
    </div>
  );
}

// ─── Live observer ───────────────────────────────────────────────────────────

function LiveObserver({ roomId, room }: { roomId: number; room: Room }) {
  const { latestEvent, isConnected } = useSSE(roomId);
  const { data: polledState } = useGameState(roomId);

  const gameState: GameStateResponse | null = (latestEvent as GameStateResponse | null) ?? polledState ?? null;
  const gameName = room.game_type?.name ?? '';
  const BoardComponent = BOARD_COMPONENTS[gameName];
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
              players={(gameState.players as WerewolfPlayer[]) ?? []}
              isReplay={false}
            />
          ) : (
            <div
              className="rounded-xl flex items-center justify-center h-64"
              style={{ background: 'rgba(255,255,255,0.02)', border: '1px solid rgba(255,255,255,0.06)' }}
            >
              {gameState ? (
                <span className="text-text-muted/30 text-xs font-mono italic">No board for this game type</span>
              ) : (
                <ShimmerCard />
              )}
            </div>
          )}

          {/* Turn counter */}
          {gameState?.turn !== undefined && (
            <div className="mt-2 flex items-center gap-2">
              <span className="text-[10px] font-mono text-text-muted/30 uppercase tracking-widest">Turn</span>
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
            liveEvents={gameState?.events ?? []}
            isReplay={false}
          />
        </div>
      </div>
    </>
  );
}

// ─── Replay observer ─────────────────────────────────────────────────────────

function ReplayObserver({ roomId, room }: { roomId: number; room: Room }) {
  const { history, step, total, isPlaying, speed, setSpeed, isLoading, goNext, goPrev, goTo, togglePlay } = useReplay(roomId);
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
        <div className="text-accent-mag text-sm font-mono">Failed to load history</div>
      </>
    );
  }

  const replayAgents = room.agents.map(ra => ({ ...ra }));

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
              })) as WerewolfPlayer[]}
              isReplay={true}
            />
          ) : (
            <div
              className="rounded-xl flex items-center justify-center h-64"
              style={{ background: 'rgba(255,255,255,0.02)', border: '1px solid rgba(255,255,255,0.06)' }}
            >
              <span className="text-text-muted/30 text-xs font-mono italic">
                {currentEntry ? 'No board for this game type' : 'No history data'}
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
  const { id } = useParams<{ id: string }>();
  const roomId = Number(id);

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
          <div className="text-accent-mag text-sm font-mono">Failed to load room #{roomId}</div>
        </div>
      </div>
    );
  }

  const isReplayMode = room.status === 'finished' || room.status === 'cancelled';

  return (
    <div className="max-w-6xl mx-auto px-4 py-6">
      {isReplayMode ? (
        <ReplayObserver roomId={roomId} room={room} />
      ) : (
        <LiveObserver roomId={roomId} room={room} />
      )}
    </div>
  );
}
