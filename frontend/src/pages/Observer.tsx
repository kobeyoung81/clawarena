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
import { TicTacToeBoard } from '../components/boards/TicTacToeBoard';
import { WerewolfBoard } from '../components/boards/WerewolfBoard';
import type { Room, GameStateResponse, WerewolfPlayer } from '../types';
import type { BoardProps } from '../components/boards/TicTacToeBoard';

const BOARD_COMPONENTS: Record<string, React.FC<BoardProps>> = {
  tic_tac_toe: TicTacToeBoard,
  werewolf: WerewolfBoard,
};

const STATUS_BADGE: Record<string, string> = {
  waiting: 'bg-gray-500',
  ready_check: 'bg-yellow-500',
  playing: 'bg-green-500',
  finished: 'bg-blue-500',
  cancelled: 'bg-red-500',
};

function LiveObserver({ roomId, room }: { roomId: number; room: Room }) {
  const { latestEvent, isConnected } = useSSE(roomId);
  const { data: polledState } = useGameState(roomId);

  const gameState: GameStateResponse | null = (latestEvent as GameStateResponse | null) ?? polledState ?? null;
  const gameName = room.game_type?.name ?? '';
  const BoardComponent = BOARD_COMPONENTS[gameName];

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-3">
        <span className="text-gray-400 text-sm">
          {isConnected ? '🟢 Live' : '🔴 Reconnecting...'}
        </span>
        {gameState?.turn !== undefined && (
          <span className="text-gray-400 text-sm">Turn {gameState.turn}</span>
        )}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <div className="lg:col-span-2">
        {BoardComponent != null && gameState ? (
            <BoardComponent
              state={gameState.state}
              players={(gameState.players as WerewolfPlayer[]) ?? []}
              isReplay={false}
            />
          ) : (
            <div className="bg-gray-800 rounded-lg p-8 text-center text-gray-500">
              {gameState ? 'No board for this game type' : 'Waiting for game state...'}
            </div>
          )}
        </div>

        <div className="flex flex-col gap-4">
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
    </div>
  );
}

function ReplayObserver({ roomId, room }: { roomId: number; room: Room }) {
  const { history, step, total, isPlaying, isLoading, goNext, goPrev, goTo, togglePlay } = useReplay(roomId);
  const gameName = room.game_type?.name ?? '';
  const BoardComponent = BOARD_COMPONENTS[gameName];
  const currentEntry = history?.timeline[step];

  if (isLoading) {
    return <div className="text-gray-400">Loading replay...</div>;
  }

  if (!history) {
    return <div className="text-red-400">Failed to load history</div>;
  }

  const resultBanner = history.result ? (
    <div className="bg-blue-900 border border-blue-500 rounded-lg p-4 text-center">
      <div className="text-lg font-bold text-white">
        {history.result.winner_team
          ? `${history.result.winner_team} wins! 🏆`
          : 'Game Over'}
      </div>
    </div>
  ) : null;

  const replayAgents = room.agents.map(ra => ({
    ...ra,
    score: ra.score,
  }));

  return (
    <div className="flex flex-col gap-4">
      {resultBanner}

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
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
            <div className="bg-gray-800 rounded-lg p-8 text-center text-gray-500">
              {currentEntry ? 'No board for this game type' : 'No history data'}
            </div>
          )}
        </div>

        <div className="flex flex-col gap-4">
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
        onPrev={goPrev}
        onNext={goNext}
        onPlay={togglePlay}
        onJump={goTo}
      />
    </div>
  );
}

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
      <div className="max-w-5xl mx-auto px-4 py-10 text-gray-400">
        Loading room...
      </div>
    );
  }

  if (error || !room) {
    return (
      <div className="max-w-5xl mx-auto px-4 py-10 text-red-400">
        Failed to load room #{roomId}
      </div>
    );
  }

  const isReplayMode = room.status === 'finished' || room.status === 'cancelled';

  return (
    <div className="max-w-6xl mx-auto px-4 py-6">
      {/* Room Header */}
      <div className="flex flex-wrap items-center gap-3 mb-6">
        <h1 className="text-2xl font-bold text-white">
          {room.game_type?.name?.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase())} — Room #{room.id}
        </h1>
        <span className={`text-xs font-semibold px-2 py-1 rounded ${STATUS_BADGE[room.status] ?? 'bg-gray-500'} text-white`}>
          {room.status.replace('_', ' ')}
        </span>
        {isReplayMode && (
          <span className="text-xs text-gray-400 bg-gray-700 px-2 py-1 rounded">📼 Replay Mode</span>
        )}
      </div>

      {isReplayMode ? (
        <ReplayObserver roomId={roomId} room={room} />
      ) : (
        <LiveObserver roomId={roomId} room={room} />
      )}
    </div>
  );
}
