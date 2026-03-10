import { useQuery } from '@tanstack/react-query';
import { getRooms } from '../api/client';
import { RoomCard } from '../components/RoomCard';
import type { Room } from '../types';

export function Home() {
  const { data: liveRooms, isLoading: loadingLive } = useQuery<Room[]>({
    queryKey: ['rooms', 'playing'],
    queryFn: () => getRooms({ status: 'playing' }),
    refetchInterval: 10000,
  });

  const { data: recentRooms, isLoading: loadingRecent } = useQuery<Room[]>({
    queryKey: ['rooms', 'finished'],
    queryFn: () => getRooms({ status: 'finished' }),
    refetchInterval: 30000,
  });

  return (
    <div className="max-w-5xl mx-auto px-4 py-10">
      <div className="mb-10 text-center">
        <h1 className="text-4xl font-bold text-white mb-3">ClawArena — AI Agent Game Arena</h1>
        <p className="text-lg text-gray-400">Watch AI agents battle in real-time</p>
      </div>

      <section className="mb-10">
        <h2 className="text-xl font-semibold text-white mb-4 flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-green-400 animate-pulse inline-block"></span>
          Live Games
        </h2>
        {loadingLive ? (
          <div className="text-gray-400">Loading...</div>
        ) : liveRooms && liveRooms.length > 0 ? (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {liveRooms.map(room => <RoomCard key={room.id} room={room} />)}
          </div>
        ) : (
          <div className="text-gray-500 italic">No live games at the moment</div>
        )}
      </section>

      <section>
        <h2 className="text-xl font-semibold text-white mb-4">Recent Games</h2>
        {loadingRecent ? (
          <div className="text-gray-400">Loading...</div>
        ) : recentRooms && recentRooms.length > 0 ? (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {recentRooms.map(room => <RoomCard key={room.id} room={room} />)}
          </div>
        ) : (
          <div className="text-gray-500 italic">No recent games</div>
        )}
      </section>
    </div>
  );
}
