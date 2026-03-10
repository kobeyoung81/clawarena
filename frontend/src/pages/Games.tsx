import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { getGameTypes } from '../api/client';
import type { GameType } from '../types';

function formatGameName(name: string): string {
  return name
    .split('_')
    .map(w => w.charAt(0).toUpperCase() + w.slice(1))
    .join('-');
}

export function Games() {
  const { data: games, isLoading, error } = useQuery<GameType[]>({
    queryKey: ['games'],
    queryFn: getGameTypes,
  });

  return (
    <div className="max-w-5xl mx-auto px-4 py-10">
      <h1 className="text-3xl font-bold text-white mb-8">Available Games</h1>

      {isLoading && <div className="text-gray-400">Loading games...</div>}
      {error && <div className="text-red-400">Failed to load games</div>}

      {games && (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
          {games.map(game => (
            <div
              key={game.id}
              className="bg-gray-800 rounded-xl p-5 border border-gray-700 hover:border-gray-500 transition-colors flex flex-col gap-3"
            >
              <h2 className="text-xl font-bold text-white">{formatGameName(game.name)}</h2>
              <p className="text-gray-400 text-sm flex-1">{game.description}</p>
              <div className="text-xs text-gray-500">
                Players: {game.min_players}–{game.max_players}
              </div>
              <Link
                to={`/rooms?game_type=${game.id}`}
                className="mt-auto text-sm bg-blue-600 hover:bg-blue-500 text-white px-4 py-2 rounded text-center transition-colors"
              >
                View Rooms
              </Link>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
