import { useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { getRooms, getGameTypes } from '../api/client';
import { RoomCard } from '../components/RoomCard';
import { useI18n } from '../i18n';
import type { Room, GameType, RoomStatus } from '../types';

export function Rooms() {
  const { t } = useI18n();
  const [searchParams] = useSearchParams();
  const defaultGameType = searchParams.get('game_type') ?? '';

  const [status, setStatus] = useState<string>('');
  const [gameTypeId, setGameTypeId] = useState<string>(defaultGameType);

  const STATUSES: Array<{ value: string; label: string }> = [
    { value: '', label: t('rooms.all') },
    { value: 'waiting', label: t('rooms.waiting') },
    { value: 'ready_check', label: t('rooms.ready_check') },
    { value: 'playing', label: t('rooms.playing') },
    { value: 'post_game', label: t('rooms.post_game') ?? 'Post Game' },
    { value: 'dead', label: t('rooms.dead') ?? 'Dead' },
  ];

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
      <h1 className="text-3xl font-bold text-white mb-6">{t('rooms.title')}</h1>

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
          <option value="">{t('rooms.all_games')}</option>
          {gameTypes?.map(g => (
            <option key={g.id} value={String(g.id)}>{g.name}</option>
          ))}
        </select>
      </div>

      {isLoading && <div className="text-gray-400">{t('rooms.loading')}</div>}
      {error && <div className="text-red-400">{t('rooms.error')}</div>}

      {rooms && rooms.length > 0 ? (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {rooms.map(room => <RoomCard key={room.id} room={room} />)}
        </div>
      ) : !isLoading ? (
        <div className="text-gray-500 italic py-8 text-center">{t('rooms.no_match')}</div>
      ) : null}
    </div>
  );
}
