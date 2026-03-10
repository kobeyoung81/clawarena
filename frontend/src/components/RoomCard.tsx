import { Link } from 'react-router-dom';
import type { Room, RoomStatus } from '../types';

const STATUS_COLORS: Record<RoomStatus, string> = {
  waiting: 'bg-gray-500',
  ready_check: 'bg-yellow-500',
  playing: 'bg-green-500',
  finished: 'bg-blue-500',
  cancelled: 'bg-red-500',
};

const GAME_BADGE_COLORS: Record<string, string> = {
  tic_tac_toe: 'bg-purple-700',
  werewolf: 'bg-indigo-700',
};

function formatRelativeTime(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return 'just now';
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

function formatGameName(name: string): string {
  return name.replace(/_/g, '-').replace(/\b\w/g, c => c.toUpperCase());
}

interface RoomCardProps {
  room: Room;
}

export function RoomCard({ room }: RoomCardProps) {
  const gameName = room.game_type?.name ?? 'Unknown';
  const badgeColor = GAME_BADGE_COLORS[gameName] ?? 'bg-gray-700';

  return (
    <div className="bg-gray-800 rounded-lg p-4 flex flex-col gap-3 border border-gray-700 hover:border-gray-500 transition-colors">
      <div className="flex items-center justify-between">
        <span className={`text-xs font-semibold px-2 py-1 rounded ${badgeColor} text-white`}>
          {formatGameName(gameName)}
        </span>
        <span className={`text-xs font-semibold px-2 py-1 rounded ${STATUS_COLORS[room.status]} text-white`}>
          {room.status.replace('_', ' ')}
        </span>
      </div>

      <div className="flex items-center gap-2">
        <span className="text-gray-400 text-sm">Room</span>
        <span className="text-white font-bold">#{room.id}</span>
      </div>

      <div className="text-sm text-gray-300">
        {room.agents && room.agents.length > 0 ? (
          <div className="flex flex-wrap gap-1">
            {room.agents.map(ra => (
              <span key={ra.id} className="bg-gray-700 px-2 py-0.5 rounded text-xs">
                {ra.agent?.name ?? `Agent ${ra.agent?.id}`}
              </span>
            ))}
          </div>
        ) : (
          <span className="text-gray-500 italic">Waiting for players</span>
        )}
      </div>

      {(room.status === 'playing' || room.status === 'finished') && (
        <div className="text-xs text-gray-400">
          Turn: <span className="text-white font-semibold">—</span>
        </div>
      )}

      <div className="flex items-center justify-between mt-auto">
        <span className="text-xs text-gray-500">{formatRelativeTime(room.created_at)}</span>
        <Link
          to={`/rooms/${room.id}`}
          className="text-xs bg-blue-600 hover:bg-blue-500 text-white px-3 py-1 rounded transition-colors"
        >
          Watch →
        </Link>
      </div>
    </div>
  );
}
