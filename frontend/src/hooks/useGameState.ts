import { useQuery } from '@tanstack/react-query';
import { getRoomState } from '../api/client';
import type { GameStateResponse } from '../types';

export function useGameState(roomId: number) {
  return useQuery<GameStateResponse>({
    queryKey: ['roomState', roomId],
    queryFn: () => getRoomState(roomId),
    refetchInterval: 2000,
  });
}
