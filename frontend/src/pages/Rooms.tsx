import { useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { getRooms, getGameTypes } from '../api/client';
import { RoomCard } from '../components/RoomCard';
import type { Room, GameType, RoomStatus } from '../types';

const STATUSES: Array<{ value: string; label: string }> = [
  { value: '', label: 'All' },
  { value: 'waiting', label: 'Waiting' },
  { value: 'ready_check', label: 'Ready Check' },
  { value: 'playing', label: 'Playing' },
  { value: 'finished', label: 'Finished' },
  { value: 'cancelled', label: 'Cancelled' },
];

export function Rooms() {
  const [searchParams] = useSearchParams();
  const defaultGameType = searchParams.get('game_type') ?? '';

  const [status, setStatus] = useState<string>('');
  const [gameTypeId, setGameTypeId] = useState<string>(defaultGameType);

  const params: Record<string, string> = {};
  if (status) params.status = status as RoomStatus;
  if (gameTypeId) params.game_type_id = gameTypeId;

  const { data: rooms, isLoading, error } = useQuery<Room[]>({
    queryKey: ['rooms', params],
    queryFn: () => getRooms(params),
    refetchInterval: 5000,
  });

  const { data: gameTypes } = useQuery<GameType[]>({
    queryKey: ['games'],
    queryFn: getGameTypes,
  });

  return (
    <div className="max-w-5xl mx-auto px-4 py-10">
      <h1 className="text-3xl font-bold text-white mb-6">Game Rooms</h1>

      {/* Filters */}
      <div className="flex flex-wrap gap-3 mb-6">
        <select
          value={status}
          onChange={e => setStatus(e.target.value)}
          className="bg-gray-700 text-white text-sm px-3 py-2 rounded border border-gray-600 focus:border-blue-500 focus:outline-none"
        >
          {STATUSES.map(s => (
            <option key={s.value} value={s.value}>{s.label}</option>
          ))}
        </select>

        <select
          value={gameTypeId}
          onChange={e => setGameTypeId(e.target.value)}
          className="bg-gray-700 text-white text-sm px-3 py-2 rounded border border-gray-600 focus:border-blue-500 focus:outline-none"
        >
          <option value="">All Games</option>
          {gameTypes?.map(g => (
            <option key={g.id} value={String(g.id)}>{g.name}</option>
          ))}
        </select>
      </div>

      {isLoading && <div className="text-gray-400">Loading rooms...</div>}
      {error && <div className="text-red-400">Failed to load rooms</div>}

      {rooms && rooms.length > 0 ? (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {rooms.map(room => <RoomCard key={room.id} room={room} />)}
        </div>
      ) : !isLoading ? (
        <div className="text-gray-500 italic py-8 text-center">No rooms match your filters</div>
      ) : null}
    </div>
  );
}
